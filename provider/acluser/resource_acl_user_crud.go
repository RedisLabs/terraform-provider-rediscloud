package acluser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/users"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// The SDKv2 implementation declared these per-operation default timeouts.
const (
	aclUserDefaultTimeout     = 5 * time.Minute
	aclUserDefaultReadTimeout = 3 * time.Minute
)

func (r *aclUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AclUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, aclUserDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	id, err := r.client.Client.Users.Create(ctx, users.CreateUserRequest{
		Name:     plan.Name.ValueStringPointer(),
		Role:     plan.Role.ValueStringPointer(),
		Password: plan.Password.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ACL user", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(id))

	if err := waitForAclUserToSettle(ctx, id, r.client, createTimeout); err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("ACL user failed to become active", err.Error())
		return
	}

	if notFound := r.readAclUserIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("ACL user not found", "The ACL user was not found immediately after creation.")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AclUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, aclUserDefaultReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL user ID", err.Error())
		return
	}

	if notFound := r.readAclUserIntoModel(ctx, id, &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aclUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AclUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, aclUserDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL user ID", err.Error())
		return
	}

	if !plan.Role.Equal(state.Role) || !plan.Password.Equal(state.Password) {
		err := r.client.Client.Users.Update(ctx, id, users.UpdateUserRequest{
			Role:     plan.Role.ValueStringPointer(),
			Password: plan.Password.ValueStringPointer(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update ACL user", err.Error())
			return
		}

		if err := waitForAclUserToSettle(ctx, id, r.client, updateTimeout); err != nil {
			resp.Diagnostics.AddError("ACL user failed to become active", err.Error())
			return
		}
	}

	plan.ID = state.ID
	if notFound := r.readAclUserIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AclUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, aclUserDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL user ID", err.Error())
		return
	}

	// ACL users can still flip between active and pending shortly after a create or update.
	if err := waitForAclUserToBeActive(ctx, id, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("ACL user failed to become active", err.Error())
		return
	}

	if err := r.client.Client.Users.Delete(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete ACL user", err.Error())
		return
	}

	if err := waitForAclUserToBeDeleted(ctx, id, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("ACL user failed to delete", err.Error())
		return
	}
}

func (r *aclUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric ACL user ID, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readAclUserIntoModel maps the API user onto the model and reports whether it no longer exists.
// The API never returns the password, so the model keeps its configured or prior value.
func (r *aclUserResource) readAclUserIntoModel(ctx context.Context, id int, model *AclUserResourceModel, diags *diag.Diagnostics) (notFound bool) {
	user, err := r.client.Client.Users.Get(ctx, id)
	if err != nil {
		notFoundErr := &users.NotFound{}
		if errors.As(err, &notFoundErr) {
			return true
		}
		diags.AddError("Failed to read ACL user", err.Error())
		return false
	}

	model.ID = types.StringValue(strconv.Itoa(id))
	model.Name = types.StringValue(redis.StringValue(user.Name))
	model.Role = types.StringValue(redis.StringValue(user.Role))

	return false
}

// waitForAclUserToSettle waits twice because ACL users flip between active and pending a few
// times after a create or update before the API settles.
func waitForAclUserToSettle(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	if err := waitForAclUserToBeActive(ctx, id, c, timeout); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
	}

	return waitForAclUserToBeActive(ctx, id, c, timeout)
}

func waitForAclUserToBeActive(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		PollInterval: c.WaitPollInterval(0),
		Delay:        c.WaitDelay(5 * time.Second),
		Pending:      []string{users.StatusPending},
		Target:       []string{users.StatusActive},
		Timeout:      timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for user %d to be active", id)

			user, err := c.Client.Users.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(user.Status), redis.StringValue(user.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}

func waitForAclUserToBeDeleted(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		user, err := c.Client.Users.Get(ctx, id)
		if err != nil {
			notFound := &users.NotFound{}
			if errors.As(err, &notFound) {
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("error getting user: %w", err))
		}

		if user != nil {
			return retry.RetryableError(fmt.Errorf("expected user %d to be deleted but was not", id))
		}
		return retry.NonRetryableError(fmt.Errorf("unexpected error getting user"))
	})
}
