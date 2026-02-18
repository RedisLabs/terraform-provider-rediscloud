package essentials

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fs "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read refreshes the Terraform state with the latest data.
func (d *essentialsSubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var state EssentialsSubscriptionDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	subs, err := d.client.Client.FixedSubscriptions.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Essentials Subscriptions",
			fmt.Sprintf("An error occurred while reading Essentials subscriptions: %s", err.Error()),
		)
		return
	}

	var filters []func(sub *fs.FixedSubscriptionResponse) bool

	if !state.SubscriptionID.IsNull() {
		subID := int(state.SubscriptionID.ValueInt64())
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			if sub == nil {
				return false
			}
			return redis.IntValue(sub.ID) == subID
		})
	}

	// Support the deprecated id attribute as a filter for backward compatibility
	if !state.ID.IsNull() && state.ID.ValueString() != "" {
		idStr := state.ID.ValueString()
		idInt, err := strconv.Atoi(idStr)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid ID",
				fmt.Sprintf("The id attribute must be a numeric subscription ID, got: %s", idStr),
			)
			return
		}
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			if sub == nil {
				return false
			}
			return redis.IntValue(sub.ID) == idInt
		})
	}

	if !state.Name.IsNull() && state.Name.ValueString() != "" {
		name := state.Name.ValueString()
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			if sub == nil {
				return false
			}
			return redis.StringValue(sub.Name) == name
		})
	}

	subs = filterFixedSubscriptions(subs, filters)

	if len(subs) == 0 {
		resp.Diagnostics.AddError(
			"No Essentials Subscriptions Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(subs) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Essentials Subscriptions Found",
			"Your query returned more than one result. Please try a more specific search criteria and try again.",
		)
		return
	}

	sub := subs[0]

	state.ID = types.StringValue(strconv.Itoa(redis.IntValue(sub.ID)))
	state.SubscriptionID = types.Int64Value(int64(redis.IntValue(sub.ID)))
	state.Name = types.StringValue(redis.StringValue(sub.Name))
	state.Status = types.StringValue(redis.StringValue(sub.Status))
	state.PlanID = types.Int64Value(int64(redis.IntValue(sub.PlanId)))
	state.PaymentMethodID = types.Int64Value(int64(redis.IntValue(sub.PaymentMethodID)))
	state.CreationDate = types.StringValue(redis.TimeValue(sub.CreationDate).String())

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func filterFixedSubscriptions(subs []*fs.FixedSubscriptionResponse, filters []func(sub *fs.FixedSubscriptionResponse) bool) []*fs.FixedSubscriptionResponse {
	var filtered []*fs.FixedSubscriptionResponse
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if filterFixedSubscription(sub, filters) {
			filtered = append(filtered, sub)
		}
	}
	return filtered
}

func filterFixedSubscription(sub *fs.FixedSubscriptionResponse, filters []func(sub *fs.FixedSubscriptionResponse) bool) bool {
	for _, f := range filters {
		if !f(sub) {
			return false
		}
	}
	return true
}
