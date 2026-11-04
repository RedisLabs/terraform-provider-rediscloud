package transitgateway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *transitGatewayInvitationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model TransitGatewayInvitationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// SDKv2 wrote the attribute's Default of 0 to state when it was not configured.
	if model.WaitForInvitationsTimeout.IsNull() {
		model.WaitForInvitationsTimeout = types.Int64Value(0)
	}

	subId, err := strconv.Atoi(model.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}

	fetch := func(ctx context.Context) ([]*attachments.TransitGatewayInvitation, error) {
		return d.client.Client.TransitGatewayAttachments.ListInvitations(ctx, subId)
	}

	var invitations []*attachments.TransitGatewayInvitation
	if waitTimeoutSeconds := model.WaitForInvitationsTimeout.ValueInt64(); waitTimeoutSeconds > 0 {
		var diags diag.Diagnostics
		invitations, diags = waitForInvitations(ctx, d.client, time.Duration(waitTimeoutSeconds)*time.Second,
			"Transit Gateway", fmt.Sprintf("subscription %d", subId), fetch)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		invitations, err = fetch(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Read Transit Gateway Invitations", err.Error())
			return
		}
	}

	list, diags := TransitGatewayInvitationsFromAPI(ctx, invitations)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	model.Invitations = list
	model.ID = types.StringValue(strconv.Itoa(subId))

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
