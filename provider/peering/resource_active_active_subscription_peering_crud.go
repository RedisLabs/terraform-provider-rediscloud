package peering

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func (r *activeActiveSubscriptionPeeringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ActiveActiveSubscriptionPeeringResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, peeringDefaultTimeout)
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

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	providerName := plan.ProviderName.ValueString()
	peeringRequest := subscriptions.CreateActiveActiveVPCPeering{}

	if providerName == providerAWS {
		if !isSet(plan.SourceRegion) {
			resp.Diagnostics.AddError("Missing source_region", "`source_region` must be set when `provider_name` is `AWS`")
			return
		}
		if !isSet(plan.DestinationRegion) {
			resp.Diagnostics.AddError("Missing destination_region", "`destination_region` must be set when `provider_name` is `AWS`")
			return
		}
		if !isSet(plan.AWSAccountID) {
			resp.Diagnostics.AddError("Missing aws_account_id", "`aws_account_id` must be set when `provider_name` is `AWS`")
			return
		}
		if !isSet(plan.VPCID) {
			resp.Diagnostics.AddError("Missing vpc_id", "`vpc_id` must be set when `provider_name` is `AWS`")
			return
		}

		cidrs, diags := vpcCidrsFromPlan(ctx, plan.VPCCidrs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if isSet(plan.VPCCidr) {
			peeringRequest.VPCCidr = plan.VPCCidr.ValueStringPointer()
		} else if len(cidrs) > 0 {
			peeringRequest.VPCCidrs = cidrs
		} else {
			resp.Diagnostics.AddError("Missing vpc_cidr", "`vpc_cidr` or `vpc_cidrs` must be set when `provider_name` is `AWS`")
			return
		}

		peeringRequest.SourceRegion = plan.SourceRegion.ValueStringPointer()
		peeringRequest.DestinationRegion = plan.DestinationRegion.ValueStringPointer()
		peeringRequest.AWSAccountID = plan.AWSAccountID.ValueStringPointer()
		peeringRequest.VPCId = plan.VPCID.ValueStringPointer()
	}

	if providerName == providerGCP {
		if !isSet(plan.GCPProjectID) {
			resp.Diagnostics.AddError("Missing gcp_project_id", "`gcp_project_id` must be set when `provider_name` is `GCP`")
			return
		}
		if !isSet(plan.GCPNetworkName) {
			resp.Diagnostics.AddError("Missing gcp_network_name", "`gcp_network_name` must be set when `provider_name` is `GCP`")
			return
		}
		if !isSet(plan.SourceRegion) {
			resp.Diagnostics.AddError("Missing source_region", "`source_region` must be set when `provider_name` is `GCP`")
			return
		}

		peeringRequest.Provider = redis.String(strings.ToLower(providerName))
		peeringRequest.SourceRegion = plan.SourceRegion.ValueStringPointer()
		peeringRequest.VPCProjectUID = plan.GCPProjectID.ValueStringPointer()
		peeringRequest.VPCNetworkName = plan.GCPNetworkName.ValueStringPointer()
	}

	peeringId, err := r.client.Client.Subscription.CreateActiveActiveVPCPeering(ctx, subId, peeringRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Active-Active subscription peering", err.Error())
		return
	}

	plan.ID = types.StringValue(utils.BuildResourceId(subId, peeringId))

	lastStatus, err := waitForActiveActivePeeringToBeInitiated(ctx, subId, peeringId, r.client, createTimeout)
	if err != nil {
		plan.Status = lastStatus
		plan.nullUnknowns()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Active-Active subscription peering failed to be initiated", err.Error())
		return
	}

	notFound, diags := r.readPeeringIntoModel(ctx, subId, peeringId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.Diagnostics.AddError("Active-Active subscription peering not found", "The peering was not found immediately after creation.")
		return
	}

	plan.nullUnknowns()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveSubscriptionPeeringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ActiveActiveSubscriptionPeeringResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, peeringDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	subId, peeringId, err := parsePeeringID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Active-Active subscription peering ID", err.Error())
		return
	}

	notFound, diags := r.readPeeringIntoModel(ctx, subId, peeringId, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ModifyPlan rejects a change to gcp_project_id or gcp_network_name. They do
// not force replacement and cannot be updated, so SDKv2 failed at apply time;
// failing at plan time keeps the same schema and surfaces the error earlier.
func (r *activeActiveSubscriptionPeeringResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	for _, name := range []string{"gcp_project_id", "gcp_network_name"} {
		var config, state types.String
		diags := req.Config.GetAttribute(ctx, path.Root(name), &config)
		diags.Append(req.State.GetAttribute(ctx, path.Root(name), &state)...)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		if config.IsNull() || config.IsUnknown() || config.Equal(state) {
			continue
		}
		resp.Diagnostics.AddAttributeError(
			path.Root(name),
			"doesn't support update",
			fmt.Sprintf("`%s` cannot be changed on an existing Active-Active subscription peering.", name),
		)
	}
}

// Update only persists timeouts: ModifyPlan rejects every other change.
func (r *activeActiveSubscriptionPeeringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ActiveActiveSubscriptionPeeringResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = plan.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *activeActiveSubscriptionPeeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ActiveActiveSubscriptionPeeringResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, peeringDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	subId, peeringId, err := parsePeeringID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Active-Active subscription peering ID", err.Error())
		return
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := r.client.Client.Subscription.DeleteActiveActiveVPCPeering(ctx, subId, peeringId); err != nil {
		notFound := &subscriptions.NotFound{}
		if errors.As(err, &notFound) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete Active-Active subscription peering", err.Error())
	}
}

func (r *activeActiveSubscriptionPeeringResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, _, err := parsePeeringID(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected an ID of the form <subscription_id>/<peering_id>, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readPeeringIntoModel maps the API peering onto the model. As in SDKv2, only
// the attributes of the detected cloud provider are written; the others keep
// their current value. It returns true when the subscription or peering no
// longer exists.
func (r *activeActiveSubscriptionPeeringResource) readPeeringIntoModel(ctx context.Context, subId, peeringId int, model *ActiveActiveSubscriptionPeeringResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	model.SubscriptionID = types.StringValue(strconv.Itoa(subId))

	regions, err := r.client.Client.Subscription.ListActiveActiveVPCPeering(ctx, subId)
	if err != nil {
		notFound := &subscriptions.NotFound{}
		if errors.As(err, &notFound) {
			return true, diags
		}
		diags.AddError("Failed to read Active-Active subscription peering", err.Error())
		return false, diags
	}

	peering, sourceRegion := findActiveActiveVPCPeering(peeringId, regions)
	if peering == nil {
		return true, diags
	}

	model.Status = types.StringValue(redis.StringValue(peering.Status))

	providerName := providerAWS
	if redis.StringValue(peering.GCPProjectUID) != "" {
		providerName = providerGCP
	}
	model.ProviderName = types.StringValue(providerName)

	//TODO(TF3.0) make the peering string fields nullable
	if providerName == providerAWS {
		model.AWSAccountID = types.StringValue(redis.StringValue(peering.AWSAccountID))
		model.AWSPeeringID = types.StringValue(redis.StringValue(peering.AWSPeeringID))
		model.VPCID = types.StringValue(redis.StringValue(peering.VPCId))
		model.SourceRegion = types.StringValue(redis.StringValue(sourceRegion))
		model.DestinationRegion = types.StringValue(redis.StringValue(peering.RegionName))

		vpcCidr, vpcCidrs, cidrDiags := vpcCidrsFromAPI(ctx, peering.VPCCidr, peering.VPCCidrs)
		diags.Append(cidrDiags...)
		if diags.HasError() {
			return false, diags
		}
		model.VPCCidr = vpcCidr
		model.VPCCidrs = vpcCidrs
	}
	if providerName == providerGCP {
		emptySetIfUnset(&model.VPCCidrs)
		model.GCPProjectID = types.StringValue(redis.StringValue(peering.GCPProjectUID))
		model.GCPNetworkName = types.StringValue(redis.StringValue(peering.NetworkName))
		model.SourceRegion = types.StringValue(redis.StringValue(sourceRegion))
		model.GCPRedisProjectID = types.StringValue(redis.StringValue(peering.RedisProjectUID))
		model.GCPRedisNetworkName = types.StringValue(redis.StringValue(peering.RedisNetworkName))
		model.GCPPeeringID = types.StringValue(redis.StringValue(peering.CloudPeeringID))
	}

	return false, diags
}

func findActiveActiveVPCPeering(id int, regions []*subscriptions.ActiveActiveVpcRegion) (*subscriptions.ActiveActiveVPCPeering, *string) {
	for _, region := range regions {
		for _, peering := range region.VPCPeerings {
			if redis.IntValue(peering.ID) == id {
				return peering, region.SourceRegion
			}
		}
	}
	return nil, nil
}

// waitForActiveActivePeeringToBeInitiated returns the last observed status
// alongside any error so a failed wait can still persist it.
func waitForActiveActivePeeringToBeInitiated(ctx context.Context, subId, id int, api *client.ApiClient, timeout time.Duration) (types.String, error) {
	lastStatus := types.StringNull()
	wait := &retry.StateChangeConf{
		Delay: api.WaitDelay(30 * time.Second),
		Pending: []string{
			subscriptions.VPCPeeringStatusInitiatingRequest,
		},
		Target: []string{
			subscriptions.VPCPeeringStatusActive,
			subscriptions.VPCPeeringStatusInactive,
			subscriptions.VPCPeeringStatusPendingAcceptance,
		},
		Timeout:      timeout,
		PollInterval: api.WaitPollInterval(30 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for vpc peering %d to be initiated. Status: %s", id, lastStatus.ValueString())

			list, err := api.Client.Subscription.ListActiveActiveVPCPeering(ctx, subId)
			if err != nil {
				return nil, "", err
			}

			peering, _ := findActiveActiveVPCPeering(id, list)
			if peering == nil {
				log.Printf("Peering %d/%d not present yet", subId, id)
				return nil, "", nil
			}

			lastStatus = types.StringValue(redis.StringValue(peering.Status))
			return lastStatus.ValueString(), lastStatus.ValueString(), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return lastStatus, err
}
