package activeactive

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func (r *activeActiveSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resp.State.Schema == nil {
		resp.State.Schema = req.Plan.Schema
	}

	timeout, diags := plan.Timeouts.Create(ctx, activeActiveSubscriptionCreateDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	r.create(ctx, &plan, &resp.Diagnostics, timeout)
	if plan.ID.IsNull() || plan.ID.ValueString() == "" {
		return
	}

	// The API accepted the subscription. Keep its ID even when a later wait fails,
	// so Terraform still owns the object.
	clearUnknownComputed(ctx, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, activeActiveSubscriptionReadDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	removed := r.refresh(ctx, &state, &resp.Diagnostics, timeout, true)
	if resp.Diagnostics.HasError() {
		return
	}
	if removed {
		resp.State.RemoveResource(ctx)
		return
	}
	clearUnknownComputed(ctx, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *activeActiveSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	if resp.State.Schema == nil {
		resp.State.Schema = req.Plan.Schema
	}

	timeout, diags := plan.Timeouts.Update(ctx, activeActiveSubscriptionUpdateDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subID, err := subscriptionID(plan.ID)
	if err != nil {
		addAPIError(&resp.Diagnostics, err)
		return
	}
	utils.SubscriptionMutex.Lock(subID)
	defer utils.SubscriptionMutex.Unlock(subID)

	removed := r.update(ctx, &plan, state, &resp.Diagnostics, timeout)
	if resp.Diagnostics.HasError() {
		return
	}
	if removed {
		resp.State.RemoveResource(ctx)
		return
	}
	clearUnknownComputed(ctx, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ActiveActiveSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, activeActiveSubscriptionDeleteDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subID, err := subscriptionID(state.ID)
	if err != nil {
		addAPIError(&resp.Diagnostics, err)
		return
	}
	utils.SubscriptionMutex.Lock(subID)
	defer utils.SubscriptionMutex.Unlock(subID)

	r.delete(ctx, subID, &resp.Diagnostics, timeout)
}

func (r *activeActiveSubscriptionResource) create(ctx context.Context, plan *ActiveActiveSubscriptionResourceModel, diags *diag.Diagnostics, timeout time.Duration) {
	if r.client == nil {
		diags.AddError("Provider Not Configured", "The provider client is not configured. This is an internal error - please report this to the provider developers.")
		return
	}

	request, requestDiags := newActiveActiveCreateSubscription(ctx, *plan)
	diags.Append(requestDiags...)
	if diags.HasError() {
		return
	}

	subID, err := r.client.Client.Subscription.Create(ctx, request)
	if err != nil {
		addAPIError(diags, err)
		return
	}
	plan.ID = types.StringValue(strconv.Itoa(subID))

	if boolOr(plan.CustomerManagedKeyEnabled, false) {
		if err := utils.WaitForSubscriptionToBeEncryptionKeyPending(ctx, subID, r.client, timeout); err != nil {
			addAPIError(diags, err)
			return
		}
		r.refresh(ctx, plan, diags, timeout, false)
		return
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
		addAPIError(diags, err)
		return
	}
	if cleanup := utils.DeleteSubscriptionDatabases(ctx, subID, r.client, timeout); cleanup.HasError() {
		appendSDKDiagnostics(diags, cleanup)
		return
	}
	if maintenanceConfigured(plan.MaintenanceWindows) {
		request, maintenanceDiags := maintenanceFromModel(ctx, plan.MaintenanceWindows)
		diags.Append(maintenanceDiags...)
		if diags.HasError() {
			return
		}
		if err := r.client.Client.Maintenance.Update(ctx, subID, request); err != nil {
			addAPIError(diags, err)
			return
		}
	}
	r.refresh(ctx, plan, diags, timeout, false)
}

func (r *activeActiveSubscriptionResource) update(ctx context.Context, plan *ActiveActiveSubscriptionResourceModel, state ActiveActiveSubscriptionResourceModel, diags *diag.Diagnostics, timeout time.Duration) bool {
	if r.client == nil {
		diags.AddError("Provider Not Configured", "The provider client is not configured. This is an internal error - please report this to the provider developers.")
		return false
	}
	subID, err := subscriptionID(plan.ID)
	if err != nil {
		addAPIError(diags, err)
		return false
	}

	subscription, err := r.client.Client.Subscription.Get(ctx, subID)
	if err != nil {
		addAPIError(diags, err)
		return false
	}

	cmkEnabled := boolOr(plan.CustomerManagedKeyEnabled, false)
	wasCmkPending := false
	if redis.StringValue(subscription.Status) == subscriptions.SubscriptionStatusEncryptionKeyPending && cmkEnabled {
		wasCmkPending = true
		if !r.activateCustomerManagedKeys(ctx, plan, subID, timeout, diags) {
			return false
		}
	} else if cmkEnabled && !plan.CustomerManagedKeyDeletionGracePeriod.Equal(state.CustomerManagedKeyDeletionGracePeriod) {
		request := subscriptions.UpdateSubscriptionCMKs{
			DeletionGracePeriod: redis.String(plan.CustomerManagedKeyDeletionGracePeriod.ValueString()),
		}
		if err := r.client.Client.Subscription.UpdateCMKs(ctx, subID, request); err != nil {
			addAPIError(diags, err)
			return false
		}
		if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
			addAPIError(diags, err)
			return false
		}
	}

	if subscriptionIdentityChanged(*plan, state) {
		request := subscriptions.UpdateSubscription{}
		if !plan.Name.Equal(state.Name) {
			name := plan.Name.ValueString()
			request.Name = &name
		}
		if !plan.PaymentMethodID.Equal(state.PaymentMethodID) {
			paymentMethodID, err := paymentMethodIDPointer(plan.PaymentMethodID)
			if err != nil {
				addAPIError(diags, err)
				return false
			}
			request.PaymentMethodID = paymentMethodID
		}
		if !plan.PublicEndpointAccess.Equal(state.PublicEndpointAccess) {
			request.PublicEndpointAccess = redis.Bool(boolOr(plan.PublicEndpointAccess, true))
		}
		if err := r.client.Client.Subscription.Update(ctx, subID, request); err != nil {
			addAPIError(diags, err)
			return false
		}
		if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
			addAPIError(diags, err)
			return false
		}
	}

	if !mapsEffectivelyEqual(plan.ResourceTags, state.ResourceTags) {
		tags, tagDiags := resourceTagsToAPI(ctx, plan.ResourceTags)
		diags.Append(tagDiags...)
		if diags.HasError() {
			return false
		}
		if err := r.client.Client.Subscription.UpdateResourceTags(ctx, subID, subscriptions.UpdateResourceTags{ResourceTags: tags}); err != nil {
			addAPIError(diags, err)
			return false
		}
		if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
			addAPIError(diags, err)
			return false
		}
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
		addAPIError(diags, err)
		return false
	}
	if !plan.PublicEndpointAccess.Equal(state.PublicEndpointAccess) {
		if err := utils.WaitForSubscriptionPublicEndpointAccess(ctx, subID, r.client, boolOr(plan.PublicEndpointAccess, true), timeout); err != nil {
			addAPIError(diags, err)
			return false
		}
	}
	if !plan.MaintenanceWindows.Equal(state.MaintenanceWindows) || wasCmkPending {
		request, maintenanceDiags := maintenanceFromModel(ctx, plan.MaintenanceWindows)
		diags.Append(maintenanceDiags...)
		if diags.HasError() {
			return false
		}
		if err := r.client.Client.Maintenance.Update(ctx, subID, request); err != nil {
			addAPIError(diags, err)
			return false
		}
	}

	return r.refresh(ctx, plan, diags, timeout, true)
}

func (r *activeActiveSubscriptionResource) activateCustomerManagedKeys(ctx context.Context, plan *ActiveActiveSubscriptionResourceModel, subID int, timeout time.Duration, diags *diag.Diagnostics) bool {
	keys, keyDiags := customerManagedKeysFromModel(ctx, plan.CustomerManagedKey)
	diags.Append(keyDiags...)
	if diags.HasError() {
		return false
	}
	request := subscriptions.UpdateSubscriptionCMKs{
		DeletionGracePeriod: redis.String(plan.CustomerManagedKeyDeletionGracePeriod.ValueString()),
		CustomerManagedKeys: &keys,
	}
	if err := r.client.Client.Subscription.UpdateCMKs(ctx, subID, request); err != nil {
		addAPIError(diags, err)
		return false
	}
	if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
		addAPIError(diags, err)
		return false
	}
	if cleanup := utils.DeleteSubscriptionDatabases(ctx, subID, r.client, timeout); cleanup.HasError() {
		appendSDKDiagnostics(diags, cleanup)
		return false
	}
	return true
}

func (r *activeActiveSubscriptionResource) delete(ctx context.Context, subID int, diags *diag.Diagnostics, timeout time.Duration) {
	if r.client == nil {
		diags.AddError("Provider Not Configured", "The provider client is not configured. This is an internal error - please report this to the provider developers.")
		return
	}

	targetStatuses := []string{
		subscriptions.SubscriptionStatusActive,
		subscriptions.SubscriptionStatusDeleting,
		subscriptions.SubscriptionStatusDeleted,
	}
	pendingStatuses := []string{subscriptions.SubscriptionStatusPending, subscriptions.SubscriptionStatusEncryptionKeyPending}
	status, err := utils.WaitForSubscriptionToBeStatus(ctx, subID, r.client, timeout, targetStatuses, pendingStatuses...)
	if err != nil {
		addAPIError(diags, err)
		return
	}

	if status == subscriptions.SubscriptionStatusActive {
		// Creation-plan database cleanup can still be in flight when the subscription
		// first reports active. Wait briefly, then check again. The second wait also
		// notices automatic deletion that started after CMK access was revoked.
		time.Sleep(r.client.WaitDelay(30 * time.Second)) //lintignore:R018
		status, err = utils.WaitForSubscriptionToBeStatus(ctx, subID, r.client, timeout, targetStatuses, pendingStatuses...)
		if err != nil {
			addAPIError(diags, err)
			return
		}
	}

	if status == subscriptions.SubscriptionStatusActive {
		if err := r.client.Client.Subscription.Delete(ctx, subID); err != nil {
			addAPIError(diags, err)
			return
		}
	}
	if status != subscriptions.SubscriptionStatusDeleted {
		if err := utils.WaitForSubscriptionToBeDeleted(ctx, subID, r.client, timeout); err != nil {
			addAPIError(diags, err)
		}
	}
}

// refresh reads the subscription into model. removeOnNotFound reports that Read
// should drop the resource. Create keeps the ID and surfaces NotFound as an error.
func (r *activeActiveSubscriptionResource) refresh(ctx context.Context, model *ActiveActiveSubscriptionResourceModel, diags *diag.Diagnostics, timeout time.Duration, removeOnNotFound bool) bool {
	if r.client == nil {
		diags.AddError("Provider Not Configured", "The provider client is not configured. This is an internal error - please report this to the provider developers.")
		return false
	}
	subID, err := subscriptionID(model.ID)
	if err != nil {
		addAPIError(diags, err)
		return false
	}

	subscription, err := r.client.Client.Subscription.Get(ctx, subID)
	if err != nil {
		if isSubscriptionNotFound(err) && removeOnNotFound {
			return true
		}
		addAPIError(diags, err)
		return false
	}

	if redis.StringValue(subscription.Status) != subscriptions.SubscriptionStatusEncryptionKeyPending {
		if err := utils.WaitForSubscriptionToBeActive(ctx, subID, r.client, timeout); err != nil {
			addAPIError(diags, err)
			return false
		}
		subscription, err = r.client.Client.Subscription.Get(ctx, subID)
		if err != nil {
			if isSubscriptionNotFound(err) && removeOnNotFound {
				return true
			}
			addAPIError(diags, err)
			return false
		}
	}

	diags.Append(writeSubscriptionState(ctx, model, subscription)...)
	if diags.HasError() {
		return false
	}

	if redis.StringValue(subscription.Status) != subscriptions.SubscriptionStatusActive {
		return false
	}

	windows, err := r.client.Client.Maintenance.Get(ctx, subID)
	if err != nil {
		addAPIError(diags, err)
		return false
	}
	prices, err := r.client.Client.Pricing.List(ctx, subID)
	if err != nil {
		addAPIError(diags, err)
		return false
	}
	diags.Append(writeActiveCollections(ctx, model, windows, prices)...)
	return false
}

func subscriptionIdentityChanged(plan, state ActiveActiveSubscriptionResourceModel) bool {
	return !plan.Name.Equal(state.Name) ||
		!plan.PaymentMethodID.Equal(state.PaymentMethodID) ||
		!plan.PublicEndpointAccess.Equal(state.PublicEndpointAccess)
}

func maintenanceConfigured(windows customtypes.MaintenanceListValue) bool {
	return !windows.IsNull() && !windows.IsUnknown() && len(windows.Elements()) > 0
}

func subscriptionID(id types.String) (int, error) {
	return strconv.Atoi(id.ValueString())
}

func isSubscriptionNotFound(err error) bool {
	var notFound *subscriptions.NotFound
	return errors.As(err, &notFound)
}

func addAPIError(diags *diag.Diagnostics, err error) {
	diags.AddError(err.Error(), err.Error())
}

func appendSDKDiagnostics(diags *diag.Diagnostics, cleanup sdkdiag.Diagnostics) {
	for _, diagnostic := range cleanup {
		diags.AddError(diagnostic.Summary, diagnostic.Detail)
	}
}

func clearUnknownComputed(ctx context.Context, model *ActiveActiveSubscriptionResourceModel) {
	if model.Name.IsUnknown() {
		model.Name = types.StringValue("")
	}
	if model.PaymentMethod.IsUnknown() {
		model.PaymentMethod = types.StringValue("")
	}
	if model.PaymentMethodID.IsUnknown() {
		model.PaymentMethodID = types.StringValue("")
	}
	if model.CloudProvider.IsUnknown() {
		model.CloudProvider = types.StringValue("")
	}
	if model.AwsAccountID.IsUnknown() {
		model.AwsAccountID = types.StringValue("")
	}
	if model.RedisVersion.IsUnknown() {
		model.RedisVersion = types.StringNull()
	}
	if model.CreationPlan.IsUnknown() {
		model.CreationPlan = nullCreationPlanListValue()
	}
	if model.MaintenanceWindows.IsUnknown() {
		model.MaintenanceWindows, _ = customtypesNewNullMaintenance(ctx)
	}
	if model.Pricing.IsUnknown() {
		model.Pricing = types.ListNull(utils.PricingElemType)
	}
	if model.CustomerManagedKey.IsUnknown() {
		model.CustomerManagedKey = nullCustomerManagedKeyListValue()
	}
	if model.CustomerManagedKeyRedisServiceAccount.IsUnknown() {
		model.CustomerManagedKeyRedisServiceAccount = types.StringValue("")
	}
	if model.CustomerManagedKeyAwsRoleArn.IsUnknown() {
		model.CustomerManagedKeyAwsRoleArn = types.StringValue("")
	}
	if model.CustomerManagedKeyDeletionGracePeriod.IsUnknown() {
		model.CustomerManagedKeyDeletionGracePeriod = types.StringValue("immediate")
	}
	if model.CustomerManagedKeyEnabled.IsUnknown() {
		model.CustomerManagedKeyEnabled = types.BoolValue(false)
	}
	if model.PublicEndpointAccess.IsUnknown() {
		model.PublicEndpointAccess = types.BoolValue(true)
	}
	if model.ResourceTags.IsUnknown() {
		tags, _ := utils.ResourceTagsFromAPI(ctx, nil)
		model.ResourceTags = tags
	}
}

func customtypesNewNullMaintenance(ctx context.Context) (customtypes.MaintenanceListValue, diag.Diagnostics) {
	return customtypes.NewMaintenanceList(ctx, nil)
}
