package pro_test

import (
	"context"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_backups"
	"github.com/RedisLabs/rediscloud-go-api/service/latest_imports"
	"github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
)

func TestProDatabaseDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := pro.NewProDatabaseDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(pro.ProDatabaseDataSourceModel{
		ReplicaOf:          types.SetNull(types.StringType),
		Alert:              types.SetNull(pro.DatabaseAlertElemType),
		Module:             types.SetNull(pro.DatabaseModuleElemType),
		HashingPolicy:      types.ListNull(types.StringType),
		LatestBackupStatus: types.SetNull(pro.LatestBackupStatusElemType),
		LatestImportStatus: types.SetNull(pro.LatestImportStatusElemType),
		Tags:               types.MapNull(types.StringType),
		SourceIPs:          types.SetNull(types.StringType),
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "data-source schema must match its model type")
	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}

func TestDatabaseAlertsAndModulesFromAPI(t *testing.T) {
	ctx := context.Background()

	alerts, diags := pro.DatabaseAlertsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, alerts.IsNull())
	assert.Empty(t, alerts.Elements())

	alerts, diags = pro.DatabaseAlertsFromAPI(ctx, []*databases.Alert{{Name: redis.String("dataset-size"), Value: redis.Int(80)}})
	require.False(t, diags.HasError())
	var alertModels []pro.DatabaseAlertModel
	require.False(t, alerts.ElementsAs(ctx, &alertModels, false).HasError())
	assert.Equal(t, []pro.DatabaseAlertModel{{Name: types.StringValue("dataset-size"), Value: types.Int64Value(80)}}, alertModels)

	modules, diags := pro.DatabaseModulesFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, modules.IsNull())
	assert.Empty(t, modules.Elements())
}

func TestDatabaseTagsFromAPI(t *testing.T) {
	ctx := context.Background()

	empty, diags := pro.DatabaseTagsFromAPI(ctx, &tags.AllTags{})
	require.False(t, diags.HasError())
	assert.False(t, empty.IsNull())
	assert.Empty(t, empty.Elements())

	tagList := []*tags.Tag{{Key: redis.String("env"), Value: redis.String("test")}}
	m, diags := pro.DatabaseTagsFromAPI(ctx, &tags.AllTags{Tags: &tagList})
	require.False(t, diags.HasError())
	assert.Equal(t, types.MapValueMust(types.StringType, map[string]attr.Value{"env": types.StringValue("test")}), m)
}

func TestLatestBackupStatusFromAPI(t *testing.T) {
	ctx := context.Background()

	missing, diags := pro.LatestBackupStatusFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, missing.IsNull())
	assert.Empty(t, missing.Elements())

	noResponse, diags := pro.LatestBackupStatusFromAPI(ctx, &latest_backups.LatestBackupStatus{})
	require.False(t, diags.HasError())
	var models []pro.LatestBackupStatusModel
	require.False(t, noResponse.ElementsAs(ctx, &models, false).HasError())
	require.Len(t, models, 1)
	assert.Empty(t, models[0].Response.Elements())
	assert.Empty(t, models[0].Error.Elements())

	backupTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	full, diags := pro.LatestBackupStatusFromAPI(ctx, &latest_backups.LatestBackupStatus{
		Response: &latest_backups.Response{
			Resource: &latest_backups.Resource{Status: redis.String("success"), LastBackupTime: &backupTime},
			Error:    &latest_backups.Error{Type: redis.String("NOT_FOUND")},
		},
	})
	require.False(t, diags.HasError())
	require.False(t, full.ElementsAs(ctx, &models, false).HasError())
	var responses []pro.LatestBackupResponseModel
	require.False(t, models[0].Response.ElementsAs(ctx, &responses, false).HasError())
	assert.Equal(t, []pro.LatestBackupResponseModel{{
		Status:         types.StringValue("success"),
		LastBackupTime: types.StringValue(backupTime.String()),
		FailureReason:  types.StringValue(""),
	}}, responses)
	var errs []pro.StatusErrorModel
	require.False(t, models[0].Error.ElementsAs(ctx, &errs, false).HasError())
	assert.Equal(t, []pro.StatusErrorModel{{
		Type:        types.StringValue("NOT_FOUND"),
		Description: types.StringValue(""),
		Status:      types.StringValue(""),
	}}, errs)
}

func TestLatestImportStatusFromAPI(t *testing.T) {
	ctx := context.Background()

	status, diags := pro.LatestImportStatusFromAPI(ctx, &latest_imports.LatestImportStatus{
		Response: &latest_imports.Response{
			Resource: &latest_imports.Resource{
				Status:              redis.String("failed"),
				FailureReason:       redis.String("file-corrupted"),
				FailureReasonParams: []*latest_imports.FailureReasonParam{{Key: redis.String("k"), Value: redis.String("v")}},
			},
		},
	})
	require.False(t, diags.HasError())
	var models []pro.LatestImportStatusModel
	require.False(t, status.ElementsAs(ctx, &models, false).HasError())
	require.Len(t, models, 1)
	assert.Empty(t, models[0].Error.Elements())

	var responses []pro.LatestImportResponseModel
	require.False(t, models[0].Response.ElementsAs(ctx, &responses, false).HasError())
	require.Len(t, responses, 1)
	assert.True(t, responses[0].LastImportTime.IsNull())
	assert.Equal(t, types.StringValue("file-corrupted"), responses[0].FailureReason)

	var params []pro.FailureReasonParamModel
	require.False(t, responses[0].FailureReasonParams.ElementsAs(ctx, &params, false).HasError())
	assert.Equal(t, []pro.FailureReasonParamModel{{Key: types.StringValue("k"), Value: types.StringValue("v")}}, params)
}
