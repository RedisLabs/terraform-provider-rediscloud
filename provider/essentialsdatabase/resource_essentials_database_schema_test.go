package essentialsdatabase_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentialsdatabase"
)

func TestEssentialsDatabaseResourceSchemaMatchesModel(t *testing.T) {
	res := essentialsdatabase.NewEssentialsDatabaseResource()
	var response resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &response)
	assert.False(t, response.Diagnostics.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(essentialsdatabase.EssentialsDatabaseResourceModel{
		SourceIPs:             types.ListNull(types.StringType),
		Replica:               types.ListNull(response.Schema.Blocks["replica"].Type().(types.ListType).ElemType),
		ClientTLSCertificates: types.ListNull(types.StringType),
		Alert:                 types.SetNull(response.Schema.Blocks["alert"].Type().(types.SetType).ElemType),
		Modules:               types.SetNull(response.Schema.Attributes["modules"].GetType().(types.SetType).ElemType),
		RegexRules:            types.ListNull(types.StringType),
		Tags:                  types.MapNull(types.StringType),
		Timeouts:              timeouts.Value{Object: types.ObjectNull(response.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)},
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "resource schema must match its model type")
}
