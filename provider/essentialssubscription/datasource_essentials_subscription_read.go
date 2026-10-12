package essentialssubscription

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fs "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
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

	var model EssentialsSubscriptionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subs, err := d.client.Client.FixedSubscriptions.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Essentials Subscriptions",
			err.Error(),
		)
		return
	}

	var filters []func(sub *fs.FixedSubscriptionResponse) bool

	// SDKv2 GetOk treated id = 0 and name = "" as unset; keep that so existing configs behave the same.
	if utils.IsConfigured(model.ID) && model.ID.ValueInt64() != 0 {
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			return int64(redis.IntValue(sub.ID)) == model.ID.ValueInt64()
		})
	}

	if utils.IsConfigured(model.Name) && model.Name.ValueString() != "" {
		filters = append(filters, func(sub *fs.FixedSubscriptionResponse) bool {
			return redis.StringValue(sub.Name) == model.Name.ValueString()
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
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	sub := subs[0]
	model.ID = types.Int64Value(int64(redis.IntValue(sub.ID)))
	model.Name = types.StringValue(redis.StringValue(sub.Name))
	model.Status = types.StringValue(redis.StringValue(sub.Status))
	model.PlanID = types.Int64Value(int64(redis.IntValue(sub.PlanId)))
	// SDKv2 wrote 0 for subscriptions without a payment method (free plans); keep it for backward compatibility.
	//TODO(TF3.0) make payment_method_id nullable
	model.PaymentMethodID = types.Int64Value(int64(redis.IntValue(sub.PaymentMethodID)))
	model.CreationDate = types.StringValue(redis.TimeValue(sub.CreationDate).String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
