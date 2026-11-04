package privatelink_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/privatelink"
)

func TestPrivateLinkResourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	privatelink.NewPrivateLinkResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)

	connections, diags := privatelink.ConnectionsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	databases, diags := privatelink.DatabasesFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(privatelink.PrivateLinkResourceModel{
		Principal:   privatelink.NullPrincipalSet(),
		Connections: connections,
		Databases:   databases,
		Timeouts:    timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{"create": types.StringType, "read": types.StringType, "update": types.StringType, "delete": types.StringType})},
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "resource schema must match its model type")

	principal, ok := response.Schema.Blocks["principal"].(schema.SetNestedBlock)
	require.True(t, ok)
	assert.Equal(t, privatelink.NewPrincipalSetType().ElementType(), principal.NestedObject.Type())
}
