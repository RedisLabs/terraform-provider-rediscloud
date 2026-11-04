package psc_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/psc"
)

func TestActiveActivePrivateServiceConnectEndpointResourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	psc.NewActiveActivePrivateServiceConnectEndpointResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics)

	attachments, diags := psc.ServiceAttachmentsFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(psc.ActiveActivePrivateServiceConnectEndpointResourceModel{
		ServiceAttachments: attachments,
		Timeouts:           endpointTimeouts(),
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "resource schema must match its model type")
}
