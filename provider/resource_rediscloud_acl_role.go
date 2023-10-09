package provider

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRedisCloudAclRole() *schema.Resource {
	return &schema.Resource{
		Description:   "Create an ACL Role within your Redis Enterprise Cloud Account",
		CreateContext: resourceRedisCloudAclRoleCreate,
		ReadContext:   resourceRedisCloudAclRoleRead,
		UpdateContext: resourceRedisCloudAclRoleUpdate,
		DeleteContext: resourceRedisCloudAclRoleDelete,

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
				Description: "A meaningful name to identify the role, must be unique",
				Type:        schema.TypeString,
				Required:    true,
			},
			"rule": {
				Description: "A set of rules which apply to the role",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The name of the rule",
							Type:        schema.TypeString,
							Required:    true,
						},
						"database": {
							Description: "A set of databases to whom this rule applies within the role",
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"subscription": {
										Description: "The subscription (id) to which the database belongs",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"database": {
										Description: "The database (id)",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"regions": {
										Description: "For ActiveActive databases only",
										Type:        schema.TypeSet,
										Optional:    true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudAclRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	name := d.Get("name").(string)
	associateWithRules := extractRules(d)

	createRoleRequest := roles.CreateRoleRequest{
		Name:       redis.String(name),
		RedisRules: associateWithRules,
	}

	id, err := api.client.Roles.Create(ctx, createRoleRequest)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	err = waitForAclRoleToBeActive(ctx, id, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudAclRoleRead(ctx, d, meta)
}

func resourceRedisCloudAclRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := api.client.Roles.Get(ctx, id)
	if err != nil {
		if _, ok := err.(*roles.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(role.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("rule", flattenRules(role.RedisRules)); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceRedisCloudAclRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("name", "rule") {
		updateRoleRequest := roles.CreateRoleRequest{}

		name := d.Get("name").(string)
		updateRoleRequest.Name = &name
		rules := extractRules(d)
		updateRoleRequest.RedisRules = rules

		waitForRulesDbToBeActive(ctx, d, api)

		err = api.client.Roles.Update(ctx, id, updateRoleRequest)
		if err != nil {
			return diag.FromErr(err)
		}

		err = waitForAclRoleToBeActive(ctx, id, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudAclRoleRead(ctx, d, meta)
}

func resourceRedisCloudAclRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	waitForRulesDbToBeActive(ctx, d, api)

	// Every now and then the status of the role changes back to the 'pending' state after being fully created.
	// The DELETE request will fail if the role is pending.
	// This block queries the endpoint until the role is no longer in the 'pending' state.
	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		var assignedRulesIds []int
		role, err := api.client.Roles.Get(ctx, id)
		if err != nil {
			// This was an unexpected error
			return retry.NonRetryableError(fmt.Errorf("error getting role: %s", err))
		}

		// Check if the rules are not in the pending state
		for _, redisRule := range role.RedisRules {
			assignedRulesIds = append(assignedRulesIds, *redisRule.RuleId)
		}
		rules, err := api.client.RedisRules.List(ctx)
		if err != nil {
			// This was an unexpected error
			return retry.NonRetryableError(fmt.Errorf("error getting role: %s", err))
		}
		for _, rule := range rules {
			for _, asassignedRuleId := range assignedRulesIds {
				ruleId := *rule.ID
				if ruleId == asassignedRuleId {
					ruleStatus := *rule.Status
					if ruleStatus != redis_rules.StatusActive {
						return retry.RetryableError(fmt.Errorf("can't delete the role %d if the rule (%d) is in %s", id, ruleId, ruleStatus))
					}
				}
			}
		}

		status := redis.StringValue(role.Status)
		if status != roles.StatusPending {
			// The role is ready for deletion
			return nil
		} else {
			return retry.RetryableError(fmt.Errorf("can't delete the role %d in state %s", id, status))
		}
	})

	if err != nil {
		return diag.FromErr(err)
	}

	err = api.client.Roles.Delete(ctx, id)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	// Wait until it's really disappeared
	err = retry.RetryContext(ctx, 10*time.Minute, func() *retry.RetryError {
		role, err := api.client.Roles.Get(ctx, id)

		if err != nil {
			if _, ok := err.(*roles.NotFound); ok {
				// All good, the resource is gone
				return nil
			}
			// This was an unexpected error
			return retry.NonRetryableError(fmt.Errorf("error getting role: %s", err))
		}

		if role != nil {
			return retry.RetryableError(fmt.Errorf("expected role %d to be deleted but was in state %s", id, redis.StringValue(role.Status)))
		}
		// Unclear at this point what's going on!
		return retry.NonRetryableError(fmt.Errorf("unexpected error getting role"))
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func extractRules(d *schema.ResourceData) []*roles.CreateRuleInRoleRequest {
	associateWithRules := make([]*roles.CreateRuleInRoleRequest, 0)
	rules := d.Get("rule").(*schema.Set).List()
	for _, rule := range rules {
		ruleMap := rule.(map[string]interface{})

		ruleName := ruleMap["name"].(string)
		associateWithDatabases := make([]*roles.CreateDatabaseInRuleInRoleRequest, 0)

		databases := ruleMap["database"].(*schema.Set).List()
		for _, database := range databases {
			databaseMap := database.(map[string]interface{})

			subscriptionId := databaseMap["subscription"].(int)
			databaseId := databaseMap["database"].(int)

			var regions []*string = nil
			if databaseMap["regions"] != nil {
				regions = setToStringSlice(databaseMap["regions"].(*schema.Set))
			}

			createDatabaseAssociation := roles.CreateDatabaseInRuleInRoleRequest{
				SubscriptionId: redis.Int(subscriptionId),
				DatabaseId:     redis.Int(databaseId),
				Regions:        regions,
			}

			associateWithDatabases = append(associateWithDatabases, &createDatabaseAssociation)
		}

		createRuleAssociation := roles.CreateRuleInRoleRequest{
			RuleName:  redis.String(ruleName),
			Databases: associateWithDatabases,
		}

		associateWithRules = append(associateWithRules, &createRuleAssociation)
	}

	return associateWithRules
}

func flattenRules(rules []*roles.GetRuleInRoleResponse) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, rule := range rules {
		tf := map[string]interface{}{
			"name":     redis.StringValue(rule.RuleName),
			"database": flattenDatabases(rule.Databases),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func flattenDatabases(databases []*roles.GetDatabaseInRuleInRoleResponse) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, database := range databases {
		tf := map[string]interface{}{
			"subscription": redis.IntValue(database.SubscriptionId),
			"database":     redis.IntValue(database.DatabaseId),
			"regions":      redis.StringSliceValue(database.Regions...),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func waitForAclRoleToBeActive(ctx context.Context, id int, api *apiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   5 * time.Second,
		Pending: []string{roles.StatusPending},
		Target:  []string{roles.StatusActive},
		Timeout: 5 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for role %d to be active", id)

			role, err := api.client.Roles.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(role.Status), redis.StringValue(role.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForRulesDbToBeActive(ctx context.Context, d *schema.ResourceData, api *apiClient) error {
	// wait for the assosiacted dbs to be in the active state
	extractedRules := extractRules(d)
	for _, er := range extractedRules {
		for _, db := range er.Databases {
			err := waitForDatabaseToBeActive(ctx, *db.SubscriptionId, *db.DatabaseId, api)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
