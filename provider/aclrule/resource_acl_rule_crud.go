package aclrule

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/redis_rules"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// The SDKv2 implementation declared these per-operation default timeouts.
const (
	aclRuleDefaultTimeout     = 5 * time.Minute
	aclRuleDefaultReadTimeout = 3 * time.Minute
)

func (r *aclRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AclRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, aclRuleDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	id, err := r.client.Client.RedisRules.Create(ctx, redis_rules.CreateRedisRuleRequest{
		Name:      plan.Name.ValueStringPointer(),
		RedisRule: plan.Rule.ValueStringPointer(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ACL rule", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(id))

	if err := waitForAclRuleToBeActive(ctx, id, r.client, createTimeout); err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("ACL rule failed to become active", err.Error())
		return
	}

	if notFound := r.readAclRuleIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("ACL rule not found", "The ACL rule was not found immediately after creation.")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AclRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, aclRuleDefaultReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL rule ID", err.Error())
		return
	}

	if notFound := r.readAclRuleIntoModel(ctx, id, &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aclRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AclRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, aclRuleDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL rule ID", err.Error())
		return
	}

	if !plan.Name.Equal(state.Name) || !plan.Rule.Equal(state.Rule) {
		request := redis_rules.CreateRedisRuleRequest{
			RedisRule: plan.Rule.ValueStringPointer(),
		}
		if !plan.Name.Equal(state.Name) {
			request.Name = plan.Name.ValueStringPointer()
		}

		if err := r.client.Client.RedisRules.Update(ctx, id, request); err != nil {
			resp.Diagnostics.AddError("Failed to update ACL rule", err.Error())
			return
		}

		if err := waitForAclRuleToBeActive(ctx, id, r.client, updateTimeout); err != nil {
			resp.Diagnostics.AddError("ACL rule failed to become active", err.Error())
			return
		}
	}

	plan.ID = state.ID
	if notFound := r.readAclRuleIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AclRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, aclRuleDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL rule ID", err.Error())
		return
	}

	if err := r.client.Client.RedisRules.Delete(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete ACL rule", err.Error())
		return
	}

	if err := waitForAclRuleToBeDeleted(ctx, id, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("ACL rule failed to delete", err.Error())
		return
	}
}

func (r *aclRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric ACL rule ID, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readAclRuleIntoModel maps the API rule onto the model and reports whether it no longer exists.
func (r *aclRuleResource) readAclRuleIntoModel(ctx context.Context, id int, model *AclRuleResourceModel, diags *diag.Diagnostics) (notFound bool) {
	rule, err := r.client.Client.RedisRules.Get(ctx, id)
	if err != nil {
		notFoundErr := &redis_rules.NotFound{}
		if errors.As(err, &notFoundErr) {
			return true
		}
		diags.AddError("Failed to read ACL rule", err.Error())
		return false
	}

	model.ID = types.StringValue(strconv.Itoa(id))
	model.Name = types.StringValue(redis.StringValue(rule.Name))
	model.Rule = types.StringValue(redis.StringValue(rule.ACL))

	return false
}

func waitForAclRuleToBeActive(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		PollInterval: c.WaitPollInterval(0),
		Delay:        c.WaitDelay(5 * time.Second),
		Pending:      []string{redis_rules.StatusPending},
		Target:       []string{redis_rules.StatusActive},
		Timeout:      timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for rule %d to be active", id)

			rule, err := c.Client.RedisRules.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(rule.Status), redis.StringValue(rule.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}

func waitForAclRuleToBeDeleted(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		rule, err := c.Client.RedisRules.Get(ctx, id)
		if err != nil {
			notFound := &redis_rules.NotFound{}
			if errors.As(err, &notFound) {
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("error getting rule: %w", err))
		}

		if rule != nil {
			return retry.RetryableError(fmt.Errorf("expected rule %d to be deleted but was in state %s", id, redis.StringValue(rule.Status)))
		}
		return retry.NonRetryableError(fmt.Errorf("unexpected error getting rule"))
	})
}
