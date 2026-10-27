package agentmemory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func (r *agentMemoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentMemoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	operationTimeout, diags := plan.Timeouts.Create(ctx, agentMemoryMutationDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	llm := modelConfigForRequest(ctx, req.Config, "llm", modelConfigBlock(ctx, plan.LLM, &resp.Diagnostics), &resp.Diagnostics)
	embedding := modelConfigForRequest(ctx, req.Config, "embedding", modelConfigBlock(ctx, plan.Embedding, &resp.Diagnostics), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	longTermMemory := longTermMemoryFromPlan(ctx, plan, &resp.Diagnostics)
	if longTermMemory != nil {
		longTermMemory.Embedding = embedding
	}
	request := agentmemoryapi.CreateStore{
		Name:                     redis.String(plan.Name.ValueString()),
		DatabaseID:               redis.Int(int(plan.DatabaseID.ValueInt64())),
		ShortMemory:              shortMemoryFromPlan(plan),
		LongTermMemory:           longTermMemory,
		LLM:                      llm,
		ExtractionStrategy:       stringPointerFromValue(plan.ExtractionStrategy),
		ExtractionCadence:        extractionCadenceFromPlan(plan),
		Summarization:            summarizationFromModel(ctx, summarizationBlock(ctx, plan.Summarization, &resp.Diagnostics), &resp.Diagnostics),
		LongTermMemoryExclusions: exclusionsFromModel(ctx, exclusionsBlock(ctx, plan.LongTermMemoryExclusions, &resp.Diagnostics), &resp.Diagnostics),
		CustomMemoryTypes:        customMemoryTypesFromModel(ctx, plan.CustomMemoryTypes, &resp.Diagnostics),
	}
	if resp.Diagnostics.HasError() {
		return
	}
	storeID, err := r.client.Client.AgentMemory.Create(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Agent Memory service", err.Error())
		return
	}

	plan.ID = types.StringValue(storeID)
	plan.Status = types.StringValue(agentmemoryapi.StatusProvisioning)
	plan.Endpoint = types.StringNull()
	plan.CreatedAt = types.StringNull()

	store, readErr := waitForAgentMemoryToBeReady(ctx, storeID, r.client, operationTimeout)
	if store != nil && store.Status != nil {
		plan.Status = types.StringPointerValue(store.Status)
	}
	needsRecovery := readErr != nil
	if readErr != nil {
		resp.Diagnostics.AddError(
			"Agent Memory service failed to become ready",
			"The Agent Memory service was created and its ID was saved, but readiness could not be confirmed: "+readErr.Error(),
		)
	} else {
		candidate := plan
		var readDiags diag.Diagnostics
		readAgentMemoryIntoModel(ctx, store, &candidate, false, &readDiags)
		needsRecovery = readDiags.HasError()
		if needsRecovery {
			// An error after acknowledged creation taints the store and can trigger deletion on retry.
			// Keep the original known inputs and surface only this hydration failure as warnings.
			for _, diagnostic := range readDiags {
				resp.Diagnostics.AddWarning(diagnostic.Summary(),
					"The Agent Memory service was created and its ID was saved. Metadata was not adopted because the follow-up read was invalid. Refresh or investigate the existing store before retrying. "+diagnostic.Detail())
			}
		} else {
			plan = candidate
			resp.Diagnostics.Append(readDiags...)
		}
	}

	// Always retain the acknowledged identity rather than an invalid read response.
	plan.ID = types.StringValue(storeID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if needsRecovery || resp.Diagnostics.HasError() {
		// Recovery state must preserve known values and replace unresolved computed values with null.
		recovered, err := tftypes.Transform(resp.State.Raw, func(_ *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
			if !value.IsKnown() {
				return tftypes.NewValue(value.Type(), nil), nil
			}
			return value, nil
		})
		if err != nil {
			resp.Diagnostics.AddError("Failed to preserve Agent Memory recovery state", err.Error())
			return
		}
		resp.State.Raw = recovered
	}
}

func (r *agentMemoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentMemoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	operationTimeout, diags := state.Timeouts.Read(ctx, agentMemoryReadDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

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

	operationTimeout, diags := plan.Timeouts.Update(ctx, agentMemoryMutationDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

	update := agentmemoryapi.UpdateStore{}
	if !plan.Name.Equal(state.Name) {
		value := plan.Name.ValueString()
		update.Name = &value
	}
	if !plan.ShortTermTTLSeconds.Equal(state.ShortTermTTLSeconds) {
		update.ShortMemory = shortMemoryFromPlan(plan)
	}
	if !plan.LongTermTTLSeconds.Equal(state.LongTermTTLSeconds) {
		update.LongTermMemory = longTermMemoryFromPlan(ctx, plan, &resp.Diagnostics)
	}
	if !modelConfigEqual(ctx, plan.Embedding, state.Embedding, &resp.Diagnostics) {
		update.LongTermMemory = longTermMemoryFromPlan(ctx, plan, &resp.Diagnostics)
		if update.LongTermMemory != nil {
			update.LongTermMemory.Embedding = modelConfigForRequest(ctx, req.Config, "embedding", modelConfigBlock(ctx, plan.Embedding, &resp.Diagnostics), &resp.Diagnostics)
		}
	} else if update.LongTermMemory != nil {
		// A TTL-only update must not resend an unchanged write-only credential.
		update.LongTermMemory.Embedding = nil
	}
	if !modelConfigEqual(ctx, plan.LLM, state.LLM, &resp.Diagnostics) {
		update.LLM = modelConfigForRequest(ctx, req.Config, "llm", modelConfigBlock(ctx, plan.LLM, &resp.Diagnostics), &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.ExtractionCadenceSeconds.Equal(state.ExtractionCadenceSeconds) {
		update.ExtractionCadence = extractionCadenceFromPlan(plan)
	}
	if len(plan.Summarization.Elements()) > 0 {
		update.Summarization = summarizationFromModel(ctx, summarizationBlock(ctx, plan.Summarization, &resp.Diagnostics), &resp.Diagnostics)
	}
	if len(plan.LongTermMemoryExclusions.Elements()) > 0 {
		update.LongTermMemoryExclusions = exclusionsFromModel(ctx, exclusionsBlock(ctx, plan.LongTermMemoryExclusions, &resp.Diagnostics), &resp.Diagnostics)
	}
	update.AddCustomMemoryTypes, update.UpdateCustomMemoryTypeStrategies = customMemoryTypeUpdatesFromPlan(ctx, plan.CustomMemoryTypes, state.CustomMemoryTypes, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}
	storeID := state.ID.ValueString()
	if err := r.client.Client.AgentMemory.Update(ctx, storeID, update); err != nil {
		resp.Diagnostics.AddError("Failed to update Agent Memory service", err.Error())
		return
	}

	store, err := waitForAgentMemoryToBeReady(ctx, storeID, r.client, operationTimeout)
	if err != nil {
		if store != nil && store.Status != nil {
			state.Status = types.StringPointerValue(store.Status)
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		resp.Diagnostics.AddError("Agent Memory service failed to become ready after update", err.Error())
		return
	}
	// Validate the response against the requested ID even when the plan ID is unknown.
	plan.ID = state.ID
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

	operationTimeout, diags := state.Timeouts.Delete(ctx, agentMemoryMutationDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()

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
	return &agentmemoryapi.ShortMemoryConfig{TTLSeconds: redis.Int(int(plan.ShortTermTTLSeconds.ValueInt64()))}
}

func longTermMemoryFromPlan(ctx context.Context, plan AgentMemoryResourceModel, diags *diag.Diagnostics) *agentmemoryapi.LongTermMemoryConfig {
	if (plan.LongTermTTLSeconds.IsNull() || plan.LongTermTTLSeconds.IsUnknown()) && len(plan.Embedding.Elements()) == 0 {
		return nil
	}
	config := &agentmemoryapi.LongTermMemoryConfig{
		Embedding: modelConfigFromModel(ctx, modelConfigBlock(ctx, plan.Embedding, diags), diags),
	}
	if !plan.LongTermTTLSeconds.IsNull() && !plan.LongTermTTLSeconds.IsUnknown() {
		ttl := int(plan.LongTermTTLSeconds.ValueInt64())
		config.TTLSeconds = &ttl
	}
	return config
}

func extractionCadenceFromPlan(plan AgentMemoryResourceModel) *agentmemoryapi.ExtractionCadenceConfig {
	if plan.ExtractionCadenceSeconds.IsNull() || plan.ExtractionCadenceSeconds.IsUnknown() {
		return nil
	}
	return &agentmemoryapi.ExtractionCadenceConfig{ActiveIntervalSeconds: redis.Int(int(plan.ExtractionCadenceSeconds.ValueInt64()))}
}

func readAgentMemoryIntoModel(ctx context.Context, store *agentmemoryapi.Store, state *AgentMemoryResourceModel, includeAPIOnlyConfig bool, diags *diag.Diagnostics) {
	// Reject invalid identity before copying any response fields into the owned model.
	if store == nil || redis.StringValue(store.StoreID) == "" {
		diags.AddError("Invalid Agent Memory response", "The API response omitted the store ID.")
		return
	}
	if !state.ID.IsNull() && !state.ID.IsUnknown() && state.ID.ValueString() != "" && state.ID.ValueString() != redis.StringValue(store.StoreID) {
		diags.AddError("Invalid Agent Memory response", "The API response did not match the requested store ID.")
		return
	}
	if redis.StringValue(store.Name) == "" {
		diags.AddError("Unexpected Agent Memory name", "The API response omitted the store name. The tracked store was preserved.")
		return
	}
	state.ID = types.StringPointerValue(store.StoreID)
	state.Name = types.StringPointerValue(store.Name)
	state.Status = types.StringPointerValue(store.Status)

	databaseID, err := strconv.ParseInt(redis.StringValue(store.DatabaseID), 10, 64)
	if err != nil {
		diags.AddError("Unexpected Agent Memory database ID", fmt.Sprintf("Expected databaseId to be numeric, got %q.", redis.StringValue(store.DatabaseID)))
		return
	}
	state.DatabaseID = types.Int64Value(databaseID)

	if store.ShortMemory != nil {
		state.ShortTermTTLSeconds = intPointerValue(store.ShortMemory.TTLSeconds)
	} else {
		state.ShortTermTTLSeconds = types.Int64Null()
	}
	if store.LongTermMemory != nil {
		state.LongTermTTLSeconds = intPointerValue(store.LongTermMemory.TTLSeconds)
	} else {
		state.LongTermTTLSeconds = types.Int64Null()
	}
	state.ExtractionStrategy = types.StringPointerValue(store.ExtractionStrategy)
	state.LLM = modelConfigList(ctx, modelConfigToModel(ctx, store.LLM, modelConfigBlock(ctx, state.LLM, diags), diags), diags)
	var embedding *agentmemoryapi.ModelStatus
	if store.LongTermMemory != nil {
		embedding = store.LongTermMemory.Embedding
	}
	state.Embedding = modelConfigList(ctx, modelConfigToModel(ctx, embedding, modelConfigBlock(ctx, state.Embedding, diags), diags), diags)
	if store.ExtractionCadence != nil {
		state.ExtractionCadenceSeconds = intPointerValue(store.ExtractionCadence.ActiveIntervalSeconds)
	} else {
		state.ExtractionCadenceSeconds = types.Int64Null()
	}
	// Terraform cannot add optional nested blocks during create/update refresh when
	// they were absent from config. Import is the one path where we intentionally
	// adopt advanced API config even when the previous state is empty.
	if includeAPIOnlyConfig || len(state.Summarization.Elements()) > 0 || store.Summarization == nil {
		state.Summarization = summarizationList(ctx, summarizationToModel(ctx, store.Summarization, summarizationBlock(ctx, state.Summarization, diags), includeAPIOnlyConfig, diags), diags)
	}
	if includeAPIOnlyConfig || len(state.LongTermMemoryExclusions.Elements()) > 0 || store.LongTermMemoryExclusions == nil {
		state.LongTermMemoryExclusions = exclusionsList(ctx, exclusionsToModel(ctx, store.LongTermMemoryExclusions, exclusionsBlock(ctx, state.LongTermMemoryExclusions, diags), includeAPIOnlyConfig, diags), diags)
	}
	if includeAPIOnlyConfig || !state.CustomMemoryTypes.IsNull() || len(store.CustomMemoryTypes) == 0 {
		existing := customMemoryTypeModels(ctx, state.CustomMemoryTypes, diags)
		models := customMemoryTypesToModel(ctx, store.CustomMemoryTypes, existing, diags)
		list, listDiags := NewCustomMemoryTypeList(ctx, models)
		diags.Append(listDiags...)
		state.CustomMemoryTypes = list
	}

	switch {
	case redis.StringValue(store.Endpoint) != "":
		state.Endpoint = types.StringPointerValue(store.Endpoint)
	case len(store.Endpoints) > 0:
		state.Endpoint = types.StringPointerValue(store.Endpoints[0].URL)
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
		if redis.StringValue(sortedEndpoints[i].Provider) != redis.StringValue(sortedEndpoints[j].Provider) {
			return redis.StringValue(sortedEndpoints[i].Provider) < redis.StringValue(sortedEndpoints[j].Provider)
		}
		if redis.StringValue(sortedEndpoints[i].Region) != redis.StringValue(sortedEndpoints[j].Region) {
			return redis.StringValue(sortedEndpoints[i].Region) < redis.StringValue(sortedEndpoints[j].Region)
		}
		return redis.StringValue(sortedEndpoints[i].URL) < redis.StringValue(sortedEndpoints[j].URL)
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
			"url":           types.StringPointerValue(endpoint.URL),
			"provider":      types.StringPointerValue(endpoint.Provider),
			"region":        types.StringPointerValue(endpoint.Region),
			"egress_ips":    egressIPs,
			"is_accessible": types.BoolPointerValue(endpoint.IsAccessible),
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
