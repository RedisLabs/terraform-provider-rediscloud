package privatelink

import "github.com/hashicorp/terraform-plugin-framework/types"

// EndpointScriptModel describes the Pro subscription PrivateLink endpoint script data source.
type EndpointScriptModel struct {
	ID             types.String `tfsdk:"id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	EndpointScript types.String `tfsdk:"endpoint_script"`
}
