package psc

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pscapi "github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const endpointDefaultTimeout = 10 * time.Minute

func (r *privateServiceConnectEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PrivateServiceConnectEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, endpointDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	subId, err := strconv.Atoi(plan.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}
	pscServiceId := int(plan.PrivateServiceConnectServiceID.ValueInt64())

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	endpointId, err := r.client.Client.PrivateServiceConnect.CreateEndpoint(ctx, subId, pscServiceId, pscapi.CreatePrivateServiceConnectEndpoint{
		GCPProjectID:           plan.GCPProjectID.ValueStringPointer(),
		GCPVPCName:             plan.GCPVPCName.ValueStringPointer(),
		GCPVPCSubnetName:       plan.GCPVPCSubnetName.ValueStringPointer(),
		EndpointConnectionName: plan.EndpointConnectionName.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Private Service Connect endpoint", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%d/%d/%d", subId, pscServiceId, endpointId))
	plan.PrivateServiceConnectEndpointID = types.Int64Value(int64(endpointId))

	// The endpoint exists from here on, so failures keep it in state like the
	// SDKv2 implementation did by setting the ID before waiting.
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, createTimeout); err != nil {
		plan.ServiceAttachments = types.ListNull(serviceAttachmentElemType)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Failed to create Private Service Connect endpoint", err.Error())
		return
	}

	endpoint, attachments, err := readEndpoint(ctx, r.endpointsGetter(subId, pscServiceId), r.creationScriptGetter(subId, pscServiceId), endpointId)
	if err != nil {
		plan.ServiceAttachments = types.ListNull(serviceAttachmentElemType)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Failed to read Private Service Connect endpoint", err.Error())
		return
	}
	if endpoint == nil {
		resp.Diagnostics.AddError("Private Service Connect endpoint not found", "The endpoint was not found immediately after creation.")
		return
	}

	plan.ServiceAttachments = attachments
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateServiceConnectEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PrivateServiceConnectEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, endpointDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	ids, err := parseIntIdParts(state.ID.ValueString(), 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Private Service Connect endpoint ID", err.Error())
		return
	}
	subId, pscServiceId, endpointId := ids[0], ids[1], ids[2]

	endpoint, attachments, err := readEndpoint(ctx, r.endpointsGetter(subId, pscServiceId), r.creationScriptGetter(subId, pscServiceId), endpointId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Private Service Connect endpoint", err.Error())
		return
	}
	if endpoint == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%d/%d/%d", subId, pscServiceId, redis.IntValue(endpoint.ID)))
	state.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	state.PrivateServiceConnectServiceID = types.Int64Value(int64(pscServiceId))
	state.PrivateServiceConnectEndpointID = types.Int64Value(int64(redis.IntValue(endpoint.ID)))
	state.GCPProjectID = types.StringPointerValue(endpoint.GCPProjectID)
	state.GCPVPCName = types.StringPointerValue(endpoint.GCPVPCName)
	state.GCPVPCSubnetName = types.StringPointerValue(endpoint.GCPVPCSubnetName)
	state.EndpointConnectionName = types.StringPointerValue(endpoint.EndpointConnectionName)
	state.ServiceAttachments = attachments

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only runs for timeouts changes, because every other argument requires replacement.
func (r *privateServiceConnectEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PrivateServiceConnectEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateServiceConnectEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PrivateServiceConnectEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, endpointDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	ids, err := parseIntIdParts(state.ID.ValueString(), 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Private Service Connect endpoint ID", err.Error())
		return
	}
	subId, pscServiceId, endpointId := ids[0], ids[1], ids[2]

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	getEndpoints := r.endpointsGetter(subId, pscServiceId)
	endpoints, err := getEndpoints(ctx)
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to read Private Service Connect endpoint", err.Error())
		return
	}

	endpoint := findEndpoint(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return
	}

	// The API only deletes endpoints in the initialized status.
	if redis.StringValue(endpoint.Status) == pscapi.EndpointStatusInitialized {
		if err := r.client.Client.PrivateServiceConnect.DeleteEndpoint(ctx, subId, pscServiceId, endpointId); err != nil {
			resp.Diagnostics.AddError("Failed to delete Private Service Connect endpoint", err.Error())
		}
		return
	}

	if err := waitForEndpointToDisappear(ctx, r.client, getEndpoints, endpointId, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("Failed to delete Private Service Connect endpoint", err.Error())
	}
}

func (r *privateServiceConnectEndpointResource) endpointsGetter(subId, pscServiceId int) endpointsGetter {
	return func(ctx context.Context) (*pscapi.PrivateServiceConnectEndpoints, error) {
		return r.client.Client.PrivateServiceConnect.GetEndpoints(ctx, subId, pscServiceId)
	}
}

func (r *privateServiceConnectEndpointResource) creationScriptGetter(subId, pscServiceId int) creationScriptGetter {
	return func(ctx context.Context, endpointId int) (*pscapi.CreationScript, error) {
		return r.client.Client.PrivateServiceConnect.GetEndpointCreationScripts(ctx, subId, pscServiceId, endpointId, true)
	}
}
