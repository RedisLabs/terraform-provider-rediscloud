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

type caFilterFunc func(account *cloud_accounts.CloudAccount) bool

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
	var config CloudAccountDataSourceModel
	diags := req.Config.Get(ctx, &config)
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
	var filters []caFilterFunc

	// Handle exclude_internal_account filter - default to false if not set
	if !config.ExcludeInternalAccount.IsNull() && config.ExcludeInternalAccount.ValueBool() {
		filters = append(filters, excludeInternalAccountFilter())
	}
	if !config.ProviderType.IsNull() && config.ProviderType.ValueString() != "" {
		filters = append(filters, providerTypeFilter(config.ProviderType.ValueString()))
	}
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		filters = append(filters, nameFilter(config.Name.ValueString()))
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
	config.ID = types.StringValue(strconv.Itoa(redis.IntValue(cloudAccount.ID)))
	config.Name = types.StringValue(redis.StringValue(cloudAccount.Name))
	config.AccessKeyID = types.StringValue(redis.StringValue(cloudAccount.AccessKeyID))
	config.ExcludeInternalAccount = types.BoolValue(config.ExcludeInternalAccount.ValueBool())
	config.ProviderType = types.StringValue(redis.StringValue(cloudAccount.Provider))

	// Set state
	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}

func filterCloudAccounts(accounts []*cloud_accounts.CloudAccount, filters []caFilterFunc) []*cloud_accounts.CloudAccount {
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

func filterCloudAccount(account *cloud_accounts.CloudAccount, filters []caFilterFunc) bool {
	for _, f := range filters {
		if !f(account) {
			return false
		}
	}
	return true
}

// excludeInternalAccountFilter matches every cloud account except the Redis Labs
// internal account (id 1).
func excludeInternalAccountFilter() caFilterFunc {
	return func(account *cloud_accounts.CloudAccount) bool {
		return redis.IntValue(account.ID) != 1
	}
}

// providerTypeFilter matches cloud accounts with the given provider (e.g. AWS, GCP).
func providerTypeFilter(providerType string) caFilterFunc {
	return func(account *cloud_accounts.CloudAccount) bool {
		return redis.StringValue(account.Provider) == providerType
	}
}

// nameFilter matches cloud accounts with the given name.
func nameFilter(name string) caFilterFunc {
	return func(account *cloud_accounts.CloudAccount) bool {
		return redis.StringValue(account.Name) == name
	}
}
