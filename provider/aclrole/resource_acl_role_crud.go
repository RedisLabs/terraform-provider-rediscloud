package aclrole

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/access_control_lists/roles"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// The SDKv2 implementation declared these per-operation default timeouts.
const (
	aclRoleDefaultTimeout     = 5 * time.Minute
	aclRoleDefaultReadTimeout = 3 * time.Minute
)

func (r *aclRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AclRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, aclRoleDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	rules, diags := rulesToAPI(ctx, plan.Rule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := r.client.Client.Roles.Create(ctx, roles.CreateRoleRequest{
		Name:       plan.Name.ValueStringPointer(),
		RedisRules: rules,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ACL role", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(id))

	if err := waitForAclRoleToSettle(ctx, id, r.client, createTimeout); err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("ACL role failed to become active", err.Error())
		return
	}

	if notFound := r.readAclRoleIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.Diagnostics.AddError("ACL role not found", "The ACL role was not found immediately after creation.")
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AclRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := state.Timeouts.Read(ctx, aclRoleDefaultReadTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL role ID", err.Error())
		return
	}

	if notFound := r.readAclRoleIntoModel(ctx, id, &state, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *aclRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AclRoleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, aclRoleDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL role ID", err.Error())
		return
	}

	if !plan.Name.Equal(state.Name) || !plan.Rule.Equal(state.Rule) {
		rules, diags := rulesToAPI(ctx, plan.Rule)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		err := r.client.Client.Roles.Update(ctx, id, roles.CreateRoleRequest{
			Name:       plan.Name.ValueStringPointer(),
			RedisRules: rules,
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update ACL role", err.Error())
			return
		}

		if err := waitForAclRoleToSettle(ctx, id, r.client, updateTimeout); err != nil {
			resp.Diagnostics.AddError("ACL role failed to become active", err.Error())
			return
		}
	}

	plan.ID = state.ID
	if notFound := r.readAclRoleIntoModel(ctx, id, &plan, &resp.Diagnostics); notFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *aclRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AclRoleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := state.Timeouts.Delete(ctx, aclRoleDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ACL role ID", err.Error())
		return
	}

	// ACL roles can still flip between active and pending shortly after a create or update.
	if err := waitForAclRoleToBeActive(ctx, id, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("ACL role failed to become active", err.Error())
		return
	}

	if err := r.client.Client.Roles.Delete(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete ACL role", err.Error())
		return
	}

	if err := waitForAclRoleToBeDeleted(ctx, id, r.client, deleteTimeout); err != nil {
		resp.Diagnostics.AddError("ACL role failed to delete", err.Error())
		return
	}
}

func (r *aclRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected a numeric ACL role ID, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// readAclRoleIntoModel maps the API role onto the model and reports whether it no longer exists.
func (r *aclRoleResource) readAclRoleIntoModel(ctx context.Context, id int, model *AclRoleResourceModel, diags *diag.Diagnostics) (notFound bool) {
	role, err := r.client.Client.Roles.Get(ctx, id)
	if err != nil {
		notFoundErr := &roles.NotFound{}
		if errors.As(err, &notFoundErr) {
			return true
		}
		diags.AddError("Failed to read ACL role", err.Error())
		return false
	}

	priorRegions, priorDiags := emptyRegionsFrom(ctx, model.Rule)
	diags.Append(priorDiags...)
	if diags.HasError() {
		return false
	}

	rules, rulesDiags := rulesFromAPI(ctx, role.RedisRules, priorRegions)
	diags.Append(rulesDiags...)
	if diags.HasError() {
		return false
	}

	model.ID = types.StringValue(strconv.Itoa(id))
	model.Name = types.StringValue(redis.StringValue(role.Name))
	model.Rule = rules

	return false
}

type ruleDatabaseKey struct {
	ruleName     string
	subscription int64
	database     int64
}

// emptyRegionsFrom decides how a database without API regions is stored. The API does not
// distinguish an omitted regions argument from an empty one, so a known empty set is kept
// only where the prior model already held one; otherwise regions stay null, as configured.
func emptyRegionsFrom(ctx context.Context, prior RuleSetValue) (func(string, int64, int64) types.Set, diag.Diagnostics) {
	knownEmpty := map[ruleDatabaseKey]bool{}
	nullRegions := func(ruleName string, subscription, database int64) types.Set {
		if knownEmpty[ruleDatabaseKey{ruleName, subscription, database}] {
			return types.SetValueMust(types.StringType, []attr.Value{})
		}
		return types.SetNull(types.StringType)
	}

	if prior.IsNull() || prior.IsUnknown() {
		return nullRegions, nil
	}

	rules, diags := prior.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}
	for _, rule := range rules {
		if rule.Database.IsNull() || rule.Database.IsUnknown() {
			continue
		}
		databases, databaseDiags := rule.Database.AsModels(ctx)
		diags.Append(databaseDiags...)
		if diags.HasError() {
			return nil, diags
		}
		for _, database := range databases {
			if !database.Regions.IsNull() && !database.Regions.IsUnknown() && len(database.Regions.Elements()) == 0 {
				knownEmpty[ruleDatabaseKey{rule.Name.ValueString(), database.Subscription.ValueInt64(), database.Database.ValueInt64()}] = true
			}
		}
	}

	return nullRegions, diags
}

func rulesToAPI(ctx context.Context, ruleSet RuleSetValue) ([]*roles.CreateRuleInRoleRequest, diag.Diagnostics) {
	requests := make([]*roles.CreateRuleInRoleRequest, 0)
	if ruleSet.IsNull() || ruleSet.IsUnknown() {
		return requests, nil
	}

	rules, diags := ruleSet.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}

	for _, rule := range rules {
		databaseRequests := make([]*roles.CreateDatabaseInRuleInRoleRequest, 0)
		if !rule.Database.IsNull() && !rule.Database.IsUnknown() {
			databases, databaseDiags := rule.Database.AsModels(ctx)
			diags.Append(databaseDiags...)
			if diags.HasError() {
				return nil, diags
			}

			for _, database := range databases {
				var regions []*string
				if !database.Regions.IsNull() && !database.Regions.IsUnknown() {
					var values []string
					diags.Append(database.Regions.ElementsAs(ctx, &values, false)...)
					if diags.HasError() {
						return nil, diags
					}
					regions = redis.StringSlice(values...)
				}

				databaseRequests = append(databaseRequests, &roles.CreateDatabaseInRuleInRoleRequest{
					SubscriptionId: redis.Int(int(database.Subscription.ValueInt64())),
					DatabaseId:     redis.Int(int(database.Database.ValueInt64())),
					Regions:        regions,
				})
			}
		}

		requests = append(requests, &roles.CreateRuleInRoleRequest{
			RuleName:  rule.Name.ValueStringPointer(),
			Databases: databaseRequests,
		})
	}

	return requests, diags
}

// waitForAclRoleToSettle waits twice because ACL roles flip between active and pending a few
// times after a create or update before the API settles.
func waitForAclRoleToSettle(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	if err := waitForAclRoleToBeActive(ctx, id, c, timeout); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
	}

	return waitForAclRoleToBeActive(ctx, id, c, timeout)
}

func waitForAclRoleToBeActive(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		PollInterval: c.WaitPollInterval(0),
		Delay:        c.WaitDelay(5 * time.Second),
		Pending:      []string{roles.StatusPending},
		Target:       []string{roles.StatusActive},
		Timeout:      timeout,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for role %d to be active", id)

			role, err := c.Client.Roles.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(role.Status), redis.StringValue(role.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}

func waitForAclRoleToBeDeleted(ctx context.Context, id int, c *client.ApiClient, timeout time.Duration) error {
	return retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		role, err := c.Client.Roles.Get(ctx, id)
		if err != nil {
			notFound := &roles.NotFound{}
			if errors.As(err, &notFound) {
				return nil
			}
			return retry.NonRetryableError(fmt.Errorf("error getting role: %w", err))
		}

		if role != nil {
			return retry.RetryableError(fmt.Errorf("expected role %d to be deleted but was in state %s", id, redis.StringValue(role.Status)))
		}
		return retry.NonRetryableError(fmt.Errorf("unexpected error getting role"))
	})
}
