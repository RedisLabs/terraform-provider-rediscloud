package pro

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func diagnosticErrors(diags diag.Diagnostics) []string {
	messages := make([]string, 0, diags.ErrorsCount())
	for _, diagnostic := range diags.Errors() {
		messages = append(messages, diagnostic.Summary()+": "+diagnostic.Detail())
	}
	return messages
}

func TestProSubscriptionsDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := NewProSubscriptionsDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	cloudProviders, diags := utils.CloudProvidersFromAPI(ctx, nil)
	require.Empty(t, diagnosticErrors(diags))

	model := ProSubscriptionsDataSourceModel{
		Filters: &proSubscriptionsFiltersModel{
			Name: types.StringNull(),
		},
		Subscriptions: []ProSubscriptionModel{
			{CloudProvider: cloudProviders},
		},
	}
	schemaType, ok := response.Schema.Type().(types.ObjectType)
	require.True(t, ok)
	_, diags = types.ObjectValueFrom(ctx, schemaType.AttrTypes, model)
	require.Empty(t, diagnosticErrors(diags))

	subscriptionType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ProSubscriptionModel{
		CloudProvider: cloudProviders,
	})}

	assert.Empty(t, response.Schema.Attributes)
	assert.Len(t, response.Schema.Blocks, 2)

	filtersBlock, ok := response.Schema.Blocks["filters"].(schema.SingleNestedBlock)
	require.True(t, ok)
	filtersType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(proSubscriptionsFiltersModel{})}
	assert.Equal(t, filtersType, filtersBlock.Type(), "filters schema must match its model type")
	assert.Len(t, filtersBlock.Attributes, 1)
	assert.Empty(t, filtersBlock.Blocks)
	nameAttribute, ok := filtersBlock.Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)
	assert.True(t, nameAttribute.Optional)
	assert.False(t, nameAttribute.Required)
	assert.False(t, nameAttribute.Computed)

	subscriptionsBlock, ok := response.Schema.Blocks["subscriptions"].(schema.ListNestedBlock)
	require.True(t, ok)
	assert.Equal(t, subscriptionType, subscriptionsBlock.NestedObject.Type(), "subscriptions schema must match its model element type")
	assert.Len(t, subscriptionsBlock.NestedObject.Blocks, 1)

	cloudProviderBlock, ok := subscriptionsBlock.NestedObject.Blocks["cloud_provider"].(schema.ListNestedBlock)
	require.True(t, ok)
	cloudProviderType, ok := subscriptionType.AttrTypes["cloud_provider"].(types.ListType)
	require.True(t, ok)
	assert.Equal(t, cloudProviderType.ElementType(), cloudProviderBlock.NestedObject.Type(), "cloud_provider schema must match its model element type")
}
