package privatelink

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func (r *activeActivePrivateLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ActiveActivePrivateLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, privateLinkCreateTimeout)
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
	regionId := int(plan.RegionID.ValueInt64())

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	principals, diags := principalsFromPlan(ctx, plan.Principal)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	firstPrincipal := principals[0]

	err = r.client.Client.PrivateLink.CreateActiveActivePrivateLink(ctx, subId, regionId, pl.CreatePrivateLink{
		Principal:      firstPrincipal.Principal,
		PrincipalType:  firstPrincipal.Type,
		PrincipalAlias: firstPrincipal.Alias,
		ShareName:      plan.ShareName.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create PrivateLink", err.Error())
		return
	}

	plan.ID = types.StringValue(makeActiveActivePrivateLinkId(subId, regionId))

	if err := r.finishCreate(ctx, subId, regionId, principals, createTimeout); err != nil {
		// The PrivateLink exists at this point, so keep it in state like the SDKv2
		// implementation did by setting the ID before waiting.
		plan.nullUnknownComputed()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Failed to create PrivateLink", err.Error())
		return
	}

	privateLink, err := r.client.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
	if err != nil {
		plan.nullUnknownComputed()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.setComputed(ctx, privateLink)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActivePrivateLinkResource) finishCreate(ctx context.Context, subId, regionId int, principals []pl.PrivateLinkPrincipal, timeout time.Duration) error {
	if err := waitForActiveActivePrivateLinkToBeActive(ctx, r.client, subId, regionId, timeout); err != nil {
		return err
	}
	if err := createActiveActivePrincipals(ctx, r.client, subId, regionId, createRequestsFor(principals[1:])); err != nil {
		return err
	}
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		return err
	}
	// Task completion does not guarantee that the requested principals exist.
	return WaitForPrivateLinkPrincipals(ctx, activeActivePrivateLinkGetter(r.client, subId, regionId), principals, timeout)
}

func (r *activeActivePrivateLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ActiveActivePrivateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, privateLinkReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	subId, regionId, err := parseActiveActivePrivateLinkId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid PrivateLink ID", err.Error())
		return
	}

	privateLink, err := r.client.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
	if err != nil {
		var notFound *pl.NotFound
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
		return
	}

	state.ID = types.StringValue(makeActiveActivePrivateLinkId(subId, regionId))
	state.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	state.RegionID = types.Int64Value(int64(regionId))
	state.ShareName = types.StringPointerValue(privateLink.ShareName)
	state.Principal, diags = configuredPrincipalsFromAPI(ctx, privateLink.Principals)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(state.setComputed(ctx, privateLink)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *activeActivePrivateLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ActiveActivePrivateLinkResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, privateLinkUpdateTimeout)
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
	regionId := int(plan.RegionID.ValueInt64())

	if !plan.Principal.Equal(state.Principal) {
		utils.SubscriptionMutex.Lock(subId)
		defer utils.SubscriptionMutex.Unlock(subId)

		privateLink, err := r.client.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
		if err != nil {
			var notFound *pl.NotFound
			if errors.As(err, &notFound) {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
			return
		}

		tfPrincipals, diags := principalsFromPlan(ctx, plan.Principal)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if err := createActiveActivePrincipals(ctx, r.client, subId, regionId, findPrincipalsToCreate(privateLink.Principals, tfPrincipals)); err != nil {
			resp.Diagnostics.AddError("Failed to create PrivateLink principals", err.Error())
			return
		}
		if err := deleteActiveActivePrincipals(ctx, r.client, subId, regionId, findPrincipalsToDelete(privateLink.Principals, tfPrincipals)); err != nil {
			resp.Diagnostics.AddError("Failed to delete PrivateLink principals", err.Error())
			return
		}

		// This resource manages the complete principal set, so an incomplete
		// change must not be reported as a successful update.
		if err := WaitForPrivateLinkPrincipals(ctx, activeActivePrivateLinkGetter(r.client, subId, regionId), tfPrincipals, updateTimeout); err != nil {
			resp.Diagnostics.AddError("Failed to update PrivateLink principals", err.Error())
			return
		}
	}

	privateLink, err := r.client.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
	if err != nil {
		var notFound *pl.NotFound
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
		return
	}

	resp.Diagnostics.Append(plan.setComputed(ctx, privateLink)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActivePrivateLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ActiveActivePrivateLinkResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, privateLinkDeleteTimeout)
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
	regionId := int(state.RegionID.ValueInt64())

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := r.client.Client.PrivateLink.DeleteActiveActivePrivateLink(ctx, subId, regionId); err != nil {
		var notFound *pl.NotFound
		if errors.As(err, &notFound) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete PrivateLink", err.Error())
		return
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("Failed to delete PrivateLink", err.Error())
	}
}

func (m *ActiveActivePrivateLinkResourceModel) setComputed(ctx context.Context, privateLink *pl.PrivateLink) diag.Diagnostics {
	return setComputedFromAPI(ctx, privateLink, &m.ResourceConfigurationID, &m.ResourceConfigurationArn, &m.ShareArn, &m.Connections, &m.Databases)
}

func (m *ActiveActivePrivateLinkResourceModel) nullUnknownComputed() {
	nullUnknownComputed(&m.ResourceConfigurationID, &m.ResourceConfigurationArn, &m.ShareArn, &m.Connections, &m.Databases)
}

func parseActiveActivePrivateLinkId(id string) (int, int, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected format of ID (%q), expected <subscription_id>/<region_id>", id)
	}
	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	regionId, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return subId, regionId, nil
}

func createRequestsFor(principals []pl.PrivateLinkPrincipal) []pl.CreatePrivateLinkPrincipal {
	requests := make([]pl.CreatePrivateLinkPrincipal, 0, len(principals))
	for _, principal := range principals {
		requests = append(requests, pl.CreatePrivateLinkPrincipal{
			Principal:      principal.Principal,
			PrincipalType:  principal.Type,
			PrincipalAlias: principal.Alias,
		})
	}
	return requests
}

func createActiveActivePrincipals(ctx context.Context, api *client.ApiClient, subId int, regionId int, principals []pl.CreatePrivateLinkPrincipal) error {
	for _, principal := range principals {
		if err := api.Client.PrivateLink.CreateActiveActivePrincipal(ctx, subId, regionId, principal); err != nil {
			return err
		}
	}
	return nil
}

func deleteActiveActivePrincipals(ctx context.Context, api *client.ApiClient, subId int, regionId int, principals []pl.PrivateLinkPrincipal) error {
	for _, principal := range principals {
		if err := api.Client.PrivateLink.DeleteActiveActivePrincipal(ctx, subId, regionId, *principal.Principal); err != nil {
			return err
		}
	}
	return nil
}

func activeActivePrivateLinkGetter(api *client.ApiClient, subscriptionID, regionID int) PrivateLinkGetter {
	return func(ctx context.Context) (*pl.PrivateLink, error) {
		return api.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subscriptionID, regionID)
	}
}
