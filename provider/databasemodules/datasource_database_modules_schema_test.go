package databasemodules_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/databasemodules"
)

func TestDatabaseModulesDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := databasemodules.NewDatabaseModulesDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	modules, diags := databasemodules.ModulesFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(databasemodules.DatabaseModulesDataSourceModel{
		Modules: modules,
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "data-source schema must match its model type")
	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}

func TestModulesFromAPI(t *testing.T) {
	ctx := context.Background()

	empty, diags := databasemodules.ModulesFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, empty.IsNull())
	assert.Empty(t, empty.Elements())

	set, diags := databasemodules.ModulesFromAPI(ctx, []*account.DatabaseModule{
		{Name: redis.String("RedisJSON"), Description: redis.String("JSON")},
		nil,
		{Name: redis.String("RediSearch")},
	})
	require.False(t, diags.HasError())

	var models []databasemodules.ModuleModel
	require.False(t, set.ElementsAs(ctx, &models, false).HasError())
	assert.ElementsMatch(t, []databasemodules.ModuleModel{
		{Name: types.StringValue("RedisJSON"), Description: types.StringValue("JSON")},
		{Name: types.StringValue("RediSearch"), Description: types.StringValue("")},
	}, models)
}
