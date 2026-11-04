package aclrole_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrole"
)

func TestRulesFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("nil rules yield a known empty set", func(t *testing.T) {
		rules, diags := aclrole.RulesFromAPI(ctx, nil)

		require.False(t, diags.HasError())
		assert.False(t, rules.IsNull())
		assert.False(t, rules.IsUnknown())
		assert.Empty(t, rules.Elements())
	})

	t.Run("rules map databases and regions", func(t *testing.T) {
		rules, diags := aclrole.RulesFromAPI(ctx, []*roles.GetRuleInRoleResponse{
			{
				RuleName: redis.String("Read-Only"),
				Databases: []*roles.GetDatabaseInRuleInRoleResponse{
					{SubscriptionId: redis.Int(10), DatabaseId: redis.Int(20)},
					{SubscriptionId: redis.Int(11), DatabaseId: redis.Int(21), Regions: redis.StringSlice("us-east-1", "eu-west-1")},
				},
			},
			nil,
		})
		require.False(t, diags.HasError())

		models, diags := rules.AsModels(ctx)
		require.False(t, diags.HasError())
		require.Len(t, models, 1)
		assert.Equal(t, "Read-Only", models[0].Name.ValueString())

		databases, diags := models[0].Database.AsModels(ctx)
		require.False(t, diags.HasError())
		require.Len(t, databases, 2)

		bySubscription := map[int64]aclrole.RuleDatabaseModel{}
		for _, database := range databases {
			bySubscription[database.Subscription.ValueInt64()] = database
		}

		withoutRegions := bySubscription[10]
		assert.EqualValues(t, 20, withoutRegions.Database.ValueInt64())
		assert.False(t, withoutRegions.Regions.IsNull(), "SDKv2 wrote an empty regions set, not null")
		assert.Empty(t, withoutRegions.Regions.Elements())

		withRegions := bySubscription[11]
		var regions []string
		require.False(t, withRegions.Regions.ElementsAs(ctx, &regions, false).HasError())
		assert.ElementsMatch(t, []string{"us-east-1", "eu-west-1"}, regions)
	})
}

func TestRuleSetTypes(t *testing.T) {
	ctx := context.Background()

	assert.True(t, aclrole.NewRuleSetType().Equal(aclrole.NewRuleSetType()))
	assert.False(t, aclrole.NewRuleSetType().Equal(types.SetType{ElemType: aclrole.NewRuleSetType().ElemType}))
	assert.True(t, aclrole.NewRuleDatabaseSetType().Equal(aclrole.NewRuleDatabaseSetType()))
	assert.False(t, aclrole.NewRuleDatabaseSetType().Equal(types.SetType{ElemType: aclrole.NewRuleDatabaseSetType().ElemType}))

	for name, value := range map[string]tftypes.Value{
		"null":    tftypes.NewValue(aclrole.NewRuleSetType().TerraformType(ctx), nil),
		"unknown": tftypes.NewValue(aclrole.NewRuleSetType().TerraformType(ctx), tftypes.UnknownValue),
	} {
		t.Run(name, func(t *testing.T) {
			converted, err := aclrole.NewRuleSetType().ValueFromTerraform(ctx, value)
			require.NoError(t, err)
			ruleSet, ok := converted.(aclrole.RuleSetValue)
			require.True(t, ok)
			assert.Equal(t, name == "null", ruleSet.IsNull())
			assert.Equal(t, name == "unknown", ruleSet.IsUnknown())
		})
	}

	t.Run("round trip preserves nested custom types", func(t *testing.T) {
		rules, diags := aclrole.RulesFromAPI(ctx, []*roles.GetRuleInRoleResponse{
			{
				RuleName:  redis.String("Full-Access"),
				Databases: []*roles.GetDatabaseInRuleInRoleResponse{{SubscriptionId: redis.Int(1), DatabaseId: redis.Int(2)}},
			},
		})
		require.False(t, diags.HasError())

		terraformValue, err := rules.ToTerraformValue(ctx)
		require.NoError(t, err)
		converted, err := aclrole.NewRuleSetType().ValueFromTerraform(ctx, terraformValue)
		require.NoError(t, err)
		assert.True(t, converted.Equal(rules))

		models, diags := converted.(aclrole.RuleSetValue).AsModels(ctx)
		require.False(t, diags.HasError())
		require.Len(t, models, 1)
		assert.IsType(t, aclrole.RuleDatabaseSetValue{}, models[0].Database)
	})
}
