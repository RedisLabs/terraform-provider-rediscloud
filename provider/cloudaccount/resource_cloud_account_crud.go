package cloudaccount

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// cloudAccountDefaultTimeout matches the per-operation timeout the SDKv2
// implementation declared for this resource.
const cloudAccountDefaultTimeout = 5 * time.Minute

// Create creates a new cloud account and waits for it to become active.
func (r *cloudAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CloudAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, cloudAccountDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	id, err := r.client.Client.CloudAccount.Create(ctx, cloud_accounts.CreateCloudAccount{
		AccessKeyID:     plan.AccessKeyID.ValueStringPointer(),
		AccessSecretKey: plan.AccessSecretKey.ValueStringPointer(),
		ConsoleUsername: plan.ConsoleUsername.ValueStringPointer(),
		ConsolePassword: plan.ConsolePassword.ValueStringPointer(),
		Name:            plan.Name.ValueStringPointer(),
		Provider:        plan.ProviderType.ValueStringPointer(),
		SignInLoginURL:  plan.SignInLoginURL.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create cloud account", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(id))

	lastStatus, err := waitForCloudAccountToBeActive(ctx, id, r.client, createTimeout)
	if err != nil {
		// The account was created but did not become active within the timeout.
		// Persist what we know — the configured values (already in plan) + id +
		// the last observed status — so the account is tracked and recoverable on
		// a re-apply instead of leaking. Mirrors the SDKv2 behaviour of calling
		// d.SetId() before waiting.
		plan.Status = lastStatus
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Cloud account failed to become active", err.Error())
		return
	}

	if notFound := r.readCloudAccountIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("Cloud account not found", "The cloud account was not found immediately after creation.")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *cloudAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CloudAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, cloudAccountDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cloud account ID", err.Error())
		return
	}

	if notFound := r.readCloudAccountIntoModel(ctx, id, &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the cloud account.
func (r *cloudAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CloudAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, cloudAccountDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	id, err := strconv.Atoi(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cloud account ID", err.Error())
		return
	}

	err = r.client.Client.CloudAccount.Update(ctx, id, cloud_accounts.UpdateCloudAccount{
		AccessKeyID:     plan.AccessKeyID.ValueStringPointer(),
		AccessSecretKey: plan.AccessSecretKey.ValueStringPointer(),
		ConsoleUsername: plan.ConsoleUsername.ValueStringPointer(),
		ConsolePassword: plan.ConsolePassword.ValueStringPointer(),
		Name:            plan.Name.ValueStringPointer(),
		SignInLoginURL:  plan.SignInLoginURL.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update cloud account", err.Error())
		return
	}

	if _, err := waitForCloudAccountToBeActive(ctx, id, r.client, updateTimeout); err != nil {
		resp.Diagnostics.AddError("Cloud account failed to become active", err.Error())
		return
	}

	if notFound := r.readCloudAccountIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the cloud account.
func (r *cloudAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CloudAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, cloudAccountDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid cloud account ID", err.Error())
		return
	}

	if err := r.client.Client.CloudAccount.Delete(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete cloud account", err.Error())
		return
	}

	if err := waitForCloudAccountToBeDeleted(ctx, id, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("Cloud account failed to delete", err.Error())
		return
	}
}

// ImportState imports an existing cloud account by its numeric ID.
func (r *cloudAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric cloud account ID, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readCloudAccountIntoModel fetches the cloud account and maps the API-backed
// fields onto the model. The access_secret_key, console_username,
// console_password and sign_in_login_url fields are never returned by the API,
// so they are deliberately left untouched to preserve the prior config/state
// values. It returns true if the account no longer exists.
func (r *cloudAccountResource) readCloudAccountIntoModel(ctx context.Context, id int, model *CloudAccountResourceModel, diags *diag.Diagnostics) (notFound bool) {
	account, err := r.client.Client.CloudAccount.Get(ctx, id)
	if err != nil {
		notFoundErr := &cloud_accounts.NotFound{}
		if errors.As(err, &notFoundErr) {
			return true
		}
		diags.AddError("Failed to read cloud account", err.Error())
		return false
	}

	model.ID = types.StringValue(strconv.Itoa(id))
	model.AccessKeyID = types.StringPointerValue(account.AccessKeyID)
	model.Name = types.StringPointerValue(account.Name)
	model.ProviderType = types.StringPointerValue(account.Provider)
	model.Status = types.StringPointerValue(account.Status)

	return false
}

// waitForCloudAccountToBeActive polls until the account reaches the active state.
// It returns the last observed status alongside any error, so callers can persist
// what was seen (e.g. on a timeout) without making an extra API call.
func waitForCloudAccountToBeActive(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) (types.String, error) {
	lastStatus := types.StringNull()
	wait := &retry.StateChangeConf{
		Delay: 10 * time.Second,
		Pending: []string{
			cloud_accounts.StatusDraft,
			cloud_accounts.StatusChangeDraft,
			cloud_accounts.StatusPending,
			cloud_accounts.StatusChangePending},
		Target:  []string{cloud_accounts.StatusActive},
		Timeout: timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for cloud account %d to be active", id)

			account, err := c.Client.CloudAccount.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			lastStatus = types.StringPointerValue(account.Status)
			log.Printf("[DEBUG] Cloud account status %v", lastStatus)
			return lastStatus, lastStatus.ValueString(), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return lastStatus, err
}

// waitForCloudAccountToBeDeleted waits for the cloud account to be deleted using retry.StateChangeConf.
func waitForCloudAccountToBeDeleted(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	deletePendingState := cloud_accounts.StatusPending
	deleteCompleteState := cloud_accounts.StatusDeleted

	wait := &retry.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{deletePendingState},
		Target:  []string{deleteCompleteState},
		Timeout: timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for cloud account %d to be deleted", id)

			if _, err := c.Client.CloudAccount.Get(ctx, id); err != nil {
				notFoundErr := &cloud_accounts.NotFound{}
				if errors.As(err, &notFoundErr) {
					return deleteCompleteState, deleteCompleteState, nil
				}
				return nil, "", err
			}

			return deletePendingState, deletePendingState, nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
