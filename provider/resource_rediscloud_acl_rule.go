package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

func resourceRedisCloudAclRule() *schema.Resource {
	return &schema.Resource{
		Description:   "Create an ACL Rule within your Redis Enterprise Cloud Account",
		CreateContext: resourceRedisCloudAclRuleCreate,
		ReadContext:   resourceRedisCloudAclRuleRead,
		UpdateContext: resourceRedisCloudAclRuleUpdate,
		DeleteContext: resourceRedisCloudAclRuleDelete,

		Importer: &schema.ResourceImporter{
			// Let the READ operation do the heavy lifting for importing values from the API.
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(3 * time.Minute),
			Read:   schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(3 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the rule. Must be unique.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"rule": {
				Description: "The Rule itself, must comply with Redis' ACL syntax",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func resourceRedisCloudAclRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	name := d.Get("name").(string)
	rule := d.Get("rule").(string)

	createRule := redis_rules.CreateRedisRuleRequest{
		Name:      redis.String(name),
		RedisRule: redis.String(rule),
	}

	id, err := api.client.RedisRules.Create(ctx, createRule)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	return diags
}

func resourceRedisCloudAclRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	rule, err := api.client.RedisRules.Get(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(rule.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("rule", redis.StringValue(rule.ACL)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudAclRuleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("name", "rule") {
		updateRedisRuleRequest := redis_rules.CreateRedisRuleRequest{}

		name := d.Get("name").(string)
		updateRedisRuleRequest.Name = &name
		rule := d.Get("rule").(string)
		updateRedisRuleRequest.RedisRule = &rule

		err = api.client.RedisRules.Update(ctx, id, updateRedisRuleRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceRedisCloudAclRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.client.RedisRules.Delete(ctx, id)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
