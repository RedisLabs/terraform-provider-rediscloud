package essentialsplan

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/fixed/plans"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Read refreshes the Terraform state with the latest data.
func (d *essentialsPlanDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Defensive nil check for client
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var model EssentialsPlanModel
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, err := getPlanList(ctx, model, d.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Essentials Plans",
			err.Error(),
		)
		return
	}

	// Build filters based on configuration
	var filters []func(plan *plans.GetPlanResponse) bool

	// Filter by ID if specified
	if !model.ID.IsNull() {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.IntValue(plan.ID) == int(model.ID.ValueInt64())
		})
	}

	if !model.Name.IsNull() && model.Name.ValueString() != "" {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Name) == model.Name.ValueString()
		})
	}

	if !model.Size.IsNull() {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.Float64Value(plan.Size) == model.Size.ValueFloat64()
		})
	}

	if !model.SizeMeasurementUnit.IsNull() && model.SizeMeasurementUnit.ValueString() != "" {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.SizeMeasurementUnit) == model.SizeMeasurementUnit.ValueString()
		})
	}

	if !model.CloudProvider.IsNull() && model.CloudProvider.ValueString() != "" {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Provider) == model.CloudProvider.ValueString()
		})
	}

	if !model.Region.IsNull() && model.Region.ValueString() != "" {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Region) == model.Region.ValueString()
		})
	}

	if !model.Availability.IsNull() && model.Availability.ValueString() != "" {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.StringValue(plan.Availability) == model.Availability.ValueString()
		})
	}

	if !model.SupportDataPersistence.IsNull() {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.BoolValue(plan.SupportDataPersistence) == model.SupportDataPersistence.ValueBool()
		})
	}

	if !model.SupportReplication.IsNull() {
		filters = append(filters, func(plan *plans.GetPlanResponse) bool {
			return redis.BoolValue(plan.SupportReplication) == model.SupportReplication.ValueBool()
		})
	}

	// Apply filters
	list = filterPlans(list, filters)

	if len(list) == 0 {
		resp.Diagnostics.AddError(
			"No Essentials Plans Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(list) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Essentials Plans Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	// Map the result to state
	plan := list[0]
	model.ID = types.Int64Value(int64(redis.IntValue(plan.ID)))
	model.Name = types.StringPointerValue(plan.Name)
	model.Size = types.Float64PointerValue(plan.Size)
	model.SizeMeasurementUnit = types.StringPointerValue(plan.SizeMeasurementUnit)
	model.CloudProvider = types.StringPointerValue(plan.Provider)
	model.Region = types.StringPointerValue(plan.Region)
	model.RegionID = types.Int64Value(int64(redis.IntValue(plan.RegionID)))
	model.Price = types.Int64Value(int64(redis.IntValue(plan.Price)))
	model.PriceCurrency = types.StringPointerValue(plan.PriceCurrency)
	model.PricePeriod = types.StringPointerValue(plan.PricePeriod)
	model.MaximumDatabases = types.Int64Value(int64(redis.IntValue(plan.MaximumDatabases)))
	model.MaximumThroughput = types.Int64Value(int64(redis.IntValue(plan.MaximumThroughput)))
	model.MaximumBandwidthInGB = types.Int64Value(int64(redis.IntValue(plan.MaximumBandwidthGB)))
	model.Availability = types.StringPointerValue(plan.Availability)
	model.Connections = types.StringPointerValue(plan.Connections)
	model.CidrAllowRules = types.Int64Value(int64(redis.IntValue(plan.CidrAllowRules)))
	model.SupportDataPersistence = types.BoolPointerValue(plan.SupportDataPersistence)
	model.SupportInstantAndDailyBackups = types.BoolPointerValue(plan.SupportInstantAndDailyBackups)
	model.SupportReplication = types.BoolPointerValue(plan.SupportReplication)
	model.SupportClustering = types.BoolPointerValue(plan.SupportClustering)
	alerts, diags := types.ListValueFrom(ctx, types.StringType, plan.SupportedAlerts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	model.SupportedAlerts = alerts
	model.CustomerSupport = types.StringValue(redis.StringValue(plan.CustomerSupport))

	// Set state
	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}

func getPlanList(ctx context.Context, model EssentialsPlanModel, api *client.ApiClient) ([]*plans.GetPlanResponse, error) {
	var list []*plans.GetPlanResponse
	var err error

	if !model.SubscriptionID.IsNull() {
		list, err = api.Client.FixedPlanSubscriptions.List(ctx, int(model.SubscriptionID.ValueInt64()))
	} else if !model.CloudProvider.IsNull() && model.CloudProvider.ValueString() != "" {
		list, err = api.Client.FixedPlans.ListWithProvider(ctx, strings.ToUpper(model.CloudProvider.ValueString()))
	} else {
		list, err = api.Client.FixedPlans.List(ctx)
	}

	return list, err
}

func filterPlans(allPlans []*plans.GetPlanResponse, filters []func(plan *plans.GetPlanResponse) bool) []*plans.GetPlanResponse {
	var filtered []*plans.GetPlanResponse
	for _, candidatePlan := range allPlans {
		if filterPlan(candidatePlan, filters) {
			filtered = append(filtered, candidatePlan)
		}
	}

	return filtered
}

func filterPlan(plan *plans.GetPlanResponse, filters []func(plan *plans.GetPlanResponse) bool) bool {
	for _, f := range filters {
		if !f(plan) {
			return false
		}
	}
	return true
}
