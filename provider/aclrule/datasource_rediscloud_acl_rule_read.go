package aclrule

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *aclRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var data AclRuleModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var filters []func(rule *redis_rules.GetRedisRuleResponse) bool
	filters = append(filters, func(rule *redis_rules.GetRedisRuleResponse) bool {
		return redis.StringValue(rule.Name) == data.Name.ValueString()
	})

	list, err := d.client.Client.RedisRules.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cloud ACL Rules",
			fmt.Sprintf("An error occurred while reading ACL rules: %s", err.Error()),
		)
		return
	}

	list = filterRules(list, filters)
	if len(list) == 0 {
		resp.Diagnostics.AddError(
			"No ACL Rules Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(list) > 1 {
		resp.Diagnostics.AddError(
			"Multiple ACL Rules Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	rule := list[0]

	data.ID = types.StringValue(strconv.Itoa(redis.IntValue(rule.ID)))
	data.Rule = types.StringValue(redis.StringValue(rule.ACL))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func filterRules(list []*redis_rules.GetRedisRuleResponse, filters []func(*redis_rules.GetRedisRuleResponse) bool) []*redis_rules.GetRedisRuleResponse {
	var filtered []*redis_rules.GetRedisRuleResponse
	for _, rule := range list {
		if filterRule(rule, filters) {
			filtered = append(filtered, rule)
		}
	}
	return filtered
}

func filterRule(rule *redis_rules.GetRedisRuleResponse, filters []func(*redis_rules.GetRedisRuleResponse) bool) bool {
	for _, filter := range filters {
		if !filter(rule) {
			return false
		}
	}
	return true
}
