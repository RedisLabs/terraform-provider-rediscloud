package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceRedisCloudAclRule() *schema.Resource {
	return &schema.Resource{
		Description: "The ACL Rule (known also as RedisRule) allows fine-grained permissions to be assigned to a subset of ACL Users",
		ReadContext: dataSourceRedisCloudAclRuleRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the rule",
				Type:        schema.TypeString,
				Required:    true,
			},
			"rule": {
				Description: "The Rule itself, must comply with Redis' ACL syntax",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceRedisCloudAclRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	var filters []func(rule *redis_rules.GetRedisRuleResponse) bool
	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(rule *redis_rules.GetRedisRuleResponse) bool {
			return redis.StringValue(rule.Name) == v.(string)
		})
	}

	list, err := api.client.RedisRules.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	list = filterRules(list, filters)

	if len(list) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(list) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	rule := list[0]
	d.SetId(strconv.Itoa(redis.IntValue(rule.ID)))
	if err := d.Set("name", redis.StringValue(rule.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("rule", redis.StringValue(rule.ACL)); err != nil {
		return diag.FromErr(err)
	}

	return diags
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
