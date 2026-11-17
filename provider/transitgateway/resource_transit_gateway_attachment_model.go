package transitgateway

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TransitGatewayAttachmentResourceModel describes the resource data model.
type TransitGatewayAttachmentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	//TODO(TF3.0) refactor tgw_id to a StringAttribute
	TgwID            types.Int64    `tfsdk:"tgw_id"`
	AwsTgwUid        types.String   `tfsdk:"aws_tgw_uid"`
	AttachmentUid    types.String   `tfsdk:"attachment_uid"`
	Status           types.String   `tfsdk:"status"`
	AttachmentStatus types.String   `tfsdk:"attachment_status"`
	AwsAccountID     types.String   `tfsdk:"aws_account_id"`
	Cidrs            types.List     `tfsdk:"cidrs"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// ActiveActiveTransitGatewayAttachmentResourceModel describes the resource data model.
type ActiveActiveTransitGatewayAttachmentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	RegionID       types.String `tfsdk:"region_id"`
	//TODO(TF3.0) refactor tgw_id to a StringAttribute
	TgwID            types.Int64    `tfsdk:"tgw_id"`
	AwsTgwUid        types.String   `tfsdk:"aws_tgw_uid"`
	AttachmentUid    types.String   `tfsdk:"attachment_uid"`
	Status           types.String   `tfsdk:"status"`
	AttachmentStatus types.String   `tfsdk:"attachment_status"`
	AwsAccountID     types.String   `tfsdk:"aws_account_id"`
	Cidrs            types.List     `tfsdk:"cidrs"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// transitGatewayAttachmentComputed holds the attributes both attachment resources read from the API.
type transitGatewayAttachmentComputed struct {
	AwsTgwUid        *types.String
	AttachmentUid    *types.String
	Status           *types.String
	AttachmentStatus *types.String
	AwsAccountID     *types.String
	Cidrs            *types.List
}

func (m *TransitGatewayAttachmentResourceModel) computed() transitGatewayAttachmentComputed {
	return transitGatewayAttachmentComputed{&m.AwsTgwUid, &m.AttachmentUid, &m.Status, &m.AttachmentStatus, &m.AwsAccountID, &m.Cidrs}
}

func (m *ActiveActiveTransitGatewayAttachmentResourceModel) computed() transitGatewayAttachmentComputed {
	return transitGatewayAttachmentComputed{&m.AwsTgwUid, &m.AttachmentUid, &m.Status, &m.AttachmentStatus, &m.AwsAccountID, &m.Cidrs}
}

// TODO(TF3.0) make aws_tgw_uid, attachment_uid, status, attachment_status and aws_account_id nullable
func (c transitGatewayAttachmentComputed) setFromAPI(ctx context.Context, tgw *attachments.TransitGatewayAttachment) diag.Diagnostics {
	*c.AwsTgwUid = types.StringValue(redis.StringValue(tgw.AwsTgwUid))
	*c.AttachmentUid = types.StringValue(redis.StringValue(tgw.AttachmentUid))
	*c.Status = types.StringValue(redis.StringValue(tgw.Status))
	*c.AttachmentStatus = types.StringValue(redis.StringValue(tgw.AttachmentStatus))
	*c.AwsAccountID = types.StringValue(redis.StringValue(tgw.AwsAccountId))

	cidrs, diags := CidrsFromAPI(ctx, tgw.Cidrs)
	*c.Cidrs = cidrs
	return diags
}

// setUnknownToNull keeps state free of unknown values when an apply stops before the attachment could be read.
func (c transitGatewayAttachmentComputed) setUnknownToNull() {
	for _, value := range []*types.String{c.AwsTgwUid, c.AttachmentUid, c.Status, c.AttachmentStatus, c.AwsAccountID} {
		if value.IsUnknown() {
			*value = types.StringNull()
		}
	}
	if c.Cidrs.IsUnknown() {
		*c.Cidrs = types.ListNull(types.StringType)
	}
}

// keepKnownPlanValues restores the planned values that are already known, because the framework rejects an apply result that differs from them.
func (c transitGatewayAttachmentComputed) keepKnownPlanValues(plan transitGatewayAttachmentComputed) {
	for _, pair := range [][2]*types.String{
		{c.AwsTgwUid, plan.AwsTgwUid},
		{c.AttachmentUid, plan.AttachmentUid},
		{c.Status, plan.Status},
		{c.AttachmentStatus, plan.AttachmentStatus},
		{c.AwsAccountID, plan.AwsAccountID},
	} {
		if !pair[1].IsUnknown() {
			*pair[0] = *pair[1]
		}
	}
	if !plan.Cidrs.IsUnknown() {
		*c.Cidrs = *plan.Cidrs
	}
}
