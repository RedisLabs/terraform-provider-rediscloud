package aclrole_test

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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrole"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

func TestAclRoleResourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	aclrole.NewAclRoleResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	rules, diags := aclrole.RulesFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(aclrole.AclRoleResourceModel{
		Rule: rules,
		Timeouts: timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		})},
	})}
	assert.True(t, response.Schema.Type().Equal(modelType), "resource schema must match its model type")

	ruleBlock, ok := response.Schema.Blocks["rule"].(schema.SetNestedBlock)
	require.True(t, ok, "rule must be a set nested block")
	assert.True(t, aclrole.NewRuleSetType().ElementType().Equal(ruleBlock.NestedObject.Type()))

	databaseBlock, ok := ruleBlock.NestedObject.Blocks["database"].(schema.SetNestedBlock)
	require.True(t, ok, "database must be a set nested block")
	assert.True(t, aclrole.NewRuleDatabaseSetType().ElementType().Equal(databaseBlock.NestedObject.Type()))
}
