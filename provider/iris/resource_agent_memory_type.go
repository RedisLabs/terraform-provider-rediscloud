package iris

import (
	"strings"
	"sync"

	"context"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

type agentMemoryTypeCreateRequest struct {
	Description        *string           `json:"description,omitempty"`
	ExtractionStrategy *memoryExtraction `json:"extractionStrategy,omitempty"`
	Fields             []memoryField     `json:"fields,omitempty"`
	Name               *string           `json:"name,omitempty"`
}
type agentMemoryTypeUpdateRequest struct {
	ExtractionStrategy *memoryExtraction `json:"extractionStrategy,omitempty"`
	ExpectedRevision   string            `json:"expectedRevision"`
}
type agentMemoryTypeResponse struct {
	Description        *string           `json:"description"`
	ExtractionStrategy *memoryExtraction `json:"extractionStrategy"`
	Fields             []memoryField     `json:"fields"`
	Name               *string           `json:"name"`
	Revision           *string           `json:"revision"`
}
type agentMemoryTypeResource struct{ client *client.IrisServiceClient }

var (
	_ resource.Resource                = (*agentMemoryTypeResource)(nil)
	_ resource.ResourceWithConfigure   = (*agentMemoryTypeResource)(nil)
	_ resource.ResourceWithImportState = (*agentMemoryTypeResource)(nil)
)

func NewAgentMemoryTypeResource() resource.Resource { return &agentMemoryTypeResource{} }
func (r *agentMemoryTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iris_agent_memory_type"
}
func (r *agentMemoryTypeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = AgentMemoryTypeResourceSchema(ctx)
}
func (r *agentMemoryTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, "agent-memory", &resp.Diagnostics)
}
func (r *agentMemoryTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config AgentMemoryTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Create, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	fields, d := memoryFieldsToAPI(ctx, config.Fields)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := agentMemoryTypeCreateRequest{
		Description:        optionalString(config.Description),
		ExtractionStrategy: extractionToAPI(config.ExtractionPrompt, config.ExtractionEnabled),
		Fields:             fields,
		Name:               optionalString(config.Name),
	}
	unlock, err := lockMemoryTypes(ctx, data.StoreID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to lock memory types", err.Error())
		return
	}
	defer unlock()
	var created agentMemoryTypeResponse
	if err := r.client.Do(ctx, http.MethodPost, "/iris/v1/agent-memory/stores/"+escaped(data.StoreID)+"/memory-types", nil, body, &created); err != nil {
		resp.Diagnostics.AddError("Failed to create Iris resource", err.Error())
		return
	}
	if created.Name == nil || *created.Name == "" {
		resp.Diagnostics.AddError("Missing Iris resource identifier", "The resource may have been created, but the response omitted its identifier. Inspect the service before retrying.")
		return
	}
	resp.Diagnostics.Append(data.setFromAPI(ctx, &created)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A failed create would taint the type, and replacing it permanently retires its name.
	resp.Diagnostics.Append(r.refresh(ctx, &resp.State, &data, "created", true)...)
}
func (r *agentMemoryTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentMemoryTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Read, defaultReadTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.read(ctx, &data)
	if client.IrisNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Iris resource", err.Error())
		return
	}
	resp.Diagnostics.Append(data.setFromAPI(ctx, current)...)
	resp.Diagnostics.Append(saveState(ctx, &resp.State, &data)...)
}
func (r *agentMemoryTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AgentMemoryTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var previous AgentMemoryTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Update, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Description.Equal(previous.Description) || !data.Fields.Equal(previous.Fields) {
		resp.Diagnostics.AddError("Immutable memory type", "Description and fields cannot be updated. Use a new type name rather than deleting and recreating a retired name.")
		return
	}
	if data.Revision.IsUnknown() {
		data.Revision = previous.Revision
	}
	body := agentMemoryTypeUpdateRequest{
		ExtractionStrategy: &memoryExtraction{Prompt: changedString(data.ExtractionPrompt, previous.ExtractionPrompt), Enabled: changedBool(data.ExtractionEnabled, previous.ExtractionEnabled)},
		ExpectedRevision:   previous.Revision.ValueString(),
	}
	if body.ExtractionStrategy.Prompt == nil && body.ExtractionStrategy.Enabled == nil {
		previous.Timeouts = data.Timeouts
		resp.Diagnostics.Append(saveState(ctx, &resp.State, &previous)...)
		return
	}
	unlock, err := lockMemoryTypes(ctx, data.StoreID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to lock memory types", err.Error())
		return
	}
	defer unlock()
	if err := r.client.Do(ctx, "PATCH", "/iris/v1/agent-memory/stores/"+escaped(data.StoreID)+"/memory-types/"+escaped(data.Name), nil, body, nil); err != nil {
		resp.Diagnostics.AddError("Failed to update Iris resource", err.Error())
		return
	}
	resp.Diagnostics.Append(r.refresh(ctx, &resp.State, &data, "updated", false)...)
}
func (r *agentMemoryTypeResource) refresh(ctx context.Context, state *tfsdk.State, data *AgentMemoryTypeModel, action string, warnOnly bool) diag.Diagnostics {
	return readAfterWrite(ctx, state, data, action, warnOnly,
		func() (*agentMemoryTypeResponse, error) { return r.read(ctx, data) },
		func(current *agentMemoryTypeResponse) diag.Diagnostics { return data.setFromAPI(ctx, current) })
}
func (r *agentMemoryTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentMemoryTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Delete, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	unlock, err := lockMemoryTypes(ctx, data.StoreID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to lock memory types", err.Error())
		return
	}
	defer unlock()
	current, err := r.read(ctx, &data)
	if client.IrisNotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read memory type before deletion", err.Error())
		return
	}
	if current.Revision == nil || *current.Revision == "" {
		resp.Diagnostics.AddError("Missing Iris revision", "The service omitted the current memory-type revision; no deletion was sent.")
		return
	}
	if memoryTypeExtractionEnabled(current) {
		disabled := false
		body := agentMemoryTypeUpdateRequest{ExpectedRevision: *current.Revision, ExtractionStrategy: &memoryExtraction{Enabled: &disabled}}
		var updated agentMemoryTypeResponse
		err = r.client.Do(ctx, http.MethodPatch, "/iris/v1/agent-memory/stores/"+escaped(data.StoreID)+"/memory-types/"+escaped(data.Name), nil, body, &updated)
		if client.IrisNotFound(err) {
			return
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to disable memory-type extraction before deletion", err.Error())
			return
		}
		// Disabling increments the revision. Read it back rather than guessing a token.
		current, err = r.read(ctx, &data)
		if client.IrisNotFound(err) {
			return
		}
		if err != nil {
			resp.Diagnostics.AddError("Failed to read disabled memory type", err.Error())
			return
		}
		if memoryTypeExtractionEnabled(current) {
			resp.Diagnostics.AddError("Memory-type extraction is still enabled", "The service did not confirm that extraction was disabled; no deletion was sent.")
			return
		}
	}
	if current.Revision == nil || *current.Revision == "" {
		resp.Diagnostics.AddError("Missing Iris revision", "The service omitted the current memory-type revision; no deletion was sent.")
		return
	}
	query := url.Values{"expectedRevision": {*current.Revision}}
	if err := r.client.Do(ctx, http.MethodDelete, "/iris/v1/agent-memory/stores/"+escaped(data.StoreID)+"/memory-types/"+escaped(data.Name), query, nil, nil); err != nil && !client.IrisNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Iris resource", err.Error())
	}

}
func (r *agentMemoryTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := importParts(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Iris import identifier", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("store_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
func (r *agentMemoryTypeResource) read(ctx context.Context, data *AgentMemoryTypeModel) (*agentMemoryTypeResponse, error) {
	var current agentMemoryTypeResponse
	if err := r.client.Do(ctx, http.MethodGet, "/iris/v1/agent-memory/stores/"+escaped(data.StoreID)+"/memory-types/"+escaped(data.Name), nil, nil, &current); err != nil {
		return nil, err
	}
	if err := verifyID(data.Name, current.Name); err != nil {
		return nil, err
	}
	return &current, nil
}
func (m *AgentMemoryTypeModel) setFromAPI(ctx context.Context, current *agentMemoryTypeResponse) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	m.Description = types.StringPointerValue(current.Description)
	if current.ExtractionStrategy != nil {
		m.ExtractionPrompt = types.StringPointerValue(current.ExtractionStrategy.Prompt)
		m.ExtractionEnabled = types.BoolPointerValue(current.ExtractionStrategy.Enabled)
	} else {
		m.ExtractionPrompt = types.StringNull()
		m.ExtractionEnabled = types.BoolNull()
	}
	v, d := memoryFieldsFromAPI(ctx, current.Fields)
	diagnostics.Append(d...)
	m.Fields = v
	if current.Name != nil {
		m.Name = types.StringPointerValue(current.Name)
	}
	m.Revision = types.StringPointerValue(current.Revision)
	m.ID = types.StringValue(m.StoreID.ValueString() + "/" + m.Name.ValueString())
	return diagnostics
}

func memoryTypeExtractionEnabled(current *agentMemoryTypeResponse) bool {
	strategy := current.ExtractionStrategy
	return strategy != nil && (strategy.Enabled == nil || *strategy.Enabled)
}

var _ resource.ResourceWithModifyPlan = (*agentMemoryTypeResource)(nil)

func (r *agentMemoryTypeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	if req.State.Raw.IsNull() {
		var config AgentMemoryTypeModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		if !resp.Diagnostics.HasError() && !config.ExtractionEnabled.IsNull() && !config.ExtractionEnabled.IsUnknown() && config.ExtractionPrompt.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("extraction_prompt"), "Missing extraction prompt", "Configure extraction_prompt when creating a type with extraction_enabled. Existing strategies support enable-only updates.")
		}
		return
	}
	var previous, planned AgentMemoryTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	if resp.Diagnostics.HasError() || planned.Name.IsUnknown() || planned.StoreID.IsUnknown() {
		return
	}
	// A new store is a new registry. A genuinely new name can also be created.
	if !planned.StoreID.Equal(previous.StoreID) || !strings.EqualFold(planned.Name.ValueString(), previous.Name.ValueString()) {
		return
	}
	detail := "Iris permanently retires removed memory-type names within a store. Deleting and recreating this name would fail. Use a distinct, unused type name or a different store."
	if previous.ExtractionPrompt.IsNull() && ((!planned.ExtractionPrompt.IsNull() && !planned.ExtractionPrompt.IsUnknown()) || (!planned.ExtractionEnabled.IsNull() && !planned.ExtractionEnabled.IsUnknown())) {
		resp.Diagnostics.AddAttributeError(path.Root("extraction_prompt"), "Missing extraction strategy", "Iris cannot add extraction to an existing type that was created without a strategy. Create a type with a distinct, unused name and configure extraction at creation.")
	}
	if !planned.Name.Equal(previous.Name) {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Immutable memory type", detail)
	}
	if !planned.Description.IsUnknown() && !planned.Description.Equal(previous.Description) {
		resp.Diagnostics.AddAttributeError(path.Root("description"), "Immutable memory type", detail)
	}
	fields, err := planned.Fields.ToTerraformValue(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid memory fields", err.Error())
		return
	}
	if fields.IsFullyKnown() && !planned.Fields.Equal(previous.Fields) {
		resp.Diagnostics.AddAttributeError(path.Root("field"), "Immutable memory type", detail)
	}
}

// The service updates one shared registry for every type in a store, so a
// capacity-one channel per store serialises each complete operation (including
// disable/read/delete) and lets a queued waiter give up when its deadline passes.
// A provider process is short-lived, so the map is deliberately never pruned.
var memoryTypeLocks sync.Map

func lockMemoryTypes(ctx context.Context, storeID string) (func(), error) {
	gate, _ := memoryTypeLocks.LoadOrStore(storeID, make(chan struct{}, 1))
	lock := gate.(chan struct{})
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
