package pro

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProSubscriptionsDataSourceModel describes the list data source data model. Each element
// is only a subset of the singular data source model, without maintenance_windows and
// pricing, because both of those require additional API calls per subscription.
type ProSubscriptionsDataSourceModel struct {
	// Filters holds the optional query filters. A pointer so an omitted `filters {}` block
	// is nil rather than an object of nulls.
	Filters       *proSubscriptionsFiltersModel `tfsdk:"filters"`
	Subscriptions []ProSubscriptionModel        `tfsdk:"subscriptions"`
}

// proSubscriptionsFiltersModel is the filters block. It exists to make the filters'
// purpose explicit and to leave room for more filter fields alongside name.
type proSubscriptionsFiltersModel struct {
	Name types.String `tfsdk:"name"`
}
