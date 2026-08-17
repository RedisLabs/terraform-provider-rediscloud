package langcache

import (
	"context"
	"errors"

	langcacheapi "github.com/RedisLabs/rediscloud-go-api/service/langcache"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *langCacheResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LangCacheResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var attributes []string
	resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &attributes, false)...)
	var searchStrategies []string
	if !plan.SearchStrategies.IsNull() && !plan.SearchStrategies.IsUnknown() {
		resp.Diagnostics.Append(plan.SearchStrategies.ElementsAs(ctx, &searchStrategies, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	cacheID, err := r.client.Client.LangCache.Create(ctx, langcacheapi.CreateCache{
		Name:                   plan.Name.ValueString(),
		Database:               langcacheapi.DatabaseConfig{DatabaseID: plan.DatabaseID.ValueString()},
		EmbeddingModel:         embeddingModelFromPlan(plan),
		DefaultSearchThreshold: plan.DefaultSearchThreshold.ValueFloat64(),
		DefaultTTLMillis:       plan.DefaultTTLMillis.ValueInt64(),
		Attributes:             attributes,
		SearchStrategies:       searchStrategies,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create LangCache", err.Error())
		return
	}

	plan.ID = types.StringValue(cacheID)
	plan.Status = types.StringValue(langcacheapi.StatusProvisioning)
	plan.Endpoint = types.StringNull()

	cache, readErr := r.client.Client.LangCache.Get(ctx, cacheID)
	if readErr != nil {
		resp.Diagnostics.AddWarning(
			"LangCache created but not yet readable",
			"The LangCache was created and its ID was saved, but the full object could not be read immediately: "+readErr.Error(),
		)
	} else {
		readLangCacheIntoModel(ctx, cache, &plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *langCacheResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state LangCacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cache, err := r.client.Client.LangCache.Get(ctx, state.ID.ValueString())
	if err != nil {
		var notFound *langcacheapi.NotFound
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read LangCache", err.Error())
		return
	}

	readLangCacheIntoModel(ctx, cache, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *langCacheResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LangCacheResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state LangCacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := langcacheapi.UpdateCache{}
	if !plan.Name.Equal(state.Name) {
		value := plan.Name.ValueString()
		update.Name = &value
	}
	if !plan.DefaultSearchThreshold.Equal(state.DefaultSearchThreshold) {
		value := plan.DefaultSearchThreshold.ValueFloat64()
		update.DefaultSearchThreshold = &value
	}
	if !plan.DefaultTTLMillis.Equal(state.DefaultTTLMillis) {
		value := plan.DefaultTTLMillis.ValueInt64()
		update.DefaultTTLMillis = &value
	}
	embeddingChanged := !plan.EmbeddingModelProvider.Equal(state.EmbeddingModelProvider) ||
		!plan.EmbeddingModelName.Equal(state.EmbeddingModelName) ||
		!plan.EmbeddingModelAPIKey.Equal(state.EmbeddingModelAPIKey) ||
		!plan.CustomModelBaseURL.Equal(state.CustomModelBaseURL) ||
		!plan.CustomModelDimensions.Equal(state.CustomModelDimensions)
	if embeddingChanged {
		embeddingModel := embeddingModelFromPlan(plan)
		update.EmbeddingModel = &embeddingModel
	}
	if !plan.Attributes.Equal(state.Attributes) {
		attributes := make([]string, 0)
		resp.Diagnostics.Append(plan.Attributes.ElementsAs(ctx, &attributes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		update.Attributes = &attributes
	}
	if !plan.SearchStrategies.Equal(state.SearchStrategies) {
		strategies := make([]string, 0)
		if !plan.SearchStrategies.IsNull() && !plan.SearchStrategies.IsUnknown() {
			resp.Diagnostics.Append(plan.SearchStrategies.ElementsAs(ctx, &strategies, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		update.SearchStrategies = &strategies
	}

	if err := r.client.Client.LangCache.Update(ctx, plan.ID.ValueString(), update); err != nil {
		resp.Diagnostics.AddError("Failed to update LangCache", err.Error())
		return
	}

	cache, err := r.client.Client.LangCache.Get(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read LangCache after update", err.Error())
		return
	}
	readLangCacheIntoModel(ctx, cache, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func embeddingModelFromPlan(plan LangCacheResourceModel) langcacheapi.EmbeddingModel {
	embeddingModel := langcacheapi.EmbeddingModel{
		Provider: plan.EmbeddingModelProvider.ValueString(),
		Name:     plan.EmbeddingModelName.ValueString(),
		APIKey:   plan.EmbeddingModelAPIKey.ValueStringPointer(),
	}
	if !plan.CustomModelBaseURL.IsNull() && !plan.CustomModelBaseURL.IsUnknown() {
		embeddingModel.CustomOptions = &langcacheapi.CustomEmbeddingModelOptions{
			BaseURL:    plan.CustomModelBaseURL.ValueString(),
			Dimensions: plan.CustomModelDimensions.ValueInt32(),
		}
	}
	return embeddingModel
}

func (r *langCacheResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LangCacheResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Client.LangCache.Delete(ctx, state.ID.ValueString(), state.FlushOnDestroy.ValueBool())
	if err == nil {
		return
	}
	var notFound *langcacheapi.NotFound
	if errors.As(err, &notFound) {
		return
	}
	resp.Diagnostics.AddError("Failed to delete LangCache", err.Error())
}

func readLangCacheIntoModel(ctx context.Context, cache *langcacheapi.Cache, state *LangCacheResourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(cache.CacheID)
	state.Name = types.StringValue(cache.CacheName)
	state.DatabaseID = types.StringValue(cache.DatabaseID)
	state.EmbeddingModelProvider = types.StringValue(cache.EmbeddingProvider.Provider)
	state.EmbeddingModelName = types.StringValue(cache.EmbeddingProvider.Model)
	state.DefaultSearchThreshold = types.Float64Value(cache.DefaultSearchThreshold)
	state.DefaultTTLMillis = types.Int64Value(cache.DefaultTTLMillis)
	state.Endpoint = types.StringValue(cache.Endpoint)
	state.Status = types.StringValue(cache.Status)

	attributes, attributeDiags := types.SetValueFrom(ctx, types.StringType, cache.Attributes)
	diags.Append(attributeDiags...)
	state.Attributes = attributes
	strategies, strategyDiags := types.ListValueFrom(ctx, types.StringType, cache.SearchStrategies)
	diags.Append(strategyDiags...)
	state.SearchStrategies = strategies
}
