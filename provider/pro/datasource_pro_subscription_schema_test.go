package pro_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestProSubscriptionDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := pro.NewProSubscriptionDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	cloudProviders, diags := utils.CloudProvidersFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	maintenance, diags := customtypes.NewMaintenanceList(ctx, nil)
	require.False(t, diags.HasError())
	pricing, diags := utils.PricingFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(pro.ProSubscriptionDataSourceModel{
		CloudProvider:      cloudProviders,
		MaintenanceWindows: maintenance,
		Pricing:            pricing,
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "data-source schema must match its model type")

	// Computed object collections must be attributes: Terraform plans blocks as
	// empty lists when the read is deferred, which breaks [0] references.
	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}
