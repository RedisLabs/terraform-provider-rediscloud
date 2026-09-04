package pro

import (
	"context"
	"sort"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// Read refreshes the Terraform state with the latest data.
func (d *proSubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProSubscriptionDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subs, diags := listProSubscriptions(ctx, d.client, config.Name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(subs) == 0 {
		resp.Diagnostics.AddError(
			"No Pro Subscriptions Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(subs) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Pro Subscriptions Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	model, diags := mapProSubscriptionWithBlocks(ctx, d.client, subs[0])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

// listProSubscriptions fetches every pro subscription in the account, optionally narrowed
// to an exact name. The returned slice is sorted by subscription id so callers get a
// stable order (see the sort comment below)
func listProSubscriptions(ctx context.Context, api *client.ApiClient, name types.String) ([]*subscriptions.Subscription, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Defensive nil check for client.
	if api == nil {
		diags.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return nil, diags
	}

	subs, err := api.Client.Subscription.List(ctx)
	if err != nil {
		diags.AddError(
			"Unable to Read Pro Subscriptions",
			err.Error(),
		)
		return nil, diags
	}

	// AA and pro subscriptions come from the same endpoint; keep only pro.
	filters := []utils.SubscriptionFilter{
		utils.ProSubscriptionFilter(),
	}

	// name is an optional exact-match filter: when null/empty every pro subscription is
	// returned, when set only exact-name matches are.
	if !name.IsNull() && name.ValueString() != "" {
		filters = append(filters, utils.SubscriptionNameFilter(name.ValueString()))
	}

	filtered := utils.FilterSubscriptions(subs, filters)

	// CAPI's GET /v1/subscriptions has no guaranteed order, so sort by the numeric
	// subscription id to keep the result stable across reads. Without this the list
	// data source would churn a perpetual plan diff.
	sort.SliceStable(filtered, func(i, j int) bool {
		return redis.IntValue(filtered[i].ID) < redis.IntValue(filtered[j].ID)
	})

	return filtered, diags
}

// mapProSubscription maps the basic pro subscription fields - the ones that are available
// directly from the Subscription.List response
func mapProSubscription(ctx context.Context, sub *subscriptions.Subscription) (ProSubscriptionModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var model ProSubscriptionModel

	model.ID = types.StringValue(strconv.Itoa(redis.IntValue(sub.ID)))
	model.Name = types.StringValue(redis.StringValue(sub.Name))

	paymentMethodID := ""
	if sub.PaymentMethodID != nil {
		paymentMethodID = strconv.Itoa(redis.IntValue(sub.PaymentMethodID))
	}
	model.PaymentMethodID = types.StringValue(paymentMethodID)
	model.PaymentMethod = types.StringValue(redis.StringValue(sub.PaymentMethod))
	model.MemoryStorage = types.StringValue(redis.StringValue(sub.MemoryStorage))
	model.NumberOfDatabases = types.Int64Value(int64(redis.IntValue(sub.NumberOfDatabases)))
	model.Status = types.StringValue(redis.StringValue(sub.Status))
	model.PrometheusEndpoint = types.StringValue(redis.StringValue(sub.PrometheusEndpoint))

	cmkEnabled := sub.PersistentStorageEncryptionType != nil &&
		redis.StringValue(sub.PersistentStorageEncryptionType) == utils.CmkEnabledString
	model.CustomerManagedKeyEnabled = types.BoolValue(cmkEnabled)

	cmkServiceAccount := ""
	cmkAwsRoleArn := ""
	if sub.CustomerManagedKeyAccessDetails != nil {
		cmkServiceAccount = redis.StringValue(sub.CustomerManagedKeyAccessDetails.RedisServiceAccount)
		cmkAwsRoleArn = redis.StringValue(sub.CustomerManagedKeyAccessDetails.AwsRoleArn)
	}
	model.CustomerManagedKeyRedisServiceAccount = types.StringValue(cmkServiceAccount)
	model.CustomerManagedKeyAwsRoleArn = types.StringValue(cmkAwsRoleArn)

	model.CustomerManagedKeyDeletionGracePeriod = types.StringValue(redis.StringValue(sub.DeletionGracePeriod))

	// Default to true if not set by API, matching Redis Cloud's default behaviour
	publicEndpointAccess := true
	if sub.PublicEndpointAccess != nil {
		publicEndpointAccess = redis.BoolValue(sub.PublicEndpointAccess)
	}
	model.PublicEndpointAccess = types.BoolValue(publicEndpointAccess)

	cloudProvider, d := utils.CloudProvidersFromAPI(ctx, sub.CloudDetails)
	diags.Append(d...)
	if diags.HasError() {
		return model, diags
	}
	model.CloudProvider = cloudProvider

	return model, diags
}

// mapProSubscriptionWithBlocks builds the full singular data-source model for a pro subscription:
// the basic subscription fields plus maintenance windows and pricing
func mapProSubscriptionWithBlocks(ctx context.Context, api *client.ApiClient, sub *subscriptions.Subscription) (ProSubscriptionDataSourceModel, diag.Diagnostics) {
	mappedSub, diags := mapProSubscription(ctx, sub)
	model := ProSubscriptionDataSourceModel{ProSubscriptionModel: mappedSub}
	if diags.HasError() {
		return model, diags
	}

	subId := redis.IntValue(sub.ID)

	m, err := api.Client.Maintenance.Get(ctx, subId)
	if err != nil {
		diags.AddError(
			"Unable to Read Subscription Maintenance Windows",
			err.Error(),
		)
		return model, diags
	}
	maintenanceWindows, d := customtypes.NewMaintenanceList(ctx, m)
	diags.Append(d...)
	if diags.HasError() {
		return model, diags
	}
	model.MaintenanceWindows = maintenanceWindows

	pricingList, err := api.Client.Pricing.List(ctx, subId)
	if err != nil {
		diags.AddError(
			"Unable to Read Subscription Pricing",
			err.Error(),
		)
		return model, diags
	}
	pricing, d := utils.PricingFromAPI(ctx, pricingList)
	diags.Append(d...)
	if diags.HasError() {
		return model, diags
	}
	model.Pricing = pricing

	return model, diags
}
