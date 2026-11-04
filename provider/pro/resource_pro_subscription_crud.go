package pro

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// proSubscriptionConfigKey stores, in private state, configuration facts that the
// Framework cannot recover from the API: whether maintenance_windows was configured
// and whether resource_tags was configured as an empty map. SDKv2 states and imports
// have no such record.
const proSubscriptionConfigKey = "subscription_config"

type proSubscriptionConfig struct {
	MaintenanceWindows bool `json:"maintenance_windows"`
	EmptyResourceTags  bool `json:"empty_resource_tags"`
}

type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
}

type privateStateSetter interface {
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func readProSubscriptionConfig(ctx context.Context, private privateState) (*proSubscriptionConfig, diag.Diagnostics) {
	raw, diags := private.GetKey(ctx, proSubscriptionConfigKey)
	if diags.HasError() || len(raw) == 0 {
		return nil, diags
	}
	var cfg proSubscriptionConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		diags.AddError("Unable to read private state", err.Error())
		return nil, diags
	}
	return &cfg, diags
}

func writeProSubscriptionConfig(ctx context.Context, private privateStateSetter, plan *ProSubscriptionResourceModel) diag.Diagnostics {
	tags, diags := cloudProviderResourceTags(ctx, plan.CloudProvider)
	if diags.HasError() {
		return diags
	}
	cfg := proSubscriptionConfig{
		MaintenanceWindows: !plan.MaintenanceWindows.IsNull() && len(plan.MaintenanceWindows.Elements()) > 0,
		EmptyResourceTags:  !tags.IsNull() && !tags.IsUnknown() && len(tags.Elements()) == 0,
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		diags.AddError("Unable to write private state", err.Error())
		return diags
	}
	diags.Append(private.SetKey(ctx, proSubscriptionConfigKey, raw)...)
	return diags
}

func (r *proSubscriptionResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var configCreationPlan types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("creation_plan"), &configCreationPlan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hasCreationPlan := configCreationPlan.IsUnknown() || len(configCreationPlan.Elements()) > 0

	if req.State.Raw.IsNull() {
		if !hasCreationPlan {
			resp.Diagnostics.AddError(`the "creation_plan" block is required`, `The "creation_plan" block is required to create a subscription.`)
			return
		}
		regions, diags := configuredRegions(ctx, req.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() || regions == nil {
			return
		}
		resp.Diagnostics.Append(setPlanRegions(ctx, &resp.Plan, regions)...)
		return
	}

	var plan, state ProSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !hasCreationPlan {
		// Replacing the subscription creates a new one, which needs a creation plan.
		if requiresReplacement(&plan, &state) {
			resp.Diagnostics.AddError(`the "creation_plan" block is required`, `The "creation_plan" block is required because this change replaces the subscription.`)
		}
		return
	}

	planRegions, diags := configuredRegions(ctx, req.Config)
	resp.Diagnostics.Append(diags...)
	_, stateRegions, diags := cloudRegionModels(ctx, types.ListNull(subscriptionCloudProviderElemType), state.CloudProvider)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || planRegions == nil || stateRegions == nil {
		return
	}

	regionsChanged := fillRegionComputedValues(planRegions, stateRegions)
	if !regionsChanged {
		// As with the SDKv2 set hash, the key alone identifies a region: Update never
		// changes a region, and Read stores no CIDR or VPC id for multi-AZ regions.
		planRegions = stateRegions
	}
	resp.Diagnostics.Append(setPlanRegions(ctx, &resp.Plan, planRegions)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if regionsChanged {
		subId, err := strconv.Atoi(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("invalid subscription ID", err.Error())
			return
		}
		subscription, err := r.client.Client.Subscription.Get(ctx, subId)
		if err != nil {
			resp.Diagnostics.AddError("failed to get subscription", err.Error())
			return
		}
		// Regions can still change while the subscription waits for its encryption key.
		if redis.StringValue(subscription.Status) != subscriptions.SubscriptionStatusEncryptionKeyPending {
			resp.RequiresReplace.Append(regionPath)
		}
		return
	}

	// Computed values are only marked unknown because some value differs from state.
	// When restoring them from state makes the plan identical to state, nothing changes.
	filled, err := fillUnknownValues(resp.Plan.Raw, req.State.Raw, false)
	if err != nil {
		resp.Diagnostics.AddError("Error modifying plan", err.Error())
		return
	}
	if filled.Equal(req.State.Raw) {
		resp.Plan.Raw = filled
	}
}

var regionPath = path.Root("cloud_provider").AtListIndex(0).AtName("region")

// configuredRegions returns the configured regions, with the default and the unknown
// computed values of a planned region. They are not left to the framework, which finds
// the configuration of a set element by its value: it misses it as soon as the planned
// element differs from the configured one, then applies defaults over configured values
// and leaves computed values null.
func configuredRegions(ctx context.Context, config tfsdk.Config) ([]SubscriptionRegionModel, diag.Diagnostics) {
	var cloudProvider types.List
	diags := config.GetAttribute(ctx, path.Root("cloud_provider"), &cloudProvider)
	if diags.HasError() {
		return nil, diags
	}
	regions, _, d := cloudRegionModels(ctx, cloudProvider, types.ListNull(subscriptionCloudProviderElemType))
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	for i, r := range regions {
		if r.MultipleAvailabilityZones.IsNull() {
			regions[i].MultipleAvailabilityZones = types.BoolValue(false)
		}
		if r.PreferredAvailabilityZones.IsNull() {
			regions[i].PreferredAvailabilityZones = types.ListUnknown(types.StringType)
		}
		if r.NetworkingVPCID.IsNull() {
			regions[i].NetworkingVPCID = types.StringUnknown()
		}
		regions[i].Networks = types.ListUnknown(subscriptionNetworkElemType)
	}
	return regions, diags
}

func setPlanRegions(ctx context.Context, plan *tfsdk.Plan, regions []SubscriptionRegionModel) diag.Diagnostics {
	regionSet, diags := types.SetValueFrom(ctx, subscriptionRegionElemType, regions)
	if diags.HasError() {
		return diags
	}
	diags.Append(plan.SetAttribute(ctx, regionPath, regionSet)...)
	return diags
}

func requiresReplacement(plan, state *ProSubscriptionResourceModel) bool {
	if !plan.MemoryStorage.Equal(state.MemoryStorage) {
		return true
	}
	redisVersion := plan.RedisVersion.ValueString()
	if redisVersion != "" && redisVersion != state.RedisVersion.ValueString() {
		return true
	}
	planCP := plan.CloudProvider.Elements()
	stateCP := state.CloudProvider.Elements()
	if len(planCP) == 0 || len(stateCP) == 0 {
		return false
	}
	planAttrs := planCP[0].(types.Object).Attributes()
	stateAttrs := stateCP[0].(types.Object).Attributes()
	return !planAttrs["provider"].Equal(stateAttrs["provider"]) || !planAttrs["cloud_account_id"].Equal(stateAttrs["cloud_account_id"])
}

func cloudRegionModels(ctx context.Context, planCP, stateCP types.List) ([]SubscriptionRegionModel, []SubscriptionRegionModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	decode := func(list types.List) []SubscriptionRegionModel {
		if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
			return nil
		}
		var providers []SubscriptionCloudProviderModel
		diags.Append(list.ElementsAs(ctx, &providers, false)...)
		if diags.HasError() || providers[0].Region.IsNull() || providers[0].Region.IsUnknown() {
			return nil
		}
		regions := make([]SubscriptionRegionModel, 0, len(providers[0].Region.Elements()))
		diags.Append(providers[0].Region.ElementsAs(ctx, &regions, false)...)
		return regions
	}
	planRegions := decode(planCP)
	stateRegions := decode(stateCP)
	return planRegions, stateRegions, diags
}

func regionKey(r SubscriptionRegionModel) (string, bool) {
	if r.Region.IsUnknown() || r.MultipleAvailabilityZones.IsUnknown() {
		return "", false
	}
	return fmt.Sprintf("%s-%t", r.Region.ValueString(), r.MultipleAvailabilityZones.ValueBool()), true
}

// fillRegionComputedValues copies the computed values of each region from the
// state region with the same key. Like the SDKv2 set hash, the key is the region
// name and its multiple_availability_zones flag. It reports whether the set of
// keys changed, which is what used to force a new subscription.
func fillRegionComputedValues(planRegions, stateRegions []SubscriptionRegionModel) bool {
	stateByKey := make(map[string]SubscriptionRegionModel, len(stateRegions))
	for _, r := range stateRegions {
		if key, ok := regionKey(r); ok {
			stateByKey[key] = r
		}
	}

	changed := len(planRegions) != len(stateRegions)
	seen := make(map[string]bool, len(planRegions))
	for i, r := range planRegions {
		key, ok := regionKey(r)
		prior, found := stateByKey[key]
		if !ok || !found || seen[key] {
			changed = true
			continue
		}
		seen[key] = true
		if r.Networks.IsUnknown() {
			planRegions[i].Networks = prior.Networks
		}
		if r.PreferredAvailabilityZones.IsUnknown() {
			planRegions[i].PreferredAvailabilityZones = prior.PreferredAvailabilityZones
		}
		if r.NetworkingVPCID.IsUnknown() {
			planRegions[i].NetworkingVPCID = prior.NetworkingVPCID
		}
	}
	return changed
}

// fillUnknownValues replaces each unknown value in target with the value at the same
// path in source. Unknown values without a known counterpart become null when
// nullFallback is set, and stay unknown otherwise.
func fillUnknownValues(target, source tftypes.Value, nullFallback bool) (tftypes.Value, error) {
	return tftypes.Transform(target, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if v.IsKnown() {
			return v, nil
		}
		if found, _, err := tftypes.WalkAttributePath(source, p); err == nil {
			if fv, ok := found.(tftypes.Value); ok && fv.IsFullyKnown() {
				return fv, nil
			}
		}
		if nullFallback {
			return tftypes.NewValue(v.Type(), nil), nil
		}
		return v, nil
	})
}

