package peering_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/peering"
)

func TestSubscriptionPeeringsDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := peering.NewSubscriptionPeeringsDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	peerings, diags := peering.PeeringsFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(peering.SubscriptionPeeringsDataSourceModel{
		Peerings: peerings,
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")
	assert.Empty(t, response.Schema.Blocks, "computed object collections must be attributes, as SDKv2 served them")

	id := response.Schema.Attributes["id"]
	assert.True(t, id.IsOptional() && id.IsComputed(), "id must stay Optional+Computed like the SDKv2 implicit id")
}
