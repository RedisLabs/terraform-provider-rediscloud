package psc

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pscapi "github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *privateServiceConnectEndpointsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PrivateServiceConnectEndpointsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(data.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}
	pscServiceId := int(data.PrivateServiceConnectServiceID.ValueInt64())

	endpoints, err := d.client.Client.PrivateServiceConnect.GetEndpoints(ctx, subId, pscServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Private Service Connect endpoints", err.Error())
		return
	}

	serviceAttachments := map[int][]pscapi.TerraformGCPServiceAttachment{}
	for _, endpoint := range endpoints.Endpoints {
		endpointId := redis.IntValue(endpoint.ID)
		status := redis.StringValue(endpoint.Status)
		if status == pscapi.EndpointStatusRejected || status == pscapi.EndpointStatusDeleted {
			continue
		}
		script, err := d.client.Client.PrivateServiceConnect.GetEndpointCreationScripts(ctx, subId, pscServiceId, endpointId, true)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read Private Service Connect endpoint creation script", err.Error())
			return
		}
		serviceAttachments[endpointId] = script.Script.TerraformGcp.ServiceAttachments
	}

	var diags diag.Diagnostics
	data.Endpoints, diags = EndpointsFromAPI(ctx, endpoints.Endpoints, serviceAttachments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(buildPrivateServiceConnectId(subId, pscServiceId))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// EndpointsFromAPI maps endpoints and their service attachments as the SDKv2 implementation did, writing "" for missing fields.
func EndpointsFromAPI(ctx context.Context, endpoints []*pscapi.PrivateServiceConnectEndpoint, serviceAttachments map[int][]pscapi.TerraformGCPServiceAttachment) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	models := make([]EndpointModel, 0, len(endpoints))
	for _, endpoint := range endpoints {
		attachments, d := ServiceAttachmentsFromAPI(ctx, serviceAttachments[redis.IntValue(endpoint.ID)])
		diags.Append(d...)
		//TODO(TF3.0) make endpoint fields nullable
		models = append(models, EndpointModel{
			PrivateServiceConnectEndpointID: types.Int64Value(int64(redis.IntValue(endpoint.ID))),
			GCPProjectID:                    types.StringValue(redis.StringValue(endpoint.GCPProjectID)),
			GCPVPCName:                      types.StringValue(redis.StringValue(endpoint.GCPVPCName)),
			GCPVPCSubnetName:                types.StringValue(redis.StringValue(endpoint.GCPVPCSubnetName)),
			EndpointConnectionName:          types.StringValue(redis.StringValue(endpoint.EndpointConnectionName)),
			Status:                          types.StringValue(redis.StringValue(endpoint.Status)),
			ServiceAttachments:              attachments,
		})
	}
	if diags.HasError() {
		return types.ListNull(endpointElemType), diags
	}
	list, d := types.ListValueFrom(ctx, endpointElemType, models)
	diags.Append(d...)
	return list, diags
}
