package aclrole

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *aclRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var data AclRoleDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := d.client.Client.Roles.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read ACL Roles",
			fmt.Sprintf("An error occurred while reading ACL roles: %s", err.Error()),
		)
		return
	}

	list = filterRolesByName(list, data.Name.ValueString())
	if len(list) == 0 {
		resp.Diagnostics.AddError(
			"No ACL Roles Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(list) > 1 {
		resp.Diagnostics.AddError(
			"Multiple ACL Roles Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	role := list[0]
	rules, diags := RulesFromAPI(ctx, role.RedisRules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(strconv.Itoa(redis.IntValue(role.ID)))
	data.Name = types.StringValue(redis.StringValue(role.Name))
	data.Rule = rules

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func filterRolesByName(list []*roles.GetRoleResponse, name string) []*roles.GetRoleResponse {
	var filtered []*roles.GetRoleResponse
	for _, role := range list {
		if role != nil && redis.StringValue(role.Name) == name {
			filtered = append(filtered, role)
		}
	}
	return filtered
}
