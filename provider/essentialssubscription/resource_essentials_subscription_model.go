package essentialssubscription

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EssentialsSubscriptionResourceModel describes the resource data model.
type EssentialsSubscriptionResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	Status          types.String   `tfsdk:"status"`
	PlanID          types.Int64    `tfsdk:"plan_id"`
	PaymentMethod   types.String   `tfsdk:"payment_method"`
	PaymentMethodID types.Int64    `tfsdk:"payment_method_id"`
	CreationDate    types.String   `tfsdk:"creation_date"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}
