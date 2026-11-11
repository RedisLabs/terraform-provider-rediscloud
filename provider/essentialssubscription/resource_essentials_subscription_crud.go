package essentialssubscription

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedSubscriptions "github.com/RedisLabs/rediscloud-go-api/service/fixed/subscriptions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	defaultPaymentMethod = "credit-card"

	createTimeout = 30 * time.Minute
	readTimeout   = 10 * time.Minute
	updateTimeout = 30 * time.Minute
	deleteTimeout = 10 * time.Minute
)

// Create creates a new essentials subscription and waits for it to become active.
func (r *essentialsSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EssentialsSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, createTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	paymentMethod := defaultPaymentMethod
	if utils.IsConfigured(plan.PaymentMethod) {
		paymentMethod = plan.PaymentMethod.ValueString()
	}
	paymentMethodID := int(plan.PaymentMethodID.ValueInt64())

	if paymentMethod != defaultPaymentMethod && paymentMethodID != 0 {
		resp.Diagnostics.AddError("Invalid Payment Method", "payment methods aside from credit-card cannot have a payment ID")
		return
	}

	request := fixedSubscriptions.FixedSubscriptionRequest{
		Name:          plan.Name.ValueStringPointer(),
		PlanId:        redis.Int(int(plan.PlanID.ValueInt64())),
		PaymentMethod: redis.String(paymentMethod),
	}
	if paymentMethodID != 0 {
		request.PaymentMethodID = redis.Int(paymentMethodID)
	}

	subId, err := r.client.Client.FixedSubscriptions.Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create Essentials Subscription", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(subId))

	if err := WaitForEssentialsSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		// Persist the ID so the subscription is tracked (and tainted) instead of leaked, as SDKv2 did with SetId.
		resolveUnknowns(&plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Essentials Subscription Failed to Become Active", err.Error())
		return
	}

	notFound, diags := r.readIntoModel(ctx, subId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.Diagnostics.AddError("Essentials Subscription Not Found", "The essentials subscription was not found immediately after creation.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *essentialsSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EssentialsSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, readTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Essentials Subscription ID", err.Error())
		return
	}

	notFound, diags := r.readIntoModel(ctx, subId, &state)
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

// Update updates the subscription name, plan or payment method ID.
func (r *essentialsSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EssentialsSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, updateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Essentials Subscription ID", err.Error())
		return
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	changed := !plan.Name.Equal(state.Name) ||
		!plan.PlanID.Equal(state.PlanID) ||
		!plan.PaymentMethodID.Equal(state.PaymentMethodID)

	if !changed {
		if plan.Status.IsUnknown() {
			plan.Status = state.Status
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	request := fixedSubscriptions.FixedSubscriptionRequest{
		Name:   plan.Name.ValueStringPointer(),
		PlanId: redis.Int(int(plan.PlanID.ValueInt64())),
	}
	// SDKv2 suppressed every payment_method diff after creation, so it always sent the stored value.
	if v := state.PaymentMethod.ValueString(); v != "" {
		request.PaymentMethod = redis.String(v)
	}
	if v := int(plan.PaymentMethodID.ValueInt64()); v != 0 {
		request.PaymentMethodID = redis.Int(v)
	}

	if err := r.client.Client.FixedSubscriptions.Update(ctx, subId, request); err != nil {
		resp.Diagnostics.AddError("Unable to Update Essentials Subscription", err.Error())
		return
	}

	notFound, diags := r.readIntoModel(ctx, subId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the subscription once it is active and waits for the deletion to complete.
func (r *essentialsSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EssentialsSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, deleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Essentials Subscription ID", err.Error())
		return
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := WaitForEssentialsSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		resp.Diagnostics.AddError("Essentials Subscription Failed to Become Active", err.Error())
		return
	}

	if err := r.client.Client.FixedSubscriptions.Delete(ctx, subId); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Essentials Subscription", err.Error())
		return
	}

	if err := waitForEssentialsSubscriptionToBeDeleted(ctx, subId, r.client, timeout); err != nil {
		// SDKv2 cleared the ID before this wait, so the deletion request already removes it from state.
		resp.State.RemoveResource(ctx)
		resp.Diagnostics.AddError("Essentials Subscription Failed to Delete", err.Error())
	}
}

// ImportState imports an existing essentials subscription by its numeric ID.
func (r *essentialsSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric essentials subscription ID, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readIntoModel maps the API subscription onto the model and reports whether it no longer exists.
// payment_method is only taken from the API when the model has no value (create without config, import):
// SDKv2 ignored every later change, and the API does not always echo the requested method.
func (r *essentialsSubscriptionResource) readIntoModel(ctx context.Context, subId int, model *EssentialsSubscriptionResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	subscription, err := r.client.Client.FixedSubscriptions.Get(ctx, subId)
	if err != nil {
		notFound := &fixedSubscriptions.NotFound{}
		if errors.As(err, &notFound) {
			return true, diags
		}
		diags.AddError("Unable to Read Essentials Subscription", err.Error())
		return false, diags
	}

	model.ID = types.StringValue(strconv.Itoa(subId))
	model.Name = types.StringValue(redis.StringValue(subscription.Name))
	model.Status = types.StringValue(redis.StringValue(subscription.Status))
	model.PlanID = types.Int64Value(int64(redis.IntValue(subscription.PlanId)))
	model.PaymentMethodID = types.Int64Value(int64(redis.IntValue(subscription.PaymentMethodID)))
	if model.PaymentMethod.IsNull() || model.PaymentMethod.IsUnknown() {
		model.PaymentMethod = types.StringValue(redis.StringValue(subscription.PaymentMethod))
	}
	model.CreationDate = types.StringValue(redis.TimeValue(subscription.CreationDate).String())

	return false, diags
}

func resolveUnknowns(model *EssentialsSubscriptionResourceModel) {
	if model.Status.IsUnknown() {
		model.Status = types.StringNull()
	}
	if model.PaymentMethod.IsUnknown() {
		model.PaymentMethod = types.StringValue(defaultPaymentMethod)
	}
	if model.PaymentMethodID.IsUnknown() {
		model.PaymentMethodID = types.Int64Null()
	}
	if model.CreationDate.IsUnknown() {
		model.CreationDate = types.StringNull()
	}
}

// WaitForEssentialsSubscriptionToBeActive polls until the essentials subscription is active.
func WaitForEssentialsSubscriptionToBeActive(ctx context.Context, id int, api *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		PollInterval: api.WaitPollInterval(0),
		Delay:        api.WaitDelay(10 * time.Second),
		Pending:      []string{subscriptions.SubscriptionStatusPending},
		Target:       []string{subscriptions.SubscriptionStatusActive},
		Timeout:      timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed subscription %d to be active", id)

			subscription, err := api.Client.FixedSubscriptions.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}

func waitForEssentialsSubscriptionToBeDeleted(ctx context.Context, id int, api *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		PollInterval: api.WaitPollInterval(0),
		Delay:        api.WaitDelay(10 * time.Second),
		Pending:      []string{subscriptions.SubscriptionStatusDeleting},
		Target:       []string{"deleted"},
		Timeout:      timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed subscription %d to be deleted", id)

			subscription, err := api.Client.FixedSubscriptions.Get(ctx, id)
			if err != nil {
				notFound := &fixedSubscriptions.NotFound{}
				if errors.As(err, &notFound) {
					return "deleted", "deleted", nil
				}
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}
