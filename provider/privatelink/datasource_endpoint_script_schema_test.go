package privatelink_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/privatelink"
)

func TestEndpointScriptDataSourceSchemaMatchesModel(t *testing.T) {
	dataSource := privatelink.NewEndpointScriptDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &response)

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(privatelink.EndpointScriptModel{})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")
}
