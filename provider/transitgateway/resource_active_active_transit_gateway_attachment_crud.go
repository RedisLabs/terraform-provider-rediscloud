package transitgateway

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func (r *activeActiveTransitGatewayAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ActiveActiveTransitGatewayAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, transitGatewayAttachmentDefaultCreateTimeout)
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
	regionId, err := strconv.Atoi(plan.RegionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid region_id", err.Error())
		return
	}
	tgwId := int(plan.TgwID.ValueInt64())

	if len(plan.Cidrs.Elements()) > 0 {
		resp.Diagnostics.AddError("Unable to Create Transit Gateway Attachment", transitGatewayAttachmentCreateWithCidrsError)
		return
	}

	if _, err := r.client.Client.TransitGatewayAttachments.CreateActiveActive(ctx, subId, regionId, tgwId); err != nil {
		resp.Diagnostics.AddError("Error Creating Transit Gateway Attachment", err.Error())
		return
	}

	plan.ID = types.StringValue(BuildActiveActiveTransitGatewayAttachmentId(subId, regionId, tgwId))

	if _, err := utils.WaitForActiveActiveTransitGatewayAttachmentToBeAvailable(ctx, subId, regionId, tgwId, r.client, createTimeout); err != nil {
		plan.computed().setUnknownToNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Error Waiting for Transit Gateway Attachment", err.Error())
		return
	}

	planned := plan
	found, diags := r.readInto(ctx, &plan, createTimeout)
	resp.Diagnostics.Append(diags...)
	if !found && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Transit Gateway Attachment Not Found", fmt.Sprintf("Transit Gateway attachment %d in subscription %d region %d was not found after creation", tgwId, subId, regionId))
	}
	if resp.Diagnostics.HasError() {
		plan.computed().setUnknownToNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	plan.SubscriptionID = planned.SubscriptionID
	plan.RegionID = planned.RegionID
	plan.TgwID = planned.TgwID
	plan.computed().keepKnownPlanValues(planned.computed())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveTransitGatewayAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ActiveActiveTransitGatewayAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, transitGatewayAttachmentDefaultReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	found, diags := r.readInto(ctx, &state, readTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *activeActiveTransitGatewayAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ActiveActiveTransitGatewayAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, transitGatewayAttachmentDefaultUpdateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	subId, regionId, tgwId, err := ParseActiveActiveTransitGatewayAttachmentId(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Transit Gateway Attachment ID", err.Error())
		return
	}

	cidrs, diags := cidrsToAPI(ctx, plan.Cidrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Client.TransitGatewayAttachments.UpdateActiveActive(ctx, subId, regionId, tgwId, cidrs); err != nil {
		resp.Diagnostics.AddError("Error Updating Transit Gateway Attachment", err.Error())
		return
	}

	planned := plan
	found, diags := r.readInto(ctx, &plan, updateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Transit Gateway Attachment Not Found", fmt.Sprintf("Transit Gateway attachment %s was not found after update", plan.ID.ValueString()))
		return
	}

	plan.SubscriptionID = planned.SubscriptionID
	plan.RegionID = planned.RegionID
	plan.TgwID = planned.TgwID
	plan.computed().keepKnownPlanValues(planned.computed())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveTransitGatewayAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ActiveActiveTransitGatewayAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, transitGatewayAttachmentDefaultDeleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	subId, regionId, tgwId, err := ParseActiveActiveTransitGatewayAttachmentId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Transit Gateway Attachment ID", err.Error())
		return
	}

	err = r.client.Client.TransitGatewayAttachments.DeleteActiveActive(ctx, subId, regionId, tgwId)
	if err != nil && !isTransitGatewayAttachmentNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Transit Gateway Attachment", err.Error())
	}
}

// readInto refreshes model from the API, using the ID as the source of truth like the SDKv2 implementation, and reports whether the attachment still exists.
func (r *activeActiveTransitGatewayAttachmentResource) readInto(ctx context.Context, model *ActiveActiveTransitGatewayAttachmentResourceModel, timeout time.Duration) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	subId, regionId, tgwId, err := ParseActiveActiveTransitGatewayAttachmentId(model.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid Transit Gateway Attachment ID", err.Error())
		return false, diags
	}

	model.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	model.RegionID = types.StringValue(strconv.Itoa(regionId))
	model.TgwID = types.Int64Value(int64(tgwId))

	tgwTask, err := utils.WaitForActiveActiveTransitGatewayResourceToBeAvailable(ctx, subId, regionId, r.client, timeout)
	if err != nil {
		diags.AddError("Error Reading Transit Gateway Attachment", err.Error())
		return false, diags
	}

	tgw, findDiags := findTransitGatewayAttachment(tgwTask, tgwId, fmt.Sprintf("subscription_id/tgw_id: %d/%d", subId, tgwId))
	diags.Append(findDiags...)
	if diags.HasError() || tgw == nil {
		return false, diags
	}

	diags.Append(model.computed().setFromAPI(ctx, tgw)...)
	return true, diags
}
