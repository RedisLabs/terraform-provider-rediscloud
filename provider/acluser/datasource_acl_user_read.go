package acluser

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/users"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read refreshes the Terraform state with the latest data.
func (d *aclUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var config AclUserDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all ACL users from the API
	userList, err := d.client.Client.Users.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read ACL Users",
			fmt.Sprintf("An error occurred while reading ACL users: %s", err.Error()),
		)
		return
	}

	// Build filters based on configuration
	var filters []func(user *users.GetUserResponse) bool

	// Filter by name if specified
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		name := config.Name.ValueString()
		filters = append(filters, func(user *users.GetUserResponse) bool {
			if user == nil {
				return false
			}
			return redis.StringValue(user.Name) == name
		})
	}

	// Apply filters
	userList = filterUsers(userList, filters)

	// Check for exactly one result
	if len(userList) == 0 {
		resp.Diagnostics.AddError(
			"No ACL Users Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(userList) > 1 {
		resp.Diagnostics.AddError(
			"Multiple ACL Users Found",
			"Your query returned more than one result. Please try a more specific search criteria and try again.",
		)
		return
	}

	// Map the result to state
	user := userList[0]
	config.ID = types.StringValue(strconv.Itoa(redis.IntValue(user.ID)))
	config.Name = types.StringValue(redis.StringValue(user.Name))
	config.Role = types.StringValue(redis.StringValue(user.Role))

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

// filterUsers applies all filters to the list of users.
func filterUsers(userList []*users.GetUserResponse, filters []func(user *users.GetUserResponse) bool) []*users.GetUserResponse {
	var filteredUsers []*users.GetUserResponse
	for _, user := range userList {
		if user == nil {
			continue
		}
		if filterUser(user, filters) {
			filteredUsers = append(filteredUsers, user)
		}
	}
	return filteredUsers
}

// filterUser checks if a single user passes all filters.
func filterUser(user *users.GetUserResponse, filters []func(user *users.GetUserResponse) bool) bool {
	for _, f := range filters {
		if !f(user) {
			return false
		}
	}
	return true
}
