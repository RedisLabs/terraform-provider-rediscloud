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

func (d *transitGatewayDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model TransitGatewayDataSourceModel
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

	filters := transitGatewayDataSourceFilters(model.TgwID, model.AwsTgwUID)
	fetch := func(ctx context.Context) (*attachments.GetAttachmentsTask, error) {
		return d.client.Client.TransitGatewayAttachments.Get(ctx, subId)
	}

	var filtered []*attachments.TransitGatewayAttachment
	if waitTimeoutSeconds := model.WaitForTgwTimeout.ValueInt64(); waitTimeoutSeconds > 0 {
		var diags diag.Diagnostics
		filtered, diags = waitForMatchingTransitGateways(ctx, d.client, time.Duration(waitTimeoutSeconds)*time.Second,
			"Transit Gateway", fmt.Sprintf("subscription %d", subId), fetch, filters)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		tgwTask, err := fetch(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Transit Gateway", err.Error())
			return
		}

		if tgwTask == nil {
			resp.Diagnostics.AddError("Unable to Read Transit Gateway", fmt.Sprintf("Transit Gateway API returned nil task for subscription %d", subId))
			return
		}
		if tgwTask.Response == nil {
			resp.Diagnostics.AddError("Unable to Read Transit Gateway", fmt.Sprintf("Transit Gateway API returned nil response for subscription %d", subId))
			return
		}
		if tgwTask.Response.Resource == nil {
			resp.Diagnostics.AddError("Unable to Read Transit Gateway", fmt.Sprintf("Transit Gateway API returned nil resource for subscription %d - subscription may not be fully provisioned yet", subId))
			return
		}

		filtered = filterTransitGatewayAttachments(tgwTask, filters)
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
