package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
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
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(3 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the rule, must be unique",
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
	api := meta.(*client.ApiClient)

	name := d.Get("name").(string)
	rule := d.Get("rule").(string)

	createRule := redis_rules.CreateRedisRuleRequest{
		Name:      redis.String(name),
		RedisRule: redis.String(rule),
	}

	id, err := api.Client.RedisRules.Create(ctx, createRule)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	err = waitForAclRuleToBeActive(ctx, id, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudAclRuleRead(ctx, d, meta)
}

func resourceRedisCloudAclRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	rule, err := api.Client.RedisRules.Get(ctx, id)
	if err != nil {
		if _, ok := err.(*redis_rules.NotFound); ok {
			d.SetId("")
			return diags
		}
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
	api := meta.(*client.ApiClient)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("name", "rule") {
		updateRedisRuleRequest := redis_rules.CreateRedisRuleRequest{}

		if d.HasChange("name") {
			name := d.Get("name").(string)
			updateRedisRuleRequest.Name = &name
		}

		rule := d.Get("rule").(string)
		updateRedisRuleRequest.RedisRule = &rule

		err = api.Client.RedisRules.Update(ctx, id, updateRedisRuleRequest)
		if err != nil {
			return diag.FromErr(err)
		}

		err = waitForAclRuleToBeActive(ctx, id, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudAclRuleRead(ctx, d, meta)
}

func resourceRedisCloudAclRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.Client.RedisRules.Delete(ctx, id)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	// Wait until it's really disappeared
	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		rule, err := api.Client.RedisRules.Get(ctx, id)

		if err != nil {
			if _, ok := err.(*redis_rules.NotFound); ok {
				// All good, the resource is gone
				return nil
			}
			// This was an unexpected error
			return retry.NonRetryableError(fmt.Errorf("error getting rule: %s", err))
		}

		if rule != nil {
			return retry.RetryableError(fmt.Errorf("expected rule %d to be deleted but was in state %s", id, redis.StringValue(rule.Status)))
		}
		// Unclear at this point what's going on!
		return retry.NonRetryableError(fmt.Errorf("unexpected error getting rule"))
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func waitForAclRuleToBeActive(ctx context.Context, id int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   5 * time.Second,
		Pending: []string{redis_rules.StatusPending},
		Target:  []string{redis_rules.StatusActive},
		Timeout: 5 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for rule %d to be active", id)

			rule, err := api.Client.RedisRules.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(rule.Status), redis.StringValue(rule.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
