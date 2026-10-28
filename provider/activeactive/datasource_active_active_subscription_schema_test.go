package activeactive_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestActiveActiveSubscriptionDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := activeactive.NewActiveActiveSubscriptionDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	// Build each collection through its typed constructor or mapper. AttrTypesOf
	// needs an initialized value to preserve the collection's element type.
	maintenance, diags := customtypes.NewMaintenanceList(ctx, nil)
	require.False(t, diags.HasError())
	pricing, diags := utils.PricingFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	resourceTags, diags := utils.ResourceTagsFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	// Derive the expected object type from the model so one comparison validates
	// every schema key, tfsdk tag, scalar type, and collection type.
	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(activeactive.ActiveActiveSubscriptionDataSourceModel{
		MaintenanceWindows: maintenance,
		Pricing:            pricing,
		ResourceTags:       resourceTags,
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")

	// Computed object collections must be attributes: Terraform plans blocks as
	// empty lists when the read is deferred, which breaks [0] references.
	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}
