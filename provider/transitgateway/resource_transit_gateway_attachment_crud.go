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

const (
	transitGatewayAttachmentDefaultCreateTimeout = 5 * time.Minute
	transitGatewayAttachmentDefaultReadTimeout   = 3 * time.Minute
	transitGatewayAttachmentDefaultUpdateTimeout = 5 * time.Minute
	transitGatewayAttachmentDefaultDeleteTimeout = 5 * time.Minute

	transitGatewayAttachmentCreateWithCidrsError = "Attachment cannot be created with Cidrs provided, it must be accepted first. This resource may then be updated with Cidrs."
)

func (r *transitGatewayAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransitGatewayAttachmentResourceModel
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
	tgwId := int(plan.TgwID.ValueInt64())

	if len(plan.Cidrs.Elements()) > 0 {
		resp.Diagnostics.AddError("Unable to Create Transit Gateway Attachment", transitGatewayAttachmentCreateWithCidrsError)
		return
	}

	if _, err := r.client.Client.TransitGatewayAttachments.Create(ctx, subId, tgwId); err != nil {
		resp.Diagnostics.AddError("Error Creating Transit Gateway Attachment", err.Error())
		return
	}

	plan.ID = types.StringValue(utils.BuildResourceId(subId, tgwId))

	if _, err := utils.WaitForTransitGatewayAttachmentToBeAvailable(ctx, subId, tgwId, r.client, createTimeout); err != nil {
		plan.computed().setUnknownToNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Error Waiting for Transit Gateway Attachment", err.Error())
		return
	}

	planned := plan
	found, diags := r.readInto(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if !found && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Transit Gateway Attachment Not Found", fmt.Sprintf("Transit Gateway attachment %d in subscription %d was not found after creation", tgwId, subId))
	}
	if resp.Diagnostics.HasError() {
		plan.computed().setUnknownToNull()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	plan.SubscriptionID = planned.SubscriptionID
	plan.TgwID = planned.TgwID
	plan.computed().keepKnownPlanValues(planned.computed())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transitGatewayAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransitGatewayAttachmentResourceModel
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

	found, diags := r.readInto(ctx, &state)
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

func (r *transitGatewayAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TransitGatewayAttachmentResourceModel
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

	subId, err := strconv.Atoi(plan.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}
	tgwId := int(plan.TgwID.ValueInt64())

	cidrs, diags := cidrsToAPI(ctx, plan.Cidrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Client.TransitGatewayAttachments.Update(ctx, subId, tgwId, cidrs); err != nil {
		resp.Diagnostics.AddError("Error Updating Transit Gateway Attachment", err.Error())
		return
	}

	planned := plan
	found, diags := r.readInto(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Transit Gateway Attachment Not Found", fmt.Sprintf("Transit Gateway attachment %s was not found after update", plan.ID.ValueString()))
		return
	}

	plan.SubscriptionID = planned.SubscriptionID
	plan.TgwID = planned.TgwID
	plan.computed().keepKnownPlanValues(planned.computed())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transitGatewayAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransitGatewayAttachmentResourceModel
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

	subId, err := strconv.Atoi(state.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}
	tgwId := int(state.TgwID.ValueInt64())

	err = r.client.Client.TransitGatewayAttachments.Delete(ctx, subId, tgwId)
	if err != nil && !isTransitGatewayAttachmentNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Transit Gateway Attachment", err.Error())
	}
}

// readInto refreshes model from the API, using the ID as the source of truth like the SDKv2 implementation, and reports whether the attachment still exists.
func (r *transitGatewayAttachmentResource) readInto(ctx context.Context, model *TransitGatewayAttachmentResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	subId, tgwId, err := ParseTransitGatewayAttachmentId(model.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid Transit Gateway Attachment ID", err.Error())
		return false, diags
	}

	model.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	model.TgwID = types.Int64Value(int64(tgwId))

	tgwTask, err := r.client.Client.TransitGatewayAttachments.Get(ctx, subId)
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
