package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceRedisCloudAclRole() *schema.Resource {
	return &schema.Resource{
		Description: "The ACL Role grants a number of permissions to databases",
		ReadContext: dataSourceRedisCloudAclRoleRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the user",
				Type:        schema.TypeString,
				Required:    true,
			},
			"rule": {
				Description: "This Role's permissions and the databases to which they apply",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The name of the Rule",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"database": {
							Description: "The databases to which this Rule applies",
							Type:        schema.TypeSet,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"subscription": {
										Description: "The name of the Rule",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"database": {
										Description: "The databases to which this Rule applies",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"regions": {
										Description: "The regional deployments of this database to which the Rule applies. Only relevant to Active/Active databases, otherwise omit",
										Type:        schema.TypeSet,
										Computed:    true,
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

func dataSourceRedisCloudAclRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	var filters []func(role *roles.GetRoleResponse) bool
	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(role *roles.GetRoleResponse) bool {
			return redis.StringValue(role.Name) == v.(string)
		})
	}

	list, err := api.client.Roles.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	list = filterRoles(list, filters)

	if len(list) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(list) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	role := list[0]
	d.SetId(strconv.Itoa(redis.IntValue(role.ID)))
	if err := d.Set("name", redis.StringValue(role.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("rule", flattenRules(role.RedisRules)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterRoles(list []*roles.GetRoleResponse, filters []func(*roles.GetRoleResponse) bool) []*roles.GetRoleResponse {
	var filtered []*roles.GetRoleResponse
	for _, role := range list {
		if filterRole(role, filters) {
			filtered = append(filtered, role)
		}
	}
	return filtered
}

func filterRole(rule *roles.GetRoleResponse, filters []func(*roles.GetRoleResponse) bool) bool {
	for _, filter := range filters {
		if !filter(rule) {
			return false
		}
	}
	return true
}
