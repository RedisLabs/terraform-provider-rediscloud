package transitgateway_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
)

func TestTransitGatewayAttachmentResourceSchemaMatchesModel(t *testing.T) {
	response := resourceSchema(t, transitgateway.NewTransitGatewayAttachmentResource())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(transitgateway.TransitGatewayAttachmentResourceModel{
		Cidrs:    emptyCidrs(t),
		Timeouts: timeoutsValue(t, response),
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "resource schema must match its model type")
}

func TestActiveActiveTransitGatewayAttachmentResourceSchemaMatchesModel(t *testing.T) {
	response := resourceSchema(t, transitgateway.NewActiveActiveTransitGatewayAttachmentResource())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(transitgateway.ActiveActiveTransitGatewayAttachmentResourceModel{
		Cidrs:    emptyCidrs(t),
		Timeouts: timeoutsValue(t, response),
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "resource schema must match its model type")
}
