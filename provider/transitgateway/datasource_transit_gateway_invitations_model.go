package transitgateway

import "github.com/hashicorp/terraform-plugin-framework/types"

// TransitGatewayInvitationsDataSourceModel describes the data source data model.
type TransitGatewayInvitationsDataSourceModel struct {
	ID                        types.String `tfsdk:"id"`
	SubscriptionID            types.String `tfsdk:"subscription_id"`
	WaitForInvitationsTimeout types.Int64  `tfsdk:"wait_for_invitations_timeout"`
	Invitations               types.List   `tfsdk:"invitations"`
}
