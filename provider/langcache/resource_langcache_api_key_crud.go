package langcache

import (
	"context"
	"errors"

	langcacheapi "github.com/RedisLabs/rediscloud-go-api/service/langcache"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *langCacheAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LangCacheAPIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.Client.LangCache.CreateAPIKey(ctx, plan.CacheID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create LangCache API key", err.Error())
		return
	}

	plan.ID = types.StringValue(created.Info.APIKeyID)
	plan.APIKey = types.StringValue(created.APIKey)
	plan.ObfuscatedToken = types.StringValue(created.Info.ObfuscatedToken)
	plan.CreatedAt = types.Int64Value(created.Info.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *langCacheAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LangCacheAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.Client.LangCache.GetAPIKey(ctx, state.CacheID.ValueString(), state.ID.ValueString())
	if err != nil {
		var notFound *langcacheapi.APIKeyNotFound
		var cacheNotFound *langcacheapi.NotFound
		if errors.As(err, &notFound) || errors.As(err, &cacheNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read LangCache API key", err.Error())
		return
	}

	state.ID = types.StringValue(key.APIKeyID)
	state.Name = types.StringValue(key.KeyName)
	state.ObfuscatedToken = types.StringValue(key.ObfuscatedToken)
	state.CreatedAt = types.Int64Value(key.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *langCacheAPIKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unable to update LangCache API key", "LangCache API keys cannot be updated; configuration changes require replacement.")
}

func (r *langCacheAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LangCacheAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Client.LangCache.DeleteAPIKey(ctx, state.CacheID.ValueString(), state.ID.ValueString())
	if err == nil {
		return
	}
	var notFound *langcacheapi.APIKeyNotFound
	if errors.As(err, &notFound) {
		return
	}
	resp.Diagnostics.AddError("Failed to delete LangCache API key", err.Error())
}
