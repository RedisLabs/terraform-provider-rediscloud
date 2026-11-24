package aclrole_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrole"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

func TestAclRoleDataSourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	var response datasource.SchemaResponse
	aclrole.NewAclRoleDataSource().Schema(ctx, datasource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	rules, diags := aclrole.RulesFromAPI(ctx, nil)
	require.False(t, diags.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(aclrole.AclRoleDataSourceModel{Rule: rules})}
	assert.True(t, response.Schema.Type().Equal(modelType), "data-source schema must match its model type")

	rule, ok := response.Schema.Attributes["rule"].(datasourceschema.SetAttribute)
	require.True(t, ok, "rule must be a set attribute")
	assert.True(t, rule.ElementType.Equal(aclrole.NewRuleSetType().ElementType()))

	assert.Empty(t, response.Schema.Blocks, "data-source schema must not declare blocks")
}
