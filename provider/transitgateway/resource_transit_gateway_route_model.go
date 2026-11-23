package transitgateway

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TransitGatewayRouteResourceModel describes the resource data model.
type TransitGatewayRouteResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	SubscriptionID types.String   `tfsdk:"subscription_id"`
	TgwID          types.Int64    `tfsdk:"tgw_id"`
	Cidrs          types.List     `tfsdk:"cidrs"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}
