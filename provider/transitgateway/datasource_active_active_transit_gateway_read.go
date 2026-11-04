package transitgateway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// activeActiveTransitGatewayDataSourceReadTimeout matches the SDKv2 default read timeout, which this data source never overrode.
const activeActiveTransitGatewayDataSourceReadTimeout = 20 * time.Minute

func (d *activeActiveTransitGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model ActiveActiveTransitGatewayDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// SDKv2 wrote the attribute's Default of 0 to state when it was not configured.
	if model.WaitForTgwTimeout.IsNull() {
		model.WaitForTgwTimeout = types.Int64Value(0)
	}

	subId, err := strconv.Atoi(model.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}
	regionId := int(model.RegionID.ValueInt64())

	filters := transitGatewayDataSourceFilters(model.TgwID, model.AwsTgwUID)
	fetch := func(ctx context.Context) (*attachments.GetAttachmentsTask, error) {
		return d.client.Client.TransitGatewayAttachments.GetActiveActive(ctx, subId, regionId)
	}

	var filtered []*attachments.TransitGatewayAttachment
	if waitTimeoutSeconds := model.WaitForTgwTimeout.ValueInt64(); waitTimeoutSeconds > 0 {
		var diags diag.Diagnostics
		filtered, diags = waitForMatchingTransitGateways(ctx, d.client, time.Duration(waitTimeoutSeconds)*time.Second,
			"Active-Active Transit Gateway", fmt.Sprintf("subscription %d, region %d", subId, regionId), fetch, filters)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		tgwTask, err := utils.WaitForActiveActiveTransitGatewayResourceToBeAvailable(ctx, subId, regionId, d.client, activeActiveTransitGatewayDataSourceReadTimeout)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Active-Active Transit Gateway", err.Error())
			return
		}

		filtered = FilterTransitGatewayAttachments(tgwTask, filters)
	}

	tgw, diags := singleTransitGateway(filtered)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cidrs, diags := CidrsFromAPI(ctx, tgw.Cidrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tgwId := redis.IntValue(tgw.Id)
	model.ID = types.StringValue(utils.BuildResourceId(subId, tgwId))
	//TODO(TF3.0) refactor tgw_id to a StringAttribute
	model.TgwID = types.Int64Value(int64(tgwId))
	//TODO(TF3.0) make aws_tgw_uid, attachment_uid, status, attachment_status and aws_account_id nullable
	model.AwsTgwUID = types.StringValue(redis.StringValue(tgw.AwsTgwUid))
	model.AttachmentUID = types.StringValue(redis.StringValue(tgw.AttachmentUid))
	model.Status = types.StringValue(redis.StringValue(tgw.Status))
	model.AttachmentStatus = types.StringValue(redis.StringValue(tgw.AttachmentStatus))
	model.AwsAccountID = types.StringValue(redis.StringValue(tgw.AwsAccountId))
	model.Cidrs = cidrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
