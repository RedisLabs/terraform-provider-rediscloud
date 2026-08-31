package activeactive_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	pricing, diags := utils.PricingListFromAPI(ctx, nil)
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

	// A custom block reports its custom list type instead of independently
	// exposing the shape declared in NestedObject. Compare the element types at
	// each custom-list level to validate the maintenance and window schemas too.
	maintenanceBlock, ok := response.Schema.Blocks["maintenance_windows"].(schema.ListNestedBlock)
	require.True(t, ok)
	maintenanceType := customtypes.NewMaintenanceListType()
	assert.Equal(t, maintenanceType.ElementType(), maintenanceBlock.NestedObject.Type(), "maintenance_windows schema must match its model element type")

	windowBlock, ok := maintenanceBlock.NestedObject.Blocks["window"].(schema.ListNestedBlock)
	require.True(t, ok)
	windowType := customtypes.NewMaintenanceWindowListType()
	assert.Equal(t, windowType.ElementType(), windowBlock.NestedObject.Type(), "window schema must match its model element type")
}
