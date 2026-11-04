package privatelink_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/privatelink"
)

func TestPrivateLinkDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	dataSource := privatelink.NewPrivateLinkDataSource()
	var response datasource.SchemaResponse
	dataSource.Schema(ctx, datasource.SchemaRequest{}, &response)

	connections, diags := privatelink.ConnectionsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	databases, diags := privatelink.DatabasesFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(privatelink.PrivateLinkDataSourceModel{
		Principals:  privatelink.NullPrincipalSet(),
		Connections: connections,
		Databases:   databases,
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")

	principals, ok := response.Schema.Attributes["principals"].(schema.SetAttribute)
	require.True(t, ok)
	assert.Equal(t, privatelink.NewPrincipalSetType().ElementType(), principals.ElementType)
}
