package transitgateway_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
)

func TestTransitGatewayDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := transitgateway.NewTransitGatewayDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	cidrs, diags := transitgateway.CidrsFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(transitgateway.TransitGatewayDataSourceModel{
		Cidrs: cidrs,
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "data-source schema must match its model type")
	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}
