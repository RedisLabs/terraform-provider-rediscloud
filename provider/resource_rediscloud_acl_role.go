package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"time"
)

func resourceRedisCloudAclRole() *schema.Resource {
	return &schema.Resource{
		Description:   "Create an ACL Role within your Redis Enterprise Cloud Account",
		CreateContext: resourceRedisCloudAclRoleCreate,
		ReadContext:   resourceRedisCloudAclRoleRead,
		UpdateContext: resourceRedisCloudAclRoleUpdate,
		DeleteContext: resourceRedisCloudAclRoleDelete,

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
				Description: "A meaningful name to identify the role",
				Type:        schema.TypeString,
				Required:    true,
			},
			"rules": {
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
						"databases": {
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
	var diags diag.Diagnostics

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

	return diags
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
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(role.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("rules", flattenRules(role.RedisRules)); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceRedisCloudAclRoleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChanges("name", "rules") {
		updateRoleRequest := roles.CreateRoleRequest{}

		name := d.Get("name").(string)
		updateRoleRequest.Name = &name
		rules := extractRules(d)
		updateRoleRequest.RedisRules = rules

		err = api.client.Roles.Update(ctx, id, updateRoleRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceRedisCloudAclRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = api.client.Roles.Delete(ctx, id)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func extractRules(d *schema.ResourceData) []*roles.CreateRuleInRoleRequest {
	associateWithRules := make([]*roles.CreateRuleInRoleRequest, 0)
	rules := d.Get("rules").(*schema.Set).List()
	for _, rule := range rules {
		ruleMap := rule.(map[string]interface{})

		ruleName := ruleMap["name"].(string)
		associateWithDatabases := make([]*roles.CreateDatabaseInRuleInRoleRequest, 0)

		databases := ruleMap["databases"].(*schema.Set).List()
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
			"name":      redis.StringValue(rule.RuleName),
			"databases": flattenDatabases(rule.Databases),
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
