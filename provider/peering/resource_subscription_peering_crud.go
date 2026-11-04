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

const peeringDefaultTimeout = 10 * time.Minute

func (r *subscriptionPeeringResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SubscriptionPeeringResourceModel
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
	peeringRequest := subscriptions.CreateVPCPeering{}

	if providerName == providerAWS {
		if !isSet(plan.Region) {
			resp.Diagnostics.AddError("Missing region", "`region` must be set when `provider_name` is `AWS`")
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

		peeringRequest.Region = plan.Region.ValueStringPointer()
		peeringRequest.AWSAccountID = plan.AWSAccountID.ValueStringPointer()
		peeringRequest.VPCId = plan.VPCID.ValueStringPointer()
	}

	if providerName == providerGCP {
		if !isSet(plan.GCPProjectID) {
			resp.Diagnostics.AddError("Missing gcp_project_id", "`gcp_project_id` must be set when `provider_name` is `GCP`")
			return
		}
		if !isSet(plan.GCPNetworkName) {
			resp.Diagnostics.AddError("Missing gcp_network_name", "`network_name` must be set when `provider_name` is `GCP`")
			return
		}

		peeringRequest.Provider = redis.String(strings.ToLower(providerName))
		peeringRequest.VPCProjectUID = plan.GCPProjectID.ValueStringPointer()
		peeringRequest.VPCNetworkName = plan.GCPNetworkName.ValueStringPointer()
	}

	peeringId, err := r.client.Client.Subscription.CreateVPCPeering(ctx, subId, peeringRequest)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create subscription peering", err.Error())
		return
	}

	plan.ID = types.StringValue(utils.BuildResourceId(subId, peeringId))

	lastStatus, err := waitForPeeringToBeInitiated(ctx, subId, peeringId, r.client, createTimeout)
	if err != nil {
		plan.Status = lastStatus
		plan.nullUnknowns()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Subscription peering failed to be initiated", err.Error())
		return
	}

	notFound, diags := r.readPeeringIntoModel(ctx, subId, peeringId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.Diagnostics.AddError("Subscription peering not found", "The peering was not found immediately after creation.")
		return
	}

	plan.nullUnknowns()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriptionPeeringResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SubscriptionPeeringResourceModel
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
		resp.Diagnostics.AddError("Invalid subscription peering ID", err.Error())
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

// Update only persists timeouts: every other configurable attribute forces replacement.
func (r *subscriptionPeeringResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SubscriptionPeeringResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = plan.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subscriptionPeeringResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SubscriptionPeeringResourceModel
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
		resp.Diagnostics.AddError("Invalid subscription peering ID", err.Error())
		return
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := r.client.Client.Subscription.DeleteVPCPeering(ctx, subId, peeringId); err != nil {
		notFound := &subscriptions.NotFound{}
		if errors.As(err, &notFound) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete subscription peering", err.Error())
	}
}

func (r *subscriptionPeeringResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
func (r *subscriptionPeeringResource) readPeeringIntoModel(ctx context.Context, subId, peeringId int, model *SubscriptionPeeringResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	model.SubscriptionID = types.StringValue(strconv.Itoa(subId))

	peerings, err := r.client.Client.Subscription.ListVPCPeering(ctx, subId)
	if err != nil {
		notFound := &subscriptions.NotFound{}
		if errors.As(err, &notFound) {
			return true, diags
		}
		diags.AddError("Failed to read subscription peering", err.Error())
		return false, diags
	}

	peering := findVPCPeering(peeringId, peerings)
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
		model.Region = types.StringValue(redis.StringValue(peering.Region))

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
		model.GCPRedisProjectID = types.StringValue(redis.StringValue(peering.RedisProjectUID))
		model.GCPRedisNetworkName = types.StringValue(redis.StringValue(peering.RedisNetworkName))
		model.GCPPeeringID = types.StringValue(redis.StringValue(peering.CloudPeeringID))
	}

	return false, diags
}

// waitForPeeringToBeInitiated returns the last observed status alongside any
// error so a failed wait can still persist it.
func waitForPeeringToBeInitiated(ctx context.Context, subId, id int, api *client.ApiClient, timeout time.Duration) (types.String, error) {
	lastStatus := types.StringNull()
	wait := &retry.StateChangeConf{
		PollInterval: api.WaitPollInterval(0),
		Delay:        api.WaitDelay(10 * time.Second),
		Pending: []string{
			subscriptions.VPCPeeringStatusInitiatingRequest,
		},
		Target: []string{
			subscriptions.VPCPeeringStatusActive,
			subscriptions.VPCPeeringStatusInactive,
			subscriptions.VPCPeeringStatusPendingAcceptance,
		},
		Timeout: timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for vpc peering %d to be initiated", id)

			list, err := api.Client.Subscription.ListVPCPeering(ctx, subId)
			if err != nil {
				return nil, "", err
			}

			peering := findVPCPeering(id, list)
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
