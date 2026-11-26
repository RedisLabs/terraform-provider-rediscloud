package pro

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Read refreshes the Terraform state with the latest data.
func (d *proSubscriptionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ProSubscriptionsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The filters block is optional; a null name means "no name filter" (return all).
	nameFilter := types.StringNull()
	if config.Filters != nil {
		nameFilter = config.Filters.Name
	}

	subs, diags := listProSubscriptions(ctx, d.client, nameFilter)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A non-null but possibly empty list: an account with no matching pro subscriptions
	// yields an empty subscriptions list rather than an error.
	subscriptionList := make([]ProSubscriptionModel, 0, len(subs))
	for _, sub := range subs {
		model, diags := mapProSubscription(ctx, sub)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		subscriptionList = append(subscriptionList, model)
	}

	// Echo the config's filters back: it's an optional, non-computed input, so state must
	// match config.
	model := ProSubscriptionsDataSourceModel{
		Filters:       config.Filters,
		Subscriptions: subscriptionList,
	}

	diags = resp.State.Set(ctx, &model)
	resp.Diagnostics.Append(diags...)
}
