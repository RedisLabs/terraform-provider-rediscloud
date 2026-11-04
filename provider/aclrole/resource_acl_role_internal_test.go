package aclrole

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRuleSet(t *testing.T, regions types.Set) RuleSetValue {
	t.Helper()
	ctx := context.Background()

	databases, diags := types.SetValueFrom(ctx, ruleDatabaseElemType, []RuleDatabaseModel{
		{Subscription: types.Int64Value(1), Database: types.Int64Value(2), Regions: regions},
	})
	require.False(t, diags.HasError())

	rules, diags := types.SetValueFrom(ctx, ruleElemType, []RuleModel{
		{Name: types.StringValue("Read-Only"), Database: RuleDatabaseSetValue{SetValue: databases}},
	})
	require.False(t, diags.HasError())

	return RuleSetValue{SetValue: rules}
}

func TestRulesToAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("null regions are omitted", func(t *testing.T) {
		requests, diags := rulesToAPI(ctx, testRuleSet(t, types.SetNull(types.StringType)))
		require.False(t, diags.HasError())
		require.Len(t, requests, 1)
		assert.Equal(t, "Read-Only", redis.StringValue(requests[0].RuleName))
		require.Len(t, requests[0].Databases, 1)
		assert.Equal(t, 1, redis.IntValue(requests[0].Databases[0].SubscriptionId))
		assert.Equal(t, 2, redis.IntValue(requests[0].Databases[0].DatabaseId))
		assert.Nil(t, requests[0].Databases[0].Regions)
	})

	t.Run("regions are sent", func(t *testing.T) {
		regions := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("us-east-1")})
		requests, diags := rulesToAPI(ctx, testRuleSet(t, regions))
		require.False(t, diags.HasError())
		assert.Equal(t, []string{"us-east-1"}, redis.StringSliceValue(requests[0].Databases[0].Regions...))
	})

	t.Run("null rules yield an empty request list", func(t *testing.T) {
		requests, diags := rulesToAPI(ctx, nullRuleSetValue())
		require.False(t, diags.HasError())
		assert.NotNil(t, requests)
		assert.Empty(t, requests)
	})
}

func TestReadKeepsConfiguredRegionsShape(t *testing.T) {
	ctx := context.Background()
	apiRules := []*roles.GetRuleInRoleResponse{
		{
			RuleName:  redis.String("Read-Only"),
			Databases: []*roles.GetDatabaseInRuleInRoleResponse{{SubscriptionId: redis.Int(1), DatabaseId: redis.Int(2)}},
		},
	}

	for name, prior := range map[string]RuleSetValue{
		"omitted regions stay null":  testRuleSet(t, types.SetNull(types.StringType)),
		"empty regions stay empty":   testRuleSet(t, types.SetValueMust(types.StringType, []attr.Value{})),
		"no prior state yields null": nullRuleSetValue(),
	} {
		t.Run(name, func(t *testing.T) {
			emptyRegions, diags := emptyRegionsFrom(ctx, prior)
			require.False(t, diags.HasError())

			rules, diags := rulesFromAPI(ctx, apiRules, emptyRegions)
			require.False(t, diags.HasError())

			if prior.IsNull() {
				prior = testRuleSet(t, types.SetNull(types.StringType))
			}
			assert.True(t, rules.Equal(prior), "got %s, want %s", rules, prior)
		})
	}
}
