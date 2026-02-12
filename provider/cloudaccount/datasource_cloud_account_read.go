package cloudaccount

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read refreshes the Terraform state with the latest data.
func (d *cloudAccountDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}
	var state CloudAccountDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all cloud accounts from the API
	cloudAccounts, err := d.client.Client.CloudAccount.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cloud Accounts",
			fmt.Sprintf("An error occurred while reading cloud accounts: %s", err.Error()),
		)
		return
	}

	// Build filters based on configuration
	var filters []func(cloudAccount *cloud_accounts.CloudAccount) bool

	// Handle exclude_internal_account filter - default to false if not set
	if !state.ExcludeInternalAccount.IsNull() && state.ExcludeInternalAccount.ValueBool() {
		filters = append(filters, func(cloudAccount *cloud_accounts.CloudAccount) bool {
			return redis.IntValue(cloudAccount.ID) != 1
		})
	}
	if !state.ProviderType.IsNull() && state.ProviderType.ValueString() != "" {
		filters = append(filters, func(cloudAccount *cloud_accounts.CloudAccount) bool {
			return redis.StringValue(cloudAccount.Provider) == state.ProviderType.ValueString()
		})
	}
	if !state.Name.IsNull() && state.Name.ValueString() != "" {
		filters = append(filters, func(cloudAccount *cloud_accounts.CloudAccount) bool {
			return redis.StringValue(cloudAccount.Name) == state.Name.ValueString()
		})
	}

	// Apply filters
	cloudAccounts = filterCloudAccounts(cloudAccounts, filters)

	// Check for exactly one result
	if len(cloudAccounts) == 0 {
		resp.Diagnostics.AddError(
			"No Cloud Accounts Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(cloudAccounts) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Cloud Accounts Found",
			"Your query returned more than one result. Please try a more specific search criteria and try again.",
		)
		return
	}

	// Map the result to state
	cloudAccount := cloudAccounts[0]
	state.ID = types.StringValue(strconv.Itoa(redis.IntValue(cloudAccount.ID)))
	state.Name = types.StringValue(redis.StringValue(cloudAccount.Name))
	state.AccessKeyID = types.StringValue(redis.StringValue(cloudAccount.AccessKeyID))
	state.ExcludeInternalAccount = types.BoolValue(state.ExcludeInternalAccount.ValueBool())
	state.ProviderType = types.StringValue(redis.StringValue(cloudAccount.Provider))

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func filterCloudAccounts(accounts []*cloud_accounts.CloudAccount, filters []func(account *cloud_accounts.CloudAccount) bool) []*cloud_accounts.CloudAccount {
	var filtered []*cloud_accounts.CloudAccount
	for _, cloudAccount := range accounts {
		if cloudAccount == nil {
			continue
		}
		if filterCloudAccount(cloudAccount, filters) {
			filtered = append(filtered, cloudAccount)
		}
	}

	return filtered
}

func filterCloudAccount(account *cloud_accounts.CloudAccount, filters []func(account *cloud_accounts.CloudAccount) bool) bool {
	for _, f := range filters {
		if !f(account) {
			return false
		}
	}
	return true
}
