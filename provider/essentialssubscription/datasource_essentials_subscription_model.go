package essentialssubscription

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EssentialsSubscriptionModel describes the data source data model.
type EssentialsSubscriptionModel struct {
	ID              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Status          types.String `tfsdk:"status"`
	PlanID          types.Int64  `tfsdk:"plan_id"`
	PaymentMethodID types.Int64  `tfsdk:"payment_method_id"`
	CreationDate    types.String `tfsdk:"creation_date"`
}
