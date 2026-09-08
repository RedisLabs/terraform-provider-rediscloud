package pro

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func (d *proSubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var model ProSubscriptionDataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subs, err := d.client.Client.Subscription.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Pro Subscriptions",
			err.Error(),
		)
		return
	}

	// AA and pro subscriptions come from the same endpoint; keep only pro.
	filters := []utils.SubscriptionFilter{
		utils.ProSubscriptionFilter(),
	}

	if !model.Name.IsNull() && model.Name.ValueString() != "" {
		filters = append(filters, utils.SubscriptionNameFilter(model.Name.ValueString()))
	}

	subs = utils.FilterSubscriptions(subs, filters)

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

	sub := subs[0]
	subId := redis.IntValue(sub.ID)

	model.ID = types.StringValue(strconv.Itoa(subId))
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

	cloudProvider, diags := utils.CloudProvidersFromAPI(ctx, sub.CloudDetails)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.CloudProvider = cloudProvider

	m, err := d.client.Client.Maintenance.Get(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Subscription Maintenance Windows",
			err.Error(),
		)
		return
	}
	maintenanceWindows, diags := customtypes.NewMaintenanceList(ctx, m)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.MaintenanceWindows = maintenanceWindows

	pricingList, err := d.client.Client.Pricing.List(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Subscription Pricing",
			err.Error(),
		)
		return
	}
	pricing, diags := utils.PricingFromAPI(ctx, pricingList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Pricing = pricing

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
