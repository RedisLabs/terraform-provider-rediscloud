package aclrule_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrule"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

func TestAclRuleResourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	aclrule.NewAclRuleResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(aclrule.AclRuleResourceModel{
		Timeouts: timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		})},
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "resource schema must match its model type")
}
