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
	transitGatewayRouteDefaultCreateTimeout = 5 * time.Minute
	transitGatewayRouteDefaultReadTimeout   = 3 * time.Minute
	transitGatewayRouteDefaultUpdateTimeout = 5 * time.Minute
	transitGatewayRouteDefaultDeleteTimeout = 5 * time.Minute
)

func (r *transitGatewayRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransitGatewayRouteResourceModel
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
	tgwId := int(plan.TgwID.ValueInt64())

	resp.Diagnostics.Append(r.updateCidrs(ctx, subId, tgwId, plan.Cidrs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(utils.BuildResourceId(subId, tgwId))

	plannedCidrs := plan.Cidrs
	found, diags := r.readInto(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Transit Gateway Not Found", fmt.Sprintf("Transit Gateway %d in subscription %d was not found after updating its CIDRs", tgwId, subId))
		return
	}
	// The API may return the CIDRs in another order; the framework rejects an apply result that differs from the plan.
	plan.Cidrs = plannedCidrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transitGatewayRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransitGatewayRouteResourceModel
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

func (r *transitGatewayRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TransitGatewayRouteResourceModel
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
	tgwId := int(plan.TgwID.ValueInt64())

	resp.Diagnostics.Append(r.updateCidrs(ctx, subId, tgwId, plan.Cidrs)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plannedCidrs := plan.Cidrs
	found, diags := r.readInto(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Transit Gateway Not Found", fmt.Sprintf("Transit Gateway %d in subscription %d was not found after updating its CIDRs", tgwId, subId))
		return
	}
	// The API may return the CIDRs in another order; the framework rejects an apply result that differs from the plan.
	plan.Cidrs = plannedCidrs

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *transitGatewayRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransitGatewayRouteResourceModel
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
	tgwId := int(state.TgwID.ValueInt64())

	err = r.client.Client.TransitGatewayAttachments.Update(ctx, subId, tgwId, []*string{})
	if err != nil && !isTransitGatewayAttachmentNotFound(err) {
		resp.Diagnostics.AddError("Error Deleting Transit Gateway Route", err.Error())
	}
}

func (r *transitGatewayRouteResource) updateCidrs(ctx context.Context, subId, tgwId int, cidrs types.List) diag.Diagnostics {
	apiCidrs, diags := cidrsToAPI(ctx, cidrs)
	if diags.HasError() {
		return diags
	}

	if err := r.client.Client.TransitGatewayAttachments.Update(ctx, subId, tgwId, apiCidrs); err != nil {
		diags.AddError("Error Updating Transit Gateway Route", err.Error())
	}
	return diags
}

// readInto refreshes model from the API, using the ID as the source of truth like the SDKv2 implementation, and reports whether the attachment still exists.
func (r *transitGatewayRouteResource) readInto(ctx context.Context, model *TransitGatewayRouteResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	subId, tgwId, err := ParseTransitGatewayAttachmentId(model.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid Transit Gateway Route ID", err.Error())
		return false, diags
	}

	model.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	model.TgwID = types.Int64Value(int64(tgwId))

	tgwTask, err := r.client.Client.TransitGatewayAttachments.Get(ctx, subId)
	if err != nil {
		diags.AddError("Error Reading Transit Gateway Route", err.Error())
		return false, diags
	}

	tgw, findDiags := findTransitGatewayAttachment(tgwTask, tgwId, fmt.Sprintf("subscription_id/tgw_id: %d/%d", subId, tgwId))
	diags.Append(findDiags...)
	if diags.HasError() || tgw == nil {
		return false, diags
	}

	cidrs, cidrDiags := CidrsFromAPI(ctx, tgw.Cidrs)
	diags.Append(cidrDiags...)
	model.Cidrs = cidrs
	return true, diags
}
