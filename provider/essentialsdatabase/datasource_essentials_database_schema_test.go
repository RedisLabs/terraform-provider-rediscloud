package essentialsdatabase_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentialsdatabase"
)

func TestEssentialsDatabaseDataSourceSchemaMatchesModel(t *testing.T) {
	dataSource := essentialsdatabase.NewEssentialsDatabaseDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &response)
	assert.False(t, response.Diagnostics.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(essentialsdatabase.EssentialsDatabaseDataSourceModel{
		SourceIPs:             types.SetNull(types.StringType),
		Replica:               types.SetNull(response.Schema.Attributes["replica"].GetType().(types.SetType).ElemType),
		ClientTLSCertificates: types.ListNull(types.StringType),
		Alert:                 types.SetNull(response.Schema.Attributes["alert"].GetType().(types.SetType).ElemType),
		Modules:               types.SetNull(response.Schema.Attributes["modules"].GetType().(types.SetType).ElemType),
		LatestBackupStatus:    types.SetNull(response.Schema.Attributes["latest_backup_status"].GetType().(types.SetType).ElemType),
		LatestImportStatus:    types.SetNull(response.Schema.Attributes["latest_import_status"].GetType().(types.SetType).ElemType),
		RegexRules:            types.ListNull(types.StringType),
		Tags:                  types.MapNull(types.StringType),
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")
}
