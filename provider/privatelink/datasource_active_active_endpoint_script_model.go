package privatelink

import "github.com/hashicorp/terraform-plugin-framework/types"

// ActiveActiveEndpointScriptModel describes the Active-Active PrivateLink endpoint script data source.
type ActiveActiveEndpointScriptModel struct {
	ID             types.String `tfsdk:"id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	RegionID       types.Int64  `tfsdk:"region_id"`
	EndpointScript types.String `tfsdk:"endpoint_script"`
}