func (r *proSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, proSubscriptionCreateDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	providers, diags := buildCreateCloudProviders(ctx, plan.CloudProvider)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	paymentMethodID, err := paymentMethodIDFromModel(plan.PaymentMethodID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid payment_method_id", err.Error())
		return
	}

	var creationPlans []CreationPlanModel
	resp.Diagnostics.Append(plan.CreationPlan.ElementsAs(ctx, &creationPlans, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(creationPlans) == 0 {
		resp.Diagnostics.AddError(`the "creation_plan" block is required`, `The "creation_plan" block is required to create a subscription.`)
		return
	}

	memoryStorage := plan.MemoryStorage.ValueString()
	dbs, diags := BuildSubscriptionCreatePlanDatabases(ctx, memoryStorage, creationPlans[0])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := subscriptions.CreateSubscription{
		Name:            redis.String(plan.Name.ValueString()),
		DryRun:          redis.Bool(false),
		PaymentMethodID: paymentMethodID,
		PaymentMethod:   redis.String(plan.PaymentMethod.ValueString()),
		MemoryStorage:   redis.String(memoryStorage),
		CloudProviders:  providers,
		Databases:       dbs,
	}

	if redisVersion := plan.RedisVersion.ValueString(); redisVersion != "" {
		createRequest.RedisVersion = redis.String(redisVersion)
	}

	cmkEnabled := plan.CustomerManagedKeyEnabled.ValueBool()
	if cmkEnabled {
		createRequest.PersistentStorageEncryptionType = redis.String(utils.CmkEnabledString)
	}

	createRequest.PublicEndpointAccess = redis.Bool(plan.PublicEndpointAccess.ValueBool())

	subId, err := r.client.Client.Subscription.Create(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating subscription", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(subId))
	resp.Diagnostics.Append(resolveCreationPlanComputedValues(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(writeProSubscriptionConfig(ctx, resp.Private, &plan)...)

	// Keep ownership of the new subscription if a later step fails.
	saveOnError := func() {
		resp.Diagnostics.Append(setStateWithoutUnknowns(ctx, &resp.State, &plan, nil)...)
	}

	if cmkEnabled {
		if err := utils.WaitForSubscriptionToBeEncryptionKeyPending(ctx, subId, r.client, timeout); err != nil {
			resp.Diagnostics.AddError("Error waiting for subscription to be pending the encryption key", err.Error())
			saveOnError()
			return
		}
		resp.Diagnostics.Append(r.readAfterApply(ctx, subId, &plan, &resp.State)...)
		return
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		resp.Diagnostics.AddError("Error waiting for subscription to be active", err.Error())
		saveOnError()
		return
	}

	// Delete the databases created by the subscription creation plan.
	if cleanupDiags := utils.DeleteSubscriptionDatabases(ctx, subId, r.client, timeout); cleanupDiags != nil {
		resp.Diagnostics.Append(frameworkDiags(cleanupDiags)...)
		if resp.Diagnostics.HasError() {
			saveOnError()
			return
		}
	}

	// Some attributes are not accessible by the subscription creation API, so apply
	// them as an update of the new subscription.
	resp.Diagnostics.Append(r.applyUpdates(ctx, subId, &plan, nil, timeout)...)
	if resp.Diagnostics.HasError() {
		saveOnError()
		return
	}

	resp.Diagnostics.Append(r.readAfterApply(ctx, subId, &plan, &resp.State)...)
}

func (r *proSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, proSubscriptionReadDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription ID", err.Error())
		return
	}

	cfg, diags := readProSubscriptionConfig(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, diags := r.refreshFromAPI(ctx, subId, &state, cfg, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(setStateWithoutUnknowns(ctx, &resp.State, &state, nil)...)
}

func (r *proSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ProSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	timeout, diags := plan.Timeouts.Update(ctx, proSubscriptionUpdateDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription ID", err.Error())
		return
	}

	resp.Diagnostics.Append(r.applyUpdates(ctx, subId, &plan, &state, timeout)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(writeProSubscriptionConfig(ctx, resp.Private, &plan)...)
	resp.Diagnostics.Append(r.readAfterApply(ctx, subId, &plan, &resp.State)...)
}

func (r *proSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, proSubscriptionDeleteDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription ID", err.Error())
		return
	}

	resp.Diagnostics.Append(r.deleteSubscription(ctx, subId, timeout)...)
}

func (r *proSubscriptionResource) deleteSubscription(ctx context.Context, subId int, timeout time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// Terraform deletes database resources before their subscription, but SM/CAPI
	// database cleanup is asynchronous and subscription deletion is rejected while
	// databases remain. Wait for active before issuing DELETE.
	// Revoking CMK access during teardown can start automatic subscription deletion;
	// accept deleting/deleted too so we wait for or acknowledge that deletion.
	targetStatuses := []string{subscriptions.SubscriptionStatusActive, subscriptions.SubscriptionStatusDeleting, subscriptions.SubscriptionStatusDeleted}
	pendingStatuses := []string{subscriptions.SubscriptionStatusPending, subscriptions.SubscriptionStatusEncryptionKeyPending}
	status, err := utils.WaitForSubscriptionToBeStatus(ctx, subId, r.client, timeout, targetStatuses, pendingStatuses...)
	if err != nil {
		diags.AddError("Error waiting for subscription before deletion", err.Error())
		return diags
	}

	if status == subscriptions.SubscriptionStatusActive {
		// CAPI can report active before creation-plan database cleanup has finished.
		// Give that cleanup 30 seconds, then recheck readiness; deleting immediately
		// can fail because the subscription is still in use. The second wait also
		// catches automatic deletion that started after the first wait.
		time.Sleep(r.client.WaitDelay(30 * time.Second)) //lintignore:R018
		status, err = utils.WaitForSubscriptionToBeStatus(ctx, subId, r.client, timeout, targetStatuses, pendingStatuses...)
		if err != nil {
			diags.AddError("Error waiting for subscription before deletion", err.Error())
			return diags
		}
	}

	if status == subscriptions.SubscriptionStatusActive {
		if err := r.client.Client.Subscription.Delete(ctx, subId); err != nil {
			diags.AddError("Error deleting subscription", err.Error())
			return diags
		}
	}

	if status != subscriptions.SubscriptionStatusDeleted {
		if err := utils.WaitForSubscriptionToBeDeleted(ctx, subId, r.client, timeout); err != nil {
			diags.AddError("Error waiting for subscription to be deleted", err.Error())
		}
	}
	return diags
}

// applyUpdates applies the changes between plan and prior. A nil prior means the
// subscription was just created: every configured value then counts as a change, as
// the SDKv2 resource did by running its update after creation.
func (r *proSubscriptionResource) applyUpdates(ctx context.Context, subId int, plan, prior *ProSubscriptionResourceModel, timeout time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics
	if prior == nil {
		prior = &ProSubscriptionResourceModel{
			Allowlist:          types.ListNull(subscriptionAllowlistElemType),
			CloudProvider:      types.ListNull(subscriptionCloudProviderElemType),
			MaintenanceWindows: customtypes.MaintenanceListValue{ListValue: types.ListNull(customtypes.NewMaintenanceListType().ElemType)},
		}
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	subscription, err := r.client.Client.Subscription.Get(ctx, subId)
	if err != nil {
		diags.AddError("Error reading subscription", err.Error())
		return diags
	}

	cmkEnabled := plan.CustomerManagedKeyEnabled.ValueBool()
	wasCmkPending := false
	if redis.StringValue(subscription.Status) == subscriptions.SubscriptionStatusEncryptionKeyPending && cmkEnabled {
		wasCmkPending = true
		diags.Append(r.updateCustomerManagedKeys(ctx, subId, plan, timeout)...)
		if diags.HasError() {
			return diags
		}
	} else if cmkEnabled && plan.CustomerManagedKeyDeletionGracePeriod.ValueString() != prior.CustomerManagedKeyDeletionGracePeriod.ValueString() {
		// Updating the grace period must not resend customer keys after activation.
		request := subscriptions.UpdateSubscriptionCMKs{
			DeletionGracePeriod: redis.String(plan.CustomerManagedKeyDeletionGracePeriod.ValueString()),
		}
		if err := r.client.Client.Subscription.UpdateCMKs(ctx, subId, request); err != nil {
			diags.AddError("Error updating the customer managed key grace period", err.Error())
			return diags
		}
		if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
			diags.AddError("Error waiting for subscription to be active", err.Error())
			return diags
		}
	}

	planCidrs, planSGs, d := allowlistValues(ctx, plan.Allowlist)
	diags.Append(d...)
	priorCidrs, priorSGs, d := allowlistValues(ctx, prior.Allowlist)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	if len(plan.Allowlist.Elements()) != len(prior.Allowlist.Elements()) || !equalStrings(planCidrs, priorCidrs) || !equalStrings(planSGs, priorSGs) {
		err := r.client.Client.Subscription.UpdateCIDRAllowlist(ctx, subId, subscriptions.UpdateCIDRAllowlist{
			CIDRIPs:          redis.StringSlice(planCidrs...),
			SecurityGroupIDs: redis.StringSlice(planSGs...),
		})
		if err != nil {
			diags.AddError("Error updating the subscription allowlist", err.Error())
			return diags
		}
	}

	nameChanged := plan.Name.ValueString() != prior.Name.ValueString()
	paymentMethodIDChanged := !plan.PaymentMethodID.IsUnknown() && plan.PaymentMethodID.ValueString() != prior.PaymentMethodID.ValueString()
	publicEndpointAccessChanged := plan.PublicEndpointAccess.ValueBool() != prior.PublicEndpointAccess.ValueBool()

	if nameChanged || paymentMethodIDChanged || publicEndpointAccessChanged {
		updateRequest := subscriptions.UpdateSubscription{}
		if nameChanged {
			updateRequest.Name = redis.String(plan.Name.ValueString())
		}
		if paymentMethodIDChanged {
			paymentMethodID, err := paymentMethodIDFromModel(plan.PaymentMethodID)
			if err != nil {
				diags.AddError("Invalid payment_method_id", err.Error())
				return diags
			}
			updateRequest.PaymentMethodID = paymentMethodID
		}
		if publicEndpointAccessChanged {
			updateRequest.PublicEndpointAccess = redis.Bool(plan.PublicEndpointAccess.ValueBool())
		}

		if err := r.client.Client.Subscription.Update(ctx, subId, updateRequest); err != nil {
			diags.AddError("Error updating subscription", err.Error())
			return diags
		}
		if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
			diags.AddError("Error waiting for subscription to be active", err.Error())
			return diags
		}
	}

	planTags, d := cloudProviderResourceTags(ctx, plan.CloudProvider)
	diags.Append(d...)
	priorTags, d := cloudProviderResourceTags(ctx, prior.CloudProvider)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	if !equalStringMaps(planTags, priorTags) {
		resourceTags := make([]*subscriptions.ResourceTag, 0, len(planTags.Elements()))
		for k, v := range planTags.Elements() {
			resourceTags = append(resourceTags, &subscriptions.ResourceTag{
				Key:   redis.String(k),
				Value: redis.String(v.(types.String).ValueString()),
			})
		}
		err := r.client.Client.Subscription.UpdateResourceTags(ctx, subId, subscriptions.UpdateResourceTags{
			ResourceTags: resourceTags,
		})
		if err != nil {
			diags.AddError("Error updating subscription resource tags", err.Error())
			return diags
		}
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		diags.AddError("Error waiting for subscription to be active", err.Error())
		return diags
	}

	if publicEndpointAccessChanged {
		if err := utils.WaitForSubscriptionPublicEndpointAccess(ctx, subId, r.client, plan.PublicEndpointAccess.ValueBool(), timeout); err != nil {
			diags.AddError("Error waiting for subscription public endpoint access", err.Error())
			return diags
		}
	}

	if !equalMaintenanceWindows(plan.MaintenanceWindows, prior.MaintenanceWindows) || wasCmkPending {
		request, d := maintenanceRequestFromModel(ctx, plan.MaintenanceWindows)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		if err := r.client.Client.Maintenance.Update(ctx, subId, request); err != nil {
			diags.AddError("Error updating subscription maintenance windows", err.Error())
			return diags
		}
	}

	return diags
}

func (r *proSubscriptionResource) updateCustomerManagedKeys(ctx context.Context, subId int, plan *ProSubscriptionResourceModel, timeout time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.CustomerManagedKey.IsNull() || plan.CustomerManagedKey.IsUnknown() || len(plan.CustomerManagedKey.Elements()) == 0 {
		diags.AddError("customer_managed_key must be set when subscription is in encryption key pending state", "")
		return diags
	}

	var keys []CustomerManagedKeyModel
	diags.Append(plan.CustomerManagedKey.ElementsAs(ctx, &keys, false)...)
	if diags.HasError() {
		return diags
	}

	customerManagedKeys := make([]subscriptions.CustomerManagedKey, 0, len(keys))
	for _, key := range keys {
		customerManagedKeys = append(customerManagedKeys, subscriptions.CustomerManagedKey{
			ResourceName: redis.String(key.ResourceName.ValueString()),
		})
	}

	request := subscriptions.UpdateSubscriptionCMKs{
		DeletionGracePeriod: redis.String(plan.CustomerManagedKeyDeletionGracePeriod.ValueString()),
		CustomerManagedKeys: &customerManagedKeys,
	}
	if err := r.client.Client.Subscription.UpdateCMKs(ctx, subId, request); err != nil {
		diags.AddError("Error updating the subscription customer managed keys", err.Error())
		return diags
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		diags.AddError("Error waiting for subscription to be active", err.Error())
		return diags
	}

	// After CMK activation, delete databases that were not cleaned up during Create.
	if cleanupDiags := utils.DeleteSubscriptionDatabases(ctx, subId, r.client, timeout); cleanupDiags != nil {
		diags.Append(frameworkDiags(cleanupDiags)...)
	}
	return diags
}

// readAfterApply writes plan to state, replacing its unknown values with the API values.
// Known plan values are kept because Terraform requires the applied state to match them.
func (r *proSubscriptionResource) readAfterApply(ctx context.Context, subId int, plan *ProSubscriptionResourceModel, state *tfsdk.State) diag.Diagnostics {
	fresh := *plan
	found, diags := r.refreshFromAPI(ctx, subId, &fresh, nil, false)
	if diags.HasError() {
		diags.Append(setStateWithoutUnknowns(ctx, state, plan, nil)...)
		return diags
	}
	if !found {
		diags.AddError("Subscription not found", fmt.Sprintf("Subscription %d was not found after it was applied.", subId))
		diags.Append(setStateWithoutUnknowns(ctx, state, plan, nil)...)
		return diags
	}

	planRegions, freshRegions, d := cloudRegionModels(ctx, plan.CloudProvider, fresh.CloudProvider)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	if planRegions != nil && freshRegions != nil {
		fillRegionComputedValues(planRegions, freshRegions)
		regionSet, d := types.SetValueFrom(ctx, subscriptionRegionElemType, planRegions)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		var providers []SubscriptionCloudProviderModel
		diags.Append(plan.CloudProvider.ElementsAs(ctx, &providers, false)...)
		if diags.HasError() {
			return diags
		}
		providers[0].Region = regionSet
		plan.CloudProvider, d = types.ListValueFrom(ctx, subscriptionCloudProviderElemType, providers)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
	}

	diags.Append(setStateWithoutUnknowns(ctx, state, plan, &fresh)...)
	return diags
}

// setStateWithoutUnknowns writes model to state and replaces its remaining unknown
// values with the values of source, or with null.
func setStateWithoutUnknowns(ctx context.Context, state *tfsdk.State, model, source *ProSubscriptionResourceModel) diag.Diagnostics {
	diags := state.Set(ctx, model)
	if diags.HasError() {
		return diags
	}

	sourceRaw := tftypes.NewValue(state.Raw.Type(), nil)
	if source != nil {
		sourceState := tfsdk.State{Schema: state.Schema}
		diags.Append(sourceState.Set(ctx, source)...)
		if diags.HasError() {
			return diags
		}
		sourceRaw = sourceState.Raw
	}

	raw, err := fillUnknownValues(state.Raw, sourceRaw, true)
	if err != nil {
		diags.AddError("Error setting state", err.Error())
		return diags
	}
	state.Raw = raw
	return diags
}

// refreshFromAPI overwrites model with the subscription returned by the API. Values
// the API does not return are kept. cfg is only used on refresh, where it decides how
// to represent values that the configuration may or may not set.
func (r *proSubscriptionResource) refreshFromAPI(ctx context.Context, subId int, model *ProSubscriptionResourceModel, cfg *proSubscriptionConfig, refresh bool) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	subscription, err := r.client.Client.Subscription.Get(ctx, subId)
	if err != nil {
		notFound := &subscriptions.NotFound{}
		if errors.As(err, &notFound) {
			return false, diags
		}
		diags.AddError("Error reading subscription", err.Error())
		return true, diags
	}

	name := redis.StringValue(subscription.Name)
	if name != "" || !model.Name.IsNull() {
		model.Name = types.StringValue(name)
	}

	if subscription.PaymentMethodID != nil && redis.IntValue(subscription.PaymentMethodID) != 0 {
		model.PaymentMethodID = types.StringValue(strconv.Itoa(redis.IntValue(subscription.PaymentMethodID)))
	}
	model.PaymentMethod = types.StringValue(redis.StringValue(subscription.PaymentMethod))
	model.MemoryStorage = types.StringValue(redis.StringValue(subscription.MemoryStorage))

	emptyResourceTags := cfg != nil && cfg.EmptyResourceTags
	cloudProviders, d := subscriptionCloudProvidersFromAPI(ctx, subscription.CloudDetails, emptyResourceTags)
	diags.Append(d...)
	if diags.HasError() {
		return true, diags
	}
	model.CloudProvider = cloudProviders

	// CIDR allowlist is not allowed for Redis Labs internal resources subscription.
	if len(subscription.CloudDetails) > 0 && redis.IntValue(subscription.CloudDetails[0].CloudAccountID) != 1 {
		allowlist, d := r.subscriptionAllowlistFromAPI(ctx, subId, model.Allowlist)
		diags.Append(d...)
		if diags.HasError() {
			return true, diags
		}
		model.Allowlist = allowlist
	}

	if redis.StringValue(subscription.Status) == subscriptions.SubscriptionStatusActive {
		m, err := r.client.Client.Maintenance.Get(ctx, subId)
		if err != nil {
			diags.AddError("Error reading subscription maintenance windows", err.Error())
			return true, diags
		}
		if refresh {
			model.MaintenanceWindows, d = maintenanceWindowsFromAPI(ctx, m, cfg)
			diags.Append(d...)
		}

		pricingList, err := r.client.Client.Pricing.List(ctx, subId)
		if err != nil {
			diags.AddError("Error reading subscription pricing", err.Error())
			return true, diags
		}
		model.Pricing, d = utils.PricingFromAPI(ctx, pricingList)
		diags.Append(d...)
		if diags.HasError() {
			return true, diags
		}
	}

	if details := subscription.CustomerManagedKeyAccessDetails; details != nil {
		if details.RedisServiceAccount != nil {
			model.CustomerManagedKeyRedisServiceAccount = types.StringValue(redis.StringValue(details.RedisServiceAccount))
		}
		if details.AwsRoleArn != nil {
			model.CustomerManagedKeyAwsRoleArn = types.StringValue(redis.StringValue(details.AwsRoleArn))
		}
	}

	// Default to true if not set by API, matching Redis Cloud's default behaviour
	publicEndpointAccess := true
	if subscription.PublicEndpointAccess != nil {
		publicEndpointAccess = redis.BoolValue(subscription.PublicEndpointAccess)
	}
	model.PublicEndpointAccess = types.BoolValue(publicEndpointAccess)

	model.PrometheusEndpoint = types.StringValue(redis.StringValue(subscription.PrometheusEndpoint))

	model.CustomerManagedKeyEnabled = types.BoolValue(redis.StringValue(subscription.PersistentStorageEncryptionType) == utils.CmkEnabledString)

	if subscription.DeletionGracePeriod != nil {
		model.CustomerManagedKeyDeletionGracePeriod = types.StringValue(redis.StringValue(subscription.DeletionGracePeriod))
	}

	// Imported states have no blocks yet; Terraform represents absent blocks as empty lists.
	if model.Allowlist.IsNull() {
		model.Allowlist = types.ListValueMust(subscriptionAllowlistElemType, []attr.Value{})
	}
	if model.CreationPlan.IsNull() {
		model.CreationPlan = types.ListValueMust(creationPlanElemType, []attr.Value{})
	}
	if model.CustomerManagedKey.IsNull() {
		model.CustomerManagedKey = types.ListValueMust(customerManagedKeyElemType, []attr.Value{})
	}
	if model.MaintenanceWindows.IsNull() {
		model.MaintenanceWindows = emptyMaintenanceList()
	}

	return true, diags
}

func emptyMaintenanceList() customtypes.MaintenanceListValue {
	return customtypes.MaintenanceListValue{ListValue: types.ListValueMust(customtypes.NewMaintenanceListType().ElemType, []attr.Value{})}
}

// maintenanceWindowsFromAPI maps the maintenance windows read on refresh. A Framework
// block cannot be computed, so windows that were not configured stay empty. Without a
// record of the configuration (SDKv2 state or import), they also stay empty: an omitted
// block then plans no change and leaves windows set outside Terraform untouched, and a
// configured block is applied once.
func maintenanceWindowsFromAPI(ctx context.Context, m *maintenance.Maintenance, cfg *proSubscriptionConfig) (customtypes.MaintenanceListValue, diag.Diagnostics) {
	if cfg == nil || !cfg.MaintenanceWindows {
		return emptyMaintenanceList(), nil
	}
	return customtypes.NewMaintenanceList(ctx, m)
}

func maintenanceRequestFromModel(ctx context.Context, windows customtypes.MaintenanceListValue) (maintenance.Maintenance, diag.Diagnostics) {
	if windows.IsNull() || windows.IsUnknown() || len(windows.Elements()) == 0 {
		return maintenance.Maintenance{Mode: redis.String("automatic")}, nil
	}

	models, diags := windows.AsModels(ctx)
	if diags.HasError() {
		return maintenance.Maintenance{}, diags
	}

	request := maintenance.Maintenance{
		Mode:    redis.String(models[0].Mode.ValueString()),
		Windows: make([]*maintenance.Window, 0),
	}
	if models[0].Window.IsNull() || models[0].Window.IsUnknown() {
		return request, diags
	}

	windowModels, d := models[0].Window.AsModels(ctx)
	diags.Append(d...)
	if diags.HasError() {
		return maintenance.Maintenance{}, diags
	}
	for _, w := range windowModels {
		var days []string
		diags.Append(w.Days.ElementsAs(ctx, &days, false)...)
		if diags.HasError() {
			return maintenance.Maintenance{}, diags
		}
		request.Windows = append(request.Windows, &maintenance.Window{
			StartHour:       redis.Int(int(w.StartHour.ValueInt64())),
			DurationInHours: redis.Int(int(w.DurationInHours.ValueInt64())),
			Days:            redis.StringSlice(days...),
		})
	}
	return request, diags
}

func equalMaintenanceWindows(a, b customtypes.MaintenanceListValue) bool {
	if len(a.Elements()) == 0 && len(b.Elements()) == 0 {
		return true
	}
	return a.ListValue.Equal(b.ListValue)
}

func (r *proSubscriptionResource) subscriptionAllowlistFromAPI(ctx context.Context, subId int, prior types.List) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	allowlist, err := r.client.Client.Subscription.GetCIDRAllowlist(ctx, subId)
	if err != nil {
		diags.AddError("Error reading subscription allowlist", err.Error())
		return prior, diags
	}
	if !isNil(allowlist.Errors) {
		diags.AddError("Error reading subscription allowlist", fmt.Sprintf("unable to read allowlist for subscription %d: %v", subId, allowlist.Errors))
		return prior, diags
	}

	if len(allowlist.CIDRIPs) == 0 && len(allowlist.SecurityGroupIDs) == 0 {
		return types.ListValueMust(subscriptionAllowlistElemType, []attr.Value{}), diags
	}

	priorCidrs := types.SetNull(types.StringType)
	if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) > 0 {
		var priorModels []SubscriptionAllowlistModel
		diags.Append(prior.ElementsAs(ctx, &priorModels, false)...)
		if diags.HasError() {
			return prior, diags
		}
		priorCidrs = priorModels[0].Cidrs
	}

	cidrs := types.SetNull(types.StringType)
	if len(allowlist.CIDRIPs) > 0 {
		cidrs, diags = types.SetValueFrom(ctx, types.StringType, redis.StringSliceValue(allowlist.CIDRIPs...))
	} else if !priorCidrs.IsNull() && !priorCidrs.IsUnknown() && len(priorCidrs.Elements()) == 0 {
		cidrs = priorCidrs
	}
	if diags.HasError() {
		return prior, diags
	}

	// security_group_ids is required, so an empty API value can only mean an empty configuration.
	sgs, d := types.SetValueFrom(ctx, types.StringType, nonNilStrings(redis.StringSliceValue(allowlist.SecurityGroupIDs...)))
	diags.Append(d...)
	if diags.HasError() {
		return prior, diags
	}

	return types.ListValueFrom(ctx, subscriptionAllowlistElemType, []SubscriptionAllowlistModel{{Cidrs: cidrs, SecurityGroupIDs: sgs}})
}

func allowlistValues(ctx context.Context, allowlist types.List) ([]string, []string, diag.Diagnostics) {
	cidrs, sgs := []string{}, []string{}
	if allowlist.IsNull() || allowlist.IsUnknown() || len(allowlist.Elements()) == 0 {
		return cidrs, sgs, nil
	}
	var models []SubscriptionAllowlistModel
	diags := allowlist.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, nil, diags
	}
	if !models[0].Cidrs.IsNull() && !models[0].Cidrs.IsUnknown() {
		diags.Append(models[0].Cidrs.ElementsAs(ctx, &cidrs, false)...)
	}
	if !models[0].SecurityGroupIDs.IsNull() && !models[0].SecurityGroupIDs.IsUnknown() {
		diags.Append(models[0].SecurityGroupIDs.ElementsAs(ctx, &sgs, false)...)
	}
	sort.Strings(cidrs)
	sort.Strings(sgs)
	return cidrs, sgs, diags
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalStringMaps(a, b types.Map) bool {
	if len(a.Elements()) == 0 && len(b.Elements()) == 0 {
		return true
	}
	return a.Equal(b)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func cloudProviderResourceTags(ctx context.Context, cloudProvider types.List) (types.Map, diag.Diagnostics) {
	if cloudProvider.IsNull() || cloudProvider.IsUnknown() || len(cloudProvider.Elements()) == 0 {
		return types.MapNull(types.StringType), nil
	}
	var providers []SubscriptionCloudProviderModel
	diags := cloudProvider.ElementsAs(ctx, &providers, false)
	if diags.HasError() {
		return types.MapNull(types.StringType), diags
	}
	return providers[0].ResourceTags, diags
}

// subscriptionCloudProvidersFromAPI maps cloud details to the resource shape, which
// also carries the networking values of the first network of single-zone regions.
func subscriptionCloudProvidersFromAPI(ctx context.Context, cloudDetails []*subscriptions.CloudDetail, emptyResourceTags bool) (types.List, diag.Diagnostics) {
	models := make([]SubscriptionCloudProviderModel, 0, len(cloudDetails))
	for _, cloudDetail := range cloudDetails {
		regions, diags := subscriptionRegionsFromAPI(ctx, cloudDetail.Regions)
		if diags.HasError() {
			return types.ListNull(subscriptionCloudProviderElemType), diags
		}

		resourceTags := types.MapNull(types.StringType)
		if len(cloudDetail.ResourceTags) > 0 || emptyResourceTags {
			resourceTags, diags = utils.ResourceTagsFromAPI(ctx, cloudDetail.ResourceTags)
			if diags.HasError() {
				return types.ListNull(subscriptionCloudProviderElemType), diags
			}
		}

		models = append(models, SubscriptionCloudProviderModel{
			Provider:       types.StringValue(redis.StringValue(cloudDetail.Provider)),
			CloudAccountID: types.StringValue(strconv.Itoa(redis.IntValue(cloudDetail.CloudAccountID))),
			AWSAccountID:   types.StringPointerValue(cloudDetail.AWSAccountID),
			Region:         regions,
			ResourceTags:   resourceTags,
		})
	}
	return types.ListValueFrom(ctx, subscriptionCloudProviderElemType, models)
}

func subscriptionRegionsFromAPI(ctx context.Context, regions []*subscriptions.Region) (types.Set, diag.Diagnostics) {
	models := make([]SubscriptionRegionModel, 0, len(regions))
	for _, region := range regions {
		networks, diags := subscriptionNetworksFromAPI(ctx, region.Networking)
		if diags.HasError() {
			return types.SetNull(subscriptionRegionElemType), diags
		}

		preferredAZs, diags := types.ListValueFrom(ctx, types.StringType, nonNilStrings(redis.StringSliceValue(region.PreferredAvailabilityZones...)))
		if diags.HasError() {
			return types.SetNull(subscriptionRegionElemType), diags
		}

		multipleAZs := redis.BoolValue(region.MultipleAvailabilityZones)
		deploymentCIDR, vpcID := "", ""
		if len(region.Networking) > 0 && !multipleAZs {
			deploymentCIDR = redis.StringValue(region.Networking[0].DeploymentCIDR)
			vpcID = redis.StringValue(region.Networking[0].VPCId)
		}

		models = append(models, SubscriptionRegionModel{
			Region:                     types.StringValue(redis.StringValue(region.Region)),
			MultipleAvailabilityZones:  types.BoolValue(multipleAZs),
			PreferredAvailabilityZones: preferredAZs,
			NetworkingDeploymentCIDR:   types.StringValue(deploymentCIDR),
			NetworkingVPCID:            types.StringValue(vpcID),
			Networks:                   networks,
		})
	}
	return types.SetValueFrom(ctx, subscriptionRegionElemType, models)
}

func subscriptionNetworksFromAPI(ctx context.Context, networks []*subscriptions.Networking) (types.List, diag.Diagnostics) {
	models := make([]utils.CloudNetworkModel, 0, len(networks))
	for _, network := range networks {
		models = append(models, utils.CloudNetworkModel{
			NetworkingSubnetID:       types.StringValue(redis.StringValue(network.SubnetID)),
			NetworkingDeploymentCIDR: types.StringValue(redis.StringValue(network.DeploymentCIDR)),
			NetworkingVPCID:          types.StringValue(redis.StringValue(network.VPCId)),
		})
	}
	return types.ListValueFrom(ctx, subscriptionNetworkElemType, models)
}

func paymentMethodIDFromModel(value types.String) (*int, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(value.ValueString())
	if err != nil {
		return nil, err
	}
	return redis.Int(id), nil
}

// resolveCreationPlanComputedValues replaces unknown computed creation_plan values with
// the zero values that SDKv2 stored for them.
func resolveCreationPlanComputedValues(ctx context.Context, plan *ProSubscriptionResourceModel) diag.Diagnostics {
	var plans []CreationPlanModel
	diags := plan.CreationPlan.ElementsAs(ctx, &plans, false)
	if diags.HasError() {
		return diags
	}
	for i := range plans {
		if plans[i].MemoryLimitInGB.IsUnknown() {
			plans[i].MemoryLimitInGB = types.Float64Value(0)
		}
		if plans[i].DatasetSizeInGB.IsUnknown() {
			plans[i].DatasetSizeInGB = types.Float64Value(0)
		}
		if plans[i].QueryPerformanceFactor.IsUnknown() {
			plans[i].QueryPerformanceFactor = types.StringValue("")
		}
		if plans[i].AverageItemSizeInBytes.IsUnknown() {
			plans[i].AverageItemSizeInBytes = types.Int64Value(0)
		}
		if plans[i].RamPercentage.IsUnknown() {
			plans[i].RamPercentage = types.Int64Value(0)
		}
		if plans[i].Modules.IsUnknown() {
			plans[i].Modules = types.ListValueMust(types.StringType, []attr.Value{})
		}
	}
	plan.CreationPlan, diags = types.ListValueFrom(ctx, creationPlanElemType, plans)
	return diags
}

func buildCreateCloudProviders(ctx context.Context, cloudProvider types.List) ([]*subscriptions.CreateCloudProvider, diag.Diagnostics) {
	var providers []SubscriptionCloudProviderModel
	diags := cloudProvider.ElementsAs(ctx, &providers, false)
	if diags.HasError() {
		return nil, diags
	}

	createCloudProviders := make([]*subscriptions.CreateCloudProvider, 0, len(providers))
	for _, provider := range providers {
		cloudAccountID, err := strconv.Atoi(provider.CloudAccountID.ValueString())
		if err != nil {
			diags.AddError("Invalid cloud_account_id", err.Error())
			return nil, diags
		}

		var regions []SubscriptionRegionModel
		diags.Append(provider.Region.ElementsAs(ctx, &regions, false)...)
		if diags.HasError() {
			return nil, diags
		}

		createRegions := make([]*subscriptions.CreateRegion, 0, len(regions))
		for _, region := range regions {
			var preferredAZs []string
			if !region.PreferredAvailabilityZones.IsNull() && !region.PreferredAvailabilityZones.IsUnknown() {
				diags.Append(region.PreferredAvailabilityZones.ElementsAs(ctx, &preferredAZs, false)...)
				if diags.HasError() {
					return nil, diags
				}
			}

			createRegion := subscriptions.CreateRegion{
				Region:                     redis.String(region.Region.ValueString()),
				MultipleAvailabilityZones:  redis.Bool(region.MultipleAvailabilityZones.ValueBool()),
				PreferredAvailabilityZones: redis.StringSlice(preferredAZs...),
			}

			if v := region.NetworkingDeploymentCIDR.ValueString(); v != "" {
				createRegion.Networking = &subscriptions.CreateNetworking{
					DeploymentCIDR: redis.String(v),
				}
			}

			if v := region.NetworkingVPCID.ValueString(); v != "" {
				if createRegion.Networking == nil {
					createRegion.Networking = &subscriptions.CreateNetworking{}
				}
				createRegion.Networking.VPCId = redis.String(v)
			}

			createRegions = append(createRegions, &createRegion)
		}

		createCloudProvider := &subscriptions.CreateCloudProvider{
			Provider:       redis.String(provider.Provider.ValueString()),
			CloudAccountID: redis.Int(cloudAccountID),
			Regions:        createRegions,
		}

		if tags := provider.ResourceTags.Elements(); len(tags) > 0 {
			resourceTags := make([]*subscriptions.ResourceTag, 0, len(tags))
			for k, v := range tags {
				resourceTags = append(resourceTags, &subscriptions.ResourceTag{
					Key:   redis.String(k),
					Value: redis.String(v.(types.String).ValueString()),
				})
			}
			createCloudProvider.ResourceTags = resourceTags
		}

		createCloudProviders = append(createCloudProviders, createCloudProvider)
	}

	return createCloudProviders, diags
}

// BuildSubscriptionCreatePlanDatabases builds the databases of the subscription creation plan.
func BuildSubscriptionCreatePlanDatabases(ctx context.Context, memoryStorage string, plan CreationPlanModel) ([]*subscriptions.CreateDatabase, diag.Diagnostics) {
	var diags diag.Diagnostics
	createDatabases := make([]*subscriptions.CreateDatabase, 0)

	dbName := "creation-plan-db-"
	idx := 1
	throughputMeasurementBy := plan.ThroughputMeasurementBy.ValueString()
	throughputMeasurementValue := int(plan.ThroughputMeasurementValue.ValueInt64())
	averageItemSizeInBytes := int(plan.AverageItemSizeInBytes.ValueInt64())
	ramPercentage := int(plan.RamPercentage.ValueInt64())
	numDatabases := int(plan.Quantity.ValueInt64())
	supportOSSClusterAPI := plan.SupportOSSClusterAPI.ValueBool()
	replication := plan.Replication.ValueBool()
	memoryLimitInGB := plan.MemoryLimitInGB.ValueFloat64()
	datasetSizeInGB := plan.DatasetSizeInGB.ValueFloat64()
	queryPerformanceFactor := plan.QueryPerformanceFactor.ValueString()

	var moduleNames []string
	if !plan.Modules.IsNull() && !plan.Modules.IsUnknown() {
		diags.Append(plan.Modules.ElementsAs(ctx, &moduleNames, false)...)
		if diags.HasError() {
			return nil, diags
		}
	}
	planModules := redis.StringSlice(moduleNames...)

	if memoryStorage == databases.MemoryStorageRam && averageItemSizeInBytes != 0 {
		// TODO This should be changed to an error when releasing 2.0 of the provider
		diags.AddWarning(
			"`average_item_size_in_bytes` not applicable for `ram` memory storage ",
			"`average_item_size_in_bytes` is only applicable when `memory_storage` is `ram-and-flash`. This will be an error in a future release of the provider",
		)
	}

	if memoryStorage == databases.MemoryStorageRam && ramPercentage != 0 {
		diags.AddWarning(
			"`ram_percentage` not applicable for `ram` memory storage ",
			"`ram_percentage` is only applicable when `memory_storage` is `ram-and-flash`",
		)
	}

	containsGraph := false
	for _, module := range planModules {
		if *module == "RedisGraph" {
			containsGraph = true
			break
		}
	}

	if !containsGraph || len(planModules) <= 1 {
		var modules []*subscriptions.CreateModules
		for _, v := range planModules {
			modules = append(modules, &subscriptions.CreateModules{Name: v})
		}
		createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules, throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, ramPercentage, supportOSSClusterAPI, replication, numDatabases, queryPerformanceFactor)...)
	} else {
		// make RedisGraph module the first module, then append the rest of the modules
		var modules []*subscriptions.CreateModules
		modules = append(modules, &subscriptions.CreateModules{Name: redis.String("RedisGraph")})
		for _, v := range planModules {
			if *v != "RedisGraph" {
				modules = append(modules, &subscriptions.CreateModules{Name: v})
			}
		}
		// create a DB with the RedisGraph module
		createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules[:1], throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, ramPercentage, supportOSSClusterAPI, replication, 1, queryPerformanceFactor)...)
		if numDatabases == 1 {
			// create one extra DB with all other modules
			createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules[1:], throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, ramPercentage, supportOSSClusterAPI, replication, 1, queryPerformanceFactor)...)
		} else if numDatabases > 1 {
			// create the remaining DBs with all other modules
			createDatabases = append(createDatabases, createDatabase(dbName, &idx, modules[1:], throughputMeasurementBy, throughputMeasurementValue, memoryLimitInGB, datasetSizeInGB, averageItemSizeInBytes, ramPercentage, supportOSSClusterAPI, replication, numDatabases-1, queryPerformanceFactor)...)
		}
	}
	return createDatabases, diags
}

