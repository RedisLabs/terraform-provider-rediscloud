package transitgateway

import "github.com/hashicorp/terraform-plugin-framework/types"

// TransitGatewayDataSourceModel describes the data source data model.
type TransitGatewayDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	SubscriptionID    types.String `tfsdk:"subscription_id"`
	TgwID             types.Int64  `tfsdk:"tgw_id"`
	AwsTgwUID         types.String `tfsdk:"aws_tgw_uid"`
	WaitForTgwTimeout types.Int64  `tfsdk:"wait_for_tgw_timeout"`
	AttachmentUID     types.String `tfsdk:"attachment_uid"`
	Status            types.String `tfsdk:"status"`
	AttachmentStatus  types.String `tfsdk:"attachment_status"`
	AwsAccountID      types.String `tfsdk:"aws_account_id"`
	Cidrs             types.List   `tfsdk:"cidrs"`
}
