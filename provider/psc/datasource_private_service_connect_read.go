package psc

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *privateServiceConnectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateServiceConnectModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(data.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}

	service, err := d.client.Client.PrivateServiceConnect.GetService(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Private Service Connect service", err.Error())
		return
	}
	if service.ID == nil {
		resp.Diagnostics.AddError("Failed to read Private Service Connect service", "the API returned a service without an ID")
		return
	}

	data.ID = types.StringValue(buildPrivateServiceConnectId(subId, *service.ID))
	data.PrivateServiceConnectServiceID = types.Int64Value(int64(*service.ID))
	//TODO(TF3.0) make connection_host_name, service_attachment_name and status nullable
	data.ConnectionHostName = types.StringValue(redis.StringValue(service.ConnectionHostName))
	data.ServiceAttachmentName = types.StringValue(redis.StringValue(service.ServiceAttachmentName))
	data.Status = types.StringValue(redis.StringValue(service.Status))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
