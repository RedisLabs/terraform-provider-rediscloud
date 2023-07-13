package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/users"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
)

func dataSourceRedisCloudAclUser() *schema.Resource {
	return &schema.Resource{
		Description: "The ACL User is an authenticated entity whose permissions are described by an associated Role.",
		ReadContext: dataSourceRedisCloudAclUserRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "A meaningful name to identify the user",
				Type:        schema.TypeString,
				Required:    true,
			},
			"role": {
				Description: "The Role which this User has.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceRedisCloudAclUserRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	var filters []func(user *users.GetUserResponse) bool
	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(user *users.GetUserResponse) bool {
			return redis.StringValue(user.Name) == v.(string)
		})
	}

	list, err := api.client.Users.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}
	list = filterUsers(list, filters)

	if len(list) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(list) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	user := list[0]
	d.SetId(strconv.Itoa(redis.IntValue(user.ID)))
	if err := d.Set("name", redis.StringValue(user.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("role", redis.StringValue(user.Role)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterUsers(list []*users.GetUserResponse, filters []func(*users.GetUserResponse) bool) []*users.GetUserResponse {
	var filtered []*users.GetUserResponse
	for _, user := range list {
		if filterUser(user, filters) {
			filtered = append(filtered, user)
		}
	}
	return filtered
}

func filterUser(rule *users.GetUserResponse, filters []func(*users.GetUserResponse) bool) bool {
	for _, filter := range filters {
		if !filter(rule) {
			return false
		}
	}
	return true
}
