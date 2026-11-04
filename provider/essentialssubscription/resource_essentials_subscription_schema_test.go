package essentialssubscription_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentialssubscription"
)

func TestEssentialsSubscriptionResourceSchemaMatchesModel(t *testing.T) {
	res := essentialssubscription.NewEssentialsSubscriptionResource()
	var response resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &response)
	assert.False(t, response.Diagnostics.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(essentialssubscription.EssentialsSubscriptionResourceModel{
		Timeouts: timeouts.Value{Object: types.ObjectNull(response.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)},
	})}
	assert.Equal(t, modelType, response.Schema.Type(), "resource schema must match its model type")
}
