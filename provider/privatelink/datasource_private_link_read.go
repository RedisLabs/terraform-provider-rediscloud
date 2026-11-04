package privatelink

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *privateLinkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateLinkDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(data.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}

	privateLink, err := d.client.Client.PrivateLink.GetPrivateLink(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.Itoa(subId))
	data.ShareName = types.StringPointerValue(privateLink.ShareName)
	data.ResourceConfigurationID = types.StringPointerValue(privateLink.ResourceConfigurationId)
	data.ResourceConfigurationArn = types.StringPointerValue(privateLink.ResourceConfigurationArn)
	data.ShareArn = types.StringPointerValue(privateLink.ShareArn)

	var diags diag.Diagnostics
	data.Principals, diags = PrincipalsFromAPI(ctx, privateLink.Principals)
	resp.Diagnostics.Append(diags...)
	data.Connections, diags = ConnectionsFromAPI(ctx, privateLink.Connections)
	resp.Diagnostics.Append(diags...)
	data.Databases, diags = DatabasesFromAPI(ctx, privateLink.Databases)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
