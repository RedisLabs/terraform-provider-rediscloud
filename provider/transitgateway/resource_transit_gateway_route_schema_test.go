package transitgateway_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
)

func resourceSchema(t *testing.T, r resource.Resource) resource.SchemaResponse {
	t.Helper()
	var response resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())
	return response
}

func timeoutsValue(t *testing.T, response resource.SchemaResponse) timeouts.Value {
	t.Helper()
	timeoutsType, ok := response.Schema.Blocks["timeouts"].Type().(timeouts.Type)
	require.True(t, ok, "timeouts block must use the framework timeouts type")
	return timeouts.Value{Object: types.ObjectNull(timeoutsType.AttributeTypes())}
}

func emptyCidrs(t *testing.T) types.List {
	t.Helper()
	cidrs, diags := transitgateway.CidrsFromAPI(context.Background(), nil)
	require.False(t, diags.HasError())
	return cidrs
}

func TestTransitGatewayRouteResourceSchemaMatchesModel(t *testing.T) {
	response := resourceSchema(t, transitgateway.NewTransitGatewayRouteResource())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(transitgateway.TransitGatewayRouteResourceModel{
		Cidrs:    emptyCidrs(t),
		Timeouts: timeoutsValue(t, response),
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "resource schema must match its model type")
}

func TestActiveActiveTransitGatewayRouteResourceSchemaMatchesModel(t *testing.T) {
	response := resourceSchema(t, transitgateway.NewActiveActiveTransitGatewayRouteResource())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(transitgateway.ActiveActiveTransitGatewayRouteResourceModel{
		Cidrs:    emptyCidrs(t),
		Timeouts: timeoutsValue(t, response),
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "resource schema must match its model type")
}
