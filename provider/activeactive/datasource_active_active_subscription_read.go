package activeactive

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// Read refreshes the Terraform state with the latest data.
func (d *activeActiveSubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var model ActiveActiveSubscriptionDataSourceModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subs, err := d.client.Client.Subscription.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Active-Active Subscriptions",
			err.Error(),
		)
		return
	}

	// AA and pro subscriptions come from the same endpoint; keep only active-active.
	filters := []utils.SubscriptionFilter{
		utils.ActiveActiveSubscriptionFilter(),
	}

	if !model.Name.IsNull() && model.Name.ValueString() != "" {
		filters = append(filters, utils.SubscriptionNameFilter(model.Name.ValueString()))
	}

	subs = utils.FilterSubscriptions(subs, filters)

	if len(subs) == 0 {
		resp.Diagnostics.AddError(
			"No Active-Active Subscriptions Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(subs) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Active-Active Subscriptions Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	sub := subs[0]
	subId := redis.IntValue(sub.ID)

	model.ID = types.StringValue(strconv.Itoa(subId))
	model.Name = types.StringPointerValue(sub.Name)

	if sub.PaymentMethodID != nil {
		model.PaymentMethodID = types.StringValue(strconv.Itoa(redis.IntValue(sub.PaymentMethodID)))
	} else {
		model.PaymentMethodID = types.StringNull()
	}
	model.PaymentMethod = types.StringPointerValue(sub.PaymentMethod)
	model.NumberOfDatabases = types.Int64Value(int64(redis.IntValue(sub.NumberOfDatabases)))
	model.Status = types.StringPointerValue(sub.Status)

	cmkEnabled := sub.PersistentStorageEncryptionType != nil &&
		redis.StringValue(sub.PersistentStorageEncryptionType) == utils.CmkEnabledString
	model.CustomerManagedKeyEnabled = types.BoolValue(cmkEnabled)

	if sub.CustomerManagedKeyAccessDetails != nil {
		model.CustomerManagedKeyRedisServiceAccount = types.StringPointerValue(sub.CustomerManagedKeyAccessDetails.RedisServiceAccount)
		model.CustomerManagedKeyAwsRoleArn = types.StringPointerValue(sub.CustomerManagedKeyAccessDetails.AwsRoleArn)
	} else {
		model.CustomerManagedKeyRedisServiceAccount = types.StringNull()
		model.CustomerManagedKeyAwsRoleArn = types.StringNull()
	}

	model.CustomerManagedKeyDeletionGracePeriod = types.StringPointerValue(sub.DeletionGracePeriod)

	// Default to true if not set by API, matching Redis Cloud's default behaviour
	publicEndpointAccess := true
	if sub.PublicEndpointAccess != nil {
		publicEndpointAccess = redis.BoolValue(sub.PublicEndpointAccess)
	}
	model.PublicEndpointAccess = types.BoolValue(publicEndpointAccess)

	cloudDetails := sub.CloudDetails
	if len(cloudDetails) == 0 {
		// A subscription with 0 databases will have no CloudDetail blocks
		model.CloudProvider = types.StringNull()
		model.AwsAccountID = types.StringNull()
		model.ResourceTags = types.MapNull(types.StringType)
	} else {
		cd := cloudDetails[0]
		model.CloudProvider = types.StringPointerValue(cd.Provider)
		model.AwsAccountID = types.StringPointerValue(cd.AWSAccountID)

		tags, diags := utils.FlattenResourceTags(ctx, cd.ResourceTags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		model.ResourceTags = tags
	}

	m, err := d.client.Client.Maintenance.Get(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Subscription Maintenance Windows",
			err.Error(),
		)
		return
	}
	maintenanceWindows, diags := utils.FlattenMaintenance(ctx, m)
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
	pricing, diags := utils.FlattenPricing(ctx, pricingList)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.Pricing = pricing

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
