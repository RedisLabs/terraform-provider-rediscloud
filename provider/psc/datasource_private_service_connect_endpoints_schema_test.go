package psc_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/psc"
)

func TestPrivateServiceConnectEndpointsDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response datasource.SchemaResponse
	psc.NewPrivateServiceConnectEndpointsDataSource().Schema(ctx, datasource.SchemaRequest{}, &response)

	endpoints, diags := psc.EndpointsFromAPI(ctx, nil, nil)
	require.False(t, diags.HasError())
	attachments, diags := psc.ServiceAttachmentsFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(psc.PrivateServiceConnectEndpointsModel{Endpoints: endpoints})}
	assert.Equal(t, modelType, response.Schema.Type(), "data-source schema must match its model type")

	endpointsAttribute, ok := response.Schema.Attributes["endpoints"].(schema.ListAttribute)
	require.True(t, ok)
	endpointType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(psc.EndpointModel{ServiceAttachments: attachments})}
	assert.Equal(t, endpointType, endpointsAttribute.ElementType, "endpoint element type must match its model type")
}
