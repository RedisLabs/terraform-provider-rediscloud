package agentmemory

import (
	"context"
	"errors"

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

	created, err := r.client.Client.AgentMemory.CreateAPIKey(ctx, plan.StoreID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Agent Memory API key", err.Error())
		return
	}

	plan.ID = types.StringValue(created.Info.APIKeyID)
	plan.APIKey = types.StringValue(created.APIKey)
	plan.ObfuscatedToken = types.StringValue(created.Info.ObfuscatedToken)
	plan.CreatedAt = types.Int64Value(created.Info.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentMemoryAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentMemoryAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	state.ID = types.StringValue(key.APIKeyID)
	state.Name = types.StringValue(key.KeyName)
	state.ObfuscatedToken = types.StringValue(key.ObfuscatedToken)
	state.CreatedAt = types.Int64Value(key.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *agentMemoryAPIKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unable to update Agent Memory API key", "Agent Memory API keys cannot be updated; configuration changes require replacement.")
}

func (r *agentMemoryAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentMemoryAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
