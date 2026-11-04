package privatelink

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	privateLinkCreateTimeout = 30 * time.Minute
	privateLinkReadTimeout   = 10 * time.Minute
	privateLinkUpdateTimeout = 30 * time.Minute
	privateLinkDeleteTimeout = 10 * time.Minute
)

var pathID = path.Root("id")

func (r *privateLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PrivateLinkResourceModel
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
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	principals, diags := principalsFromPlan(ctx, plan.Principal)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	firstPrincipal := principals[0]

	err = r.client.Client.PrivateLink.CreatePrivateLink(ctx, subId, pl.CreatePrivateLink{
		Principal:      firstPrincipal.Principal,
		PrincipalType:  firstPrincipal.Type,
		PrincipalAlias: firstPrincipal.Alias,
		ShareName:      plan.ShareName.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create PrivateLink", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(subId))

	if err := r.finishCreate(ctx, subId, principals, createTimeout); err != nil {
		// The PrivateLink exists at this point, so keep it in state like the SDKv2
		// implementation did by setting the ID before waiting.
		plan.nullUnknownComputed()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Failed to create PrivateLink", err.Error())
		return
	}

	privateLink, err := r.client.Client.PrivateLink.GetPrivateLink(ctx, subId)
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

func (r *privateLinkResource) finishCreate(ctx context.Context, subId int, principals []pl.PrivateLinkPrincipal, timeout time.Duration) error {
	if err := waitForPrivateLinkToBeActive(ctx, r.client, subId, timeout); err != nil {
		return err
	}
	if err := createOtherPrincipals(ctx, r.client, subId, principals[1:]); err != nil {
		return err
	}
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		return err
	}
	// Task completion does not guarantee that the requested principals exist.
	return WaitForPrivateLinkPrincipals(ctx, privateLinkGetter(r.client, subId), principals, timeout)
}

func (r *privateLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PrivateLinkResourceModel
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

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid PrivateLink ID", err.Error())
		return
	}

	privateLink, err := r.client.Client.PrivateLink.GetPrivateLink(ctx, subId)
	if err != nil {
		var notFound *pl.NotFound
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read PrivateLink", err.Error())
		return
	}

	state.ID = types.StringValue(strconv.Itoa(subId))
	state.SubscriptionID = types.StringValue(strconv.Itoa(subId))
	state.ShareName = types.StringPointerValue(privateLink.ShareName)
	state.Principal, diags = configuredPrincipalsFromAPI(ctx, privateLink.Principals)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(state.setComputed(ctx, privateLink)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PrivateLinkResourceModel
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

	if !plan.Principal.Equal(state.Principal) {
		utils.SubscriptionMutex.Lock(subId)
		defer utils.SubscriptionMutex.Unlock(subId)

		privateLink, err := r.client.Client.PrivateLink.GetPrivateLink(ctx, subId)
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

		if err := createPrincipals(ctx, r.client, subId, findPrincipalsToCreate(privateLink.Principals, tfPrincipals)); err != nil {
			resp.Diagnostics.AddError("Failed to create PrivateLink principals", err.Error())
			return
		}
		if err := deletePrincipals(ctx, r.client, subId, findPrincipalsToDelete(privateLink.Principals, tfPrincipals)); err != nil {
			resp.Diagnostics.AddError("Failed to delete PrivateLink principals", err.Error())
			return
		}

		// This resource manages the complete principal set, so an incomplete
		// change must not be reported as a successful update.
		if err := WaitForPrivateLinkPrincipals(ctx, privateLinkGetter(r.client, subId), tfPrincipals, updateTimeout); err != nil {
			resp.Diagnostics.AddError("Failed to update PrivateLink principals", err.Error())
			return
		}
	}

	privateLink, err := r.client.Client.PrivateLink.GetPrivateLink(ctx, subId)
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

func (r *privateLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PrivateLinkResourceModel
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
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := r.client.Client.PrivateLink.DeletePrivateLink(ctx, subId); err != nil {
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

func (m *PrivateLinkResourceModel) setComputed(ctx context.Context, privateLink *pl.PrivateLink) diag.Diagnostics {
	return setComputedFromAPI(ctx, privateLink, &m.ResourceConfigurationID, &m.ResourceConfigurationArn, &m.ShareArn, &m.Connections, &m.Databases)
}

func (m *PrivateLinkResourceModel) nullUnknownComputed() {
	nullUnknownComputed(&m.ResourceConfigurationID, &m.ResourceConfigurationArn, &m.ShareArn, &m.Connections, &m.Databases)
}

func setComputedFromAPI(ctx context.Context, privateLink *pl.PrivateLink, resourceConfigurationID, resourceConfigurationArn, shareArn *types.String, connections, databases *types.Set) diag.Diagnostics {
	var diags, d diag.Diagnostics
	*resourceConfigurationID = types.StringPointerValue(privateLink.ResourceConfigurationId)
	*resourceConfigurationArn = types.StringPointerValue(privateLink.ResourceConfigurationArn)
	*shareArn = types.StringPointerValue(privateLink.ShareArn)
	*connections, d = ConnectionsFromAPI(ctx, privateLink.Connections)
	diags.Append(d...)
	*databases, d = DatabasesFromAPI(ctx, privateLink.Databases)
	diags.Append(d...)
	return diags
}

func nullUnknownComputed(resourceConfigurationID, resourceConfigurationArn, shareArn *types.String, connections, databases *types.Set) {
	for _, v := range []*types.String{resourceConfigurationID, resourceConfigurationArn, shareArn} {
		if v.IsUnknown() {
			*v = types.StringNull()
		}
	}
	if connections.IsUnknown() {
		*connections = types.SetNull(connectionElemType)
	}
	if databases.IsUnknown() {
		*databases = types.SetNull(databaseElemType)
	}
}

// principalsFromPlan returns the configured principals sorted by principal, as
// the SDKv2 implementation did, so the first one is used to create the PrivateLink.
func principalsFromPlan(ctx context.Context, set PrincipalSetValue) ([]pl.PrivateLinkPrincipal, diag.Diagnostics) {
	models, diags := set.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}

	principals := make([]pl.PrivateLinkPrincipal, 0, len(models))
	for _, model := range models {
		principals = append(principals, pl.PrivateLinkPrincipal{
			Principal: redis.String(model.Principal.ValueString()),
			Type:      redis.String(model.PrincipalType.ValueString()),
			Alias:     redis.String(model.PrincipalAlias.ValueString()),
		})
	}

	sort.Slice(principals, func(i, j int) bool {
		return *principals[i].Principal < *principals[j].Principal
	})

	return principals, diags
}

// configuredPrincipalsFromAPI maps API principals onto the configurable block.
// SDKv2 stored an unset principal_alias as "", so an empty alias maps to null to
// keep state written by SDKv2 releases free of diffs against configurations
// that omit it.
func configuredPrincipalsFromAPI(ctx context.Context, principals []*pl.PrivateLinkPrincipal) (PrincipalSetValue, diag.Diagnostics) {
	models := PrincipalModelsFromAPI(principals)
	for i := range models {
		if models[i].PrincipalAlias.ValueString() == "" {
			models[i].PrincipalAlias = types.StringNull()
		}
	}
	return NewPrincipalSet(ctx, models)
}

func privateLinkGetter(api *client.ApiClient, subscriptionID int) PrivateLinkGetter {
	return func(ctx context.Context) (*pl.PrivateLink, error) {
		return api.Client.PrivateLink.GetPrivateLink(ctx, subscriptionID)
	}
}
