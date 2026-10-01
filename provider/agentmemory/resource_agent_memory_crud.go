package agentmemory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *agentMemoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentMemoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	storeID, err := r.client.Client.AgentMemory.Create(ctx, agentmemoryapi.CreateStore{
		Name:                     plan.Name.ValueString(),
		DatabaseID:               int(plan.DatabaseID.ValueInt64()),
		ShortMemory:              shortMemoryFromPlan(plan),
		LongTermMemory:           longTermMemoryFromPlan(plan),
		LLM:                      modelConfigFromModel(plan.LLM),
		ExtractionStrategy:       stringFromValue(plan.ExtractionStrategy),
		ExtractionCadence:        extractionCadenceFromPlan(plan),
		Summarization:            summarizationFromModel(plan.Summarization),
		LongTermMemoryExclusions: exclusionsFromModel(plan.LongTermMemoryExclusions),
		CustomMemoryTypes:        customMemoryTypesFromModel(plan.CustomMemoryTypes),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Agent Memory service", err.Error())
		return
	}

	plan.ID = types.StringValue(storeID)
	plan.Status = types.StringValue(agentmemoryapi.StatusProvisioning)
	plan.Endpoint = types.StringNull()
	plan.CreatedAt = types.StringNull()

	store, readErr := r.client.Client.AgentMemory.Get(ctx, storeID)
	if readErr != nil {
		resp.Diagnostics.AddWarning(
			"Agent Memory service created but not yet readable",
			"The Agent Memory service was created and its ID was saved, but the full object could not be read immediately: "+readErr.Error(),
		)
	} else {
		readAgentMemoryIntoModel(ctx, store, &plan, false, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentMemoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentMemoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	store, err := r.client.Client.AgentMemory.Get(ctx, state.ID.ValueString())
	if err != nil {
		var notFound *agentmemoryapi.NotFound
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Agent Memory service", err.Error())
		return
	}

	readAgentMemoryIntoModel(ctx, store, &state, false, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *agentMemoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AgentMemoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var state AgentMemoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	update := agentmemoryapi.UpdateStore{}
	if !plan.Name.Equal(state.Name) {
		value := plan.Name.ValueString()
		update.Name = &value
	}
	if !plan.ShortTermTTLSeconds.Equal(state.ShortTermTTLSeconds) {
		update.ShortMemory = shortMemoryFromPlan(plan)
	}
	if !plan.LongTermTTLSeconds.Equal(state.LongTermTTLSeconds) {
		update.LongTermMemory = longTermMemoryFromPlan(plan)
	}
	if !modelConfigEqual(plan.Embedding, state.Embedding) {
		update.LongTermMemory = longTermMemoryFromPlan(plan)
	}
	if !modelConfigEqual(plan.LLM, state.LLM) {
		update.LLM = modelConfigFromModel(plan.LLM)
	}
	if !plan.ExtractionCadenceSeconds.Equal(state.ExtractionCadenceSeconds) {
		update.ExtractionCadence = extractionCadenceFromPlan(plan)
	}
	if plan.Summarization != nil {
		update.Summarization = summarizationFromModel(plan.Summarization)
	}
	if plan.LongTermMemoryExclusions != nil {
		update.LongTermMemoryExclusions = exclusionsFromModel(plan.LongTermMemoryExclusions)
	}
	update.AddCustomMemoryTypes, update.UpdateCustomMemoryTypeStrategies = customMemoryTypeUpdatesFromPlan(plan.CustomMemoryTypes, state.CustomMemoryTypes)

	storeID := state.ID.ValueString()
	if err := r.client.Client.AgentMemory.Update(ctx, storeID, update); err != nil {
		resp.Diagnostics.AddError("Failed to update Agent Memory service", err.Error())
		return
	}

	store, err := r.client.Client.AgentMemory.Get(ctx, storeID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Agent Memory service after update", err.Error())
		return
	}
	readAgentMemoryIntoModel(ctx, store, &plan, false, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *agentMemoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentMemoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Client.AgentMemory.Delete(ctx, state.ID.ValueString())
	if err == nil {
		return
	}
	var notFound *agentmemoryapi.NotFound
	if errors.As(err, &notFound) {
		return
	}
	resp.Diagnostics.AddError("Failed to delete Agent Memory service", err.Error())
}

func shortMemoryFromPlan(plan AgentMemoryResourceModel) *agentmemoryapi.ShortMemoryConfig {
	if plan.ShortTermTTLSeconds.IsNull() || plan.ShortTermTTLSeconds.IsUnknown() {
		return nil
	}
	return &agentmemoryapi.ShortMemoryConfig{TTLSeconds: int(plan.ShortTermTTLSeconds.ValueInt64())}
}

func longTermMemoryFromPlan(plan AgentMemoryResourceModel) *agentmemoryapi.LongTermMemoryConfig {
	if (plan.LongTermTTLSeconds.IsNull() || plan.LongTermTTLSeconds.IsUnknown()) && plan.Embedding == nil {
		return nil
	}
	config := &agentmemoryapi.LongTermMemoryConfig{
		Embedding: modelConfigFromModel(plan.Embedding),
	}
	if !plan.LongTermTTLSeconds.IsNull() && !plan.LongTermTTLSeconds.IsUnknown() {
		config.TTLSeconds = int(plan.LongTermTTLSeconds.ValueInt64())
	}
	return config
}

func extractionCadenceFromPlan(plan AgentMemoryResourceModel) *agentmemoryapi.ExtractionCadenceConfig {
	if plan.ExtractionCadenceSeconds.IsNull() || plan.ExtractionCadenceSeconds.IsUnknown() {
		return nil
	}
	return &agentmemoryapi.ExtractionCadenceConfig{ActiveIntervalSeconds: int(plan.ExtractionCadenceSeconds.ValueInt64())}
}

func readAgentMemoryIntoModel(ctx context.Context, store *agentmemoryapi.Store, state *AgentMemoryResourceModel, includeAPIOnlyConfig bool, diags *diag.Diagnostics) {
	state.ID = types.StringValue(store.StoreID)
	state.Name = types.StringValue(store.Name)
	state.Status = types.StringValue(store.Status)

	databaseID, err := strconv.ParseInt(store.DatabaseID, 10, 64)
	if err != nil {
		diags.AddError("Unexpected Agent Memory database ID", fmt.Sprintf("Expected databaseId to be numeric, got %q.", store.DatabaseID))
		return
	}
	state.DatabaseID = types.Int64Value(databaseID)

	if store.ShortMemory != nil {
		state.ShortTermTTLSeconds = types.Int64Value(int64(store.ShortMemory.TTLSeconds))
	} else {
		state.ShortTermTTLSeconds = types.Int64Null()
	}
	if store.LongTermMemory != nil {
		state.LongTermTTLSeconds = types.Int64Value(int64(store.LongTermMemory.TTLSeconds))
	} else {
		state.LongTermTTLSeconds = types.Int64Null()
	}
	state.ExtractionStrategy = stringValue(store.ExtractionStrategy)
	state.LLM = modelConfigToModel(store.LLM, state.LLM)
	var embedding *agentmemoryapi.ModelStatus
	if store.LongTermMemory != nil {
		embedding = store.LongTermMemory.Embedding
	}
	state.Embedding = modelConfigToModel(embedding, state.Embedding)
	if store.ExtractionCadence != nil {
		state.ExtractionCadenceSeconds = types.Int64Value(int64(store.ExtractionCadence.ActiveIntervalSeconds))
	} else {
		state.ExtractionCadenceSeconds = types.Int64Null()
	}
	// Terraform cannot add optional nested blocks during create/update refresh when
	// they were absent from config. Import is the one path where we intentionally
	// adopt advanced API config even when the previous state is empty.
	if includeAPIOnlyConfig || state.Summarization != nil || store.Summarization == nil {
		state.Summarization = summarizationToModel(store.Summarization)
	}
	if includeAPIOnlyConfig || state.LongTermMemoryExclusions != nil || store.LongTermMemoryExclusions == nil {
		state.LongTermMemoryExclusions = exclusionsToModel(store.LongTermMemoryExclusions)
	}
	if includeAPIOnlyConfig || state.CustomMemoryTypes != nil || len(store.CustomMemoryTypes) == 0 {
		state.CustomMemoryTypes = customMemoryTypesToModel(store.CustomMemoryTypes, state.CustomMemoryTypes)
	}

	switch {
	case store.Endpoint != "":
		state.Endpoint = types.StringValue(store.Endpoint)
	case len(store.Endpoints) > 0:
		state.Endpoint = types.StringValue(store.Endpoints[0].URL)
	default:
		state.Endpoint = types.StringNull()
	}
	state.Endpoints = endpointsToModel(ctx, store.Endpoints, diags)

	if store.CreatedAt != nil {
		state.CreatedAt = types.StringValue(store.CreatedAt.Format(time.RFC3339))
	} else {
		state.CreatedAt = types.StringNull()
	}
}

func readImportedAgentMemoryIntoModel(ctx context.Context, store *agentmemoryapi.Store, state *AgentMemoryResourceModel, diags *diag.Diagnostics) {
	readAgentMemoryIntoModel(ctx, store, state, true, diags)
}

func endpointsToModel(ctx context.Context, endpoints []agentmemoryapi.Endpoint, diags *diag.Diagnostics) types.List {
	endpointObjectType := agentMemoryEndpointObjectType()
	if len(endpoints) == 0 {
		return types.ListNull(endpointObjectType)
	}

	sortedEndpoints := append([]agentmemoryapi.Endpoint(nil), endpoints...)
	sort.SliceStable(sortedEndpoints, func(i, j int) bool {
		if sortedEndpoints[i].Provider != sortedEndpoints[j].Provider {
			return sortedEndpoints[i].Provider < sortedEndpoints[j].Provider
		}
		if sortedEndpoints[i].Region != sortedEndpoints[j].Region {
			return sortedEndpoints[i].Region < sortedEndpoints[j].Region
		}
		return sortedEndpoints[i].URL < sortedEndpoints[j].URL
	})

	values := make([]attr.Value, 0, len(sortedEndpoints))
	for _, endpoint := range sortedEndpoints {
		egressIPs := types.ListNull(types.StringType)
		if endpoint.EgressIPs != nil {
			list, listDiags := types.ListValueFrom(ctx, types.StringType, endpoint.EgressIPs)
			diags.Append(listDiags...)
			if diags.HasError() {
				return types.ListNull(endpointObjectType)
			}
			egressIPs = list
		}

		value, valueDiags := types.ObjectValue(agentMemoryEndpointAttrTypes, map[string]attr.Value{
			"url":           types.StringValue(endpoint.URL),
			"provider":      types.StringValue(endpoint.Provider),
			"region":        types.StringValue(endpoint.Region),
			"egress_ips":    egressIPs,
			"is_accessible": types.BoolValue(endpoint.IsAccessible),
		})
		diags.Append(valueDiags...)
		if diags.HasError() {
			return types.ListNull(endpointObjectType)
		}
		values = append(values, value)
	}

	list, listDiags := types.ListValue(endpointObjectType, values)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ListNull(endpointObjectType)
	}
	return list
}
