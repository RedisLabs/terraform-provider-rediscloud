package peering_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/peering"
)

func TestActiveActiveSubscriptionPeeringResourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	peering.NewActiveActiveSubscriptionPeeringResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(peering.ActiveActiveSubscriptionPeeringResourceModel{
		VPCCidrs: types.SetNull(types.StringType),
		Timeouts: timeouts.Value{Object: types.ObjectNull(timeoutsAttrTypes())},
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "resource schema must match its model type")
}
