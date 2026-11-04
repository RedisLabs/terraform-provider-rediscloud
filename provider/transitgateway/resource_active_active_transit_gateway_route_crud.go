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

func (r *activeActiveTransitGatewayRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ActiveActiveTransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, transitGatewayRouteDefaultCreateTimeout)
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

	resp.Diagnostics.Append(r.updateCidrs(ctx, subId, regionId, tgwId, plan.Cidrs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(BuildActiveActiveTransitGatewayAttachmentId(subId, regionId, tgwId))

	plannedCidrs := plan.Cidrs
	found, diags := r.readInto(ctx, &plan, createTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Transit Gateway Not Found", fmt.Sprintf("Transit Gateway %d in subscription %d region %d was not found after updating its CIDRs", tgwId, subId, regionId))
		return
	}
	// The API may return the CIDRs in another order; the framework rejects an apply result that differs from the plan.
	plan.Cidrs = plannedCidrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveTransitGatewayRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ActiveActiveTransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, transitGatewayRouteDefaultReadTimeout)
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

func (r *activeActiveTransitGatewayRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ActiveActiveTransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, transitGatewayRouteDefaultUpdateTimeout)
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
	regionId, err := strconv.Atoi(plan.RegionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid region_id", err.Error())
		return
	}
	tgwId := int(plan.TgwID.ValueInt64())

	resp.Diagnostics.Append(r.updateCidrs(ctx, subId, regionId, tgwId, plan.Cidrs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plannedCidrs := plan.Cidrs
	found, diags := r.readInto(ctx, &plan, updateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Transit Gateway Not Found", fmt.Sprintf("Transit Gateway %d in subscription %d region %d was not found after updating its CIDRs", tgwId, subId, regionId))
		return
	}
	// The API may return the CIDRs in another order; the framework rejects an apply result that differs from the plan.
	plan.Cidrs = plannedCidrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveTransitGatewayRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ActiveActiveTransitGatewayRouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, transitGatewayRouteDefaultDeleteTimeout)
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
	regionId, err := strconv.Atoi(state.RegionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid region_id", err.Error())
		return
	}
	tgwId := int(state.TgwID.ValueInt64())

	err = r.client.Client.TransitGatewayAttachments.UpdateActiveActive(ctx, subId, regionId, tgwId, []*string{})
	if err != nil && !isTransitGatewayAttachmentNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Transit Gateway Route", err.Error())
	}
}

func (r *activeActiveTransitGatewayRouteResource) updateCidrs(ctx context.Context, subId, regionId, tgwId int, cidrs types.List) diag.Diagnostics {
	apiCidrs, diags := cidrsToAPI(ctx, cidrs)
	if diags.HasError() {
		return diags
	}

	if err := r.client.Client.TransitGatewayAttachments.UpdateActiveActive(ctx, subId, regionId, tgwId, apiCidrs); err != nil {
		diags.AddError("Error Updating Transit Gateway Route", err.Error())
	}
	return diags
}

// readInto refreshes model from the API, using the ID as the source of truth like the SDKv2 implementation, and reports whether the attachment still exists.
func (r *activeActiveTransitGatewayRouteResource) readInto(ctx context.Context, model *ActiveActiveTransitGatewayRouteResourceModel, timeout time.Duration) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	subId, regionId, tgwId, err := ParseActiveActiveTransitGatewayAttachmentId(model.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid Transit Gateway Route ID", err.Error())
		return false, diags
	}

	model.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	model.RegionID = types.StringValue(strconv.Itoa(regionId))
	model.TgwID = types.Int64Value(int64(tgwId))

	tgwTask, err := utils.WaitForActiveActiveTransitGatewayResourceToBeAvailable(ctx, subId, regionId, r.client, timeout)
	if err != nil {
		diags.AddError("Error Reading Transit Gateway Route", err.Error())
		return false, diags
	}

	tgw, findDiags := findTransitGatewayAttachment(tgwTask, tgwId, fmt.Sprintf("subscription_id/region_id/tgw_id: %d/%d/%d", subId, regionId, tgwId))
	diags.Append(findDiags...)
	if diags.HasError() || tgw == nil {
		return false, diags
	}

	cidrs, cidrDiags := CidrsFromAPI(ctx, tgw.Cidrs)
	diags.Append(cidrDiags...)
	model.Cidrs = cidrs
	return true, diags
}
