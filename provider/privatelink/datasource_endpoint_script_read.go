package privatelink

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *endpointScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EndpointScriptModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(data.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}

	// The subscription and its PrivateLink are read first so that a missing one is reported before the script request.
	if _, err := d.client.Client.Subscription.Get(ctx, subId); err != nil {
		resp.Diagnostics.AddError("Failed to read subscription", err.Error())
		return
	}

	if _, err := d.client.Client.PrivateLink.GetPrivateLink(ctx, subId); err != nil {
		resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
		return
	}

	endpointScript, err := d.client.Client.PrivateLink.GetPrivateLinkEndpointScript(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read PrivateLink endpoint script", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.Itoa(subId))
	data.EndpointScript = types.StringNull()
	if endpointScript != nil {
		data.EndpointScript = types.StringPointerValue(endpointScript.ResourceEndpointScript)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
