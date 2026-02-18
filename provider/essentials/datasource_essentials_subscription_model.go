package essentials

import "github.com/hashicorp/terraform-plugin-framework/types"

// EssentialsSubscriptionDataSourceModel describes the data source data model.
type EssentialsSubscriptionDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	SubscriptionID  types.Int64  `tfsdk:"subscription_id"`
	Name            types.String `tfsdk:"name"`
	Status          types.String `tfsdk:"status"`
	PlanID          types.Int64  `tfsdk:"plan_id"`
	PaymentMethodID types.Int64  `tfsdk:"payment_method_id"`
	CreationDate    types.String `tfsdk:"creation_date"`
}
