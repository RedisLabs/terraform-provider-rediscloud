package transitgateway

import "github.com/hashicorp/terraform-plugin-framework/types"

// ActiveActiveTransitGatewayInvitationsDataSourceModel describes the data source data model.
type ActiveActiveTransitGatewayInvitationsDataSourceModel struct {
	ID                        types.String `tfsdk:"id"`
	SubscriptionID            types.String `tfsdk:"subscription_id"`
	RegionID                  types.Int64  `tfsdk:"region_id"`
	WaitForInvitationsTimeout types.Int64  `tfsdk:"wait_for_invitations_timeout"`
	Invitations               types.List   `tfsdk:"invitations"`
}
