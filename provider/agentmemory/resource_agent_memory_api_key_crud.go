package agentmemory

import (
	"context"
	"errors"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *agentMemoryAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentMemoryAPIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	operationTimeout, diags := plan.Timeouts.Create(ctx, agentMemoryAPIKeyDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	created, err := r.client.Client.AgentMemory.CreateAPIKey(ctx, plan.StoreID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Agent Memory API key", err.Error())
		return
	}

	plan.ID = types.StringPointerValue(created.Info.APIKeyID)
	plan.APIKey = types.StringPointerValue(created.APIKey)
	plan.ObfuscatedToken = types.StringPointerValue(created.Info.ObfuscatedToken)
	plan.CreatedAt = types.Int64PointerValue(created.Info.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentMemoryAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentMemoryAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	operationTimeout, diags := state.Timeouts.Read(ctx, agentMemoryAPIKeyDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	key, err := r.client.Client.AgentMemory.GetAPIKey(ctx, state.StoreID.ValueString(), state.ID.ValueString())
	if err != nil {
		var notFound *agentmemoryapi.APIKeyNotFound
		var storeNotFound *agentmemoryapi.NotFound
		if errors.As(err, &notFound) || errors.As(err, &storeNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Agent Memory API key", err.Error())
		return
	}

	// An incomplete name must not become drift that replaces a key with a generated secret.
	if redis.StringValue(key.KeyName) == "" {
		resp.Diagnostics.AddError("Invalid Agent Memory API key response", "The API response omitted the key name. The tracked key and stored secret were preserved.")
		return
	}
	state.ID = types.StringPointerValue(key.APIKeyID)
	state.Name = types.StringPointerValue(key.KeyName)
	state.ObfuscatedToken = types.StringPointerValue(key.ObfuscatedToken)
	state.CreatedAt = types.Int64PointerValue(key.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *agentMemoryAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AgentMemoryAPIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.StoreID.Equal(state.StoreID) || !plan.Name.Equal(state.Name) {
		resp.Diagnostics.AddError("Unable to update Agent Memory API key", "Agent Memory API keys cannot be updated; configuration changes require replacement.")
		return
	}
	// Timeout changes are local metadata; keep the existing ID and generated secret.
	state.Timeouts = plan.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *agentMemoryAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentMemoryAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	operationTimeout, diags := state.Timeouts.Delete(ctx, agentMemoryAPIKeyDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	err := r.client.Client.AgentMemory.DeleteAPIKey(ctx, state.StoreID.ValueString(), state.ID.ValueString())
	if err == nil {
		return
	}
	var notFound *agentmemoryapi.APIKeyNotFound
	if errors.As(err, &notFound) {
		return
	}
	resp.Diagnostics.AddError("Failed to delete Agent Memory API key", err.Error())
}
