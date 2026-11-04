package privatelink

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *activeActiveEndpointScriptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActiveActiveEndpointScriptModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(data.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}
	regionId := int(data.RegionID.ValueInt64())

	endpointScript, err := d.client.Client.PrivateLink.GetActiveActivePrivateLinkEndpointScript(ctx, subId, regionId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Active-Active PrivateLink endpoint script", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.Itoa(subId))
	data.EndpointScript = types.StringNull()
	if endpointScript != nil {
		data.EndpointScript = types.StringPointerValue(endpointScript.ResourceEndpointScript)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