//nolint:unparam
func createDatabase(dbName string, idx *int, modules []*subscriptions.CreateModules, throughputMeasurementBy string, throughputMeasurementValue int, memoryLimitInGB float64, datasetSizeInGB float64, averageItemSizeInBytes int, ramPercentage int, supportOSSClusterAPI bool, replication bool, numDatabases int, queryPerformanceFactor string) []*subscriptions.CreateDatabase {
	createThroughput := &subscriptions.CreateThroughput{
		By:    redis.String(throughputMeasurementBy),
		Value: redis.Int(throughputMeasurementValue),
	}
	if len(modules) > 0 {
		// if RedisGraph is in the modules, set throughput to operations-per-second and convert the value
		if *modules[0].Name == "RedisGraph" {
			if *createThroughput.By == "number-of-shards" {
				createThroughput.By = redis.String("operations-per-second")
				if replication {
					createThroughput.Value = redis.Int(*createThroughput.Value * 500)
				} else {
					createThroughput.Value = redis.Int(*createThroughput.Value * 250)
				}
			}
		}
	}
	var dbs []*subscriptions.CreateDatabase
	for i := 0; i < numDatabases; i++ {
		createDatabase := subscriptions.CreateDatabase{
			Name:                  redis.String(dbName + strconv.Itoa(*idx)),
			Protocol:              redis.String("redis"),
			SupportOSSClusterAPI:  redis.Bool(supportOSSClusterAPI),
			Replication:           redis.Bool(replication),
			ThroughputMeasurement: createThroughput,
			Quantity:              redis.Int(1),
			Modules:               modules,
		}
		if averageItemSizeInBytes > 0 {
			createDatabase.AverageItemSizeInBytes = redis.Int(averageItemSizeInBytes)
		}

		if ramPercentage > 0 {
			createDatabase.RamPercentage = redis.Int(ramPercentage)
		}

		if datasetSizeInGB > 0 {
			createDatabase.DatasetSizeInGB = redis.Float64(datasetSizeInGB)
		}
		if memoryLimitInGB > 0 {
			createDatabase.MemoryLimitInGB = redis.Float64(memoryLimitInGB)
		}

		if queryPerformanceFactor != "" {
			createDatabase.QueryPerformanceFactor = redis.String(queryPerformanceFactor)
		}

		*idx++
		dbs = append(dbs, &createDatabase)
	}
	return dbs
}

func frameworkDiags(sdkDiags sdkdiag.Diagnostics) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, d := range sdkDiags {
		if d.Severity == sdkdiag.Error {
			diags.AddError(d.Summary, d.Detail)
		} else {
			diags.AddWarning(d.Summary, d.Detail)
		}
	}
	return diags
}
