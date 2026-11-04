package privatelink

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *activeActivePrivateLinkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActiveActivePrivateLinkDataSourceModel
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

	privateLink, err := d.client.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Active-Active PrivateLink", err.Error())
		return
	}

	data.ID = types.StringValue(makeActiveActivePrivateLinkId(subId, regionId))
	data.ResourceConfigurationID = types.StringPointerValue(privateLink.ResourceConfigurationId)
	data.ResourceConfigurationArn = types.StringPointerValue(privateLink.ResourceConfigurationArn)
	data.ShareArn = types.StringPointerValue(privateLink.ShareArn)
	data.ShareName = types.StringPointerValue(privateLink.ShareName)
	if privateLink.SubscriptionId != nil {
		data.SubscriptionID = types.StringValue(strconv.Itoa(*privateLink.SubscriptionId))
	}
	if privateLink.RegionId != nil {
		data.RegionID = types.Int64Value(int64(*privateLink.RegionId))
	}
	// The SDKv2 implementation declared principals but never set them, so they stay null.
	data.Principals = types.ListNull(principalElemType)

	var diags diag.Diagnostics
	data.Connections, diags = ConnectionsFromAPI(ctx, privateLink.Connections)
	resp.Diagnostics.Append(diags...)
	data.Databases, diags = DatabasesFromAPI(ctx, privateLink.Databases)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
