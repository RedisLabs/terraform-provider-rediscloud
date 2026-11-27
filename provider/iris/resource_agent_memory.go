package iris

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

type agentMemoryCreateRequest struct {
	Database           memoryDatabase     `json:"database"`
	CustomMemoryTypes  []memoryCustomType `json:"customMemoryTypes,omitempty"`
	ExtractionStrategy *string            `json:"extractionStrategy,omitempty"`
	agentMemorySettings
}
type agentMemorySettings struct {
	ExtractionCadence        *memoryCadence        `json:"extractionCadence,omitempty"`
	LongTermMemory           *longTermMemoryConfig `json:"longTermMemory,omitempty"`
	LongTermMemoryExclusions *memoryExclusions     `json:"longTermMemoryExclusions,omitempty"`
	Name                     *string               `json:"name,omitempty"`
	Ownership                *memoryOwnership      `json:"ownership,omitempty"`
	ShortMemory              *memoryTTL            `json:"shortMemory,omitempty"`
	Summarization            *memorySummarization  `json:"summarization,omitempty"`
}
type agentMemoryResponse struct {
	CreatedAt          *string          `json:"createdAt"`
	Database           *memoryDatabase  `json:"database"`
	Endpoints          []memoryEndpoint `json:"endpoints"`
	ExtractionStrategy *string          `json:"extractionStrategy"`
	Failure            *memoryFailure   `json:"failure"`
	State              *string          `json:"state"`
	StoreID            *string          `json:"storeId"`
	SubscriptionID     *string          `json:"subscriptionId"`
	agentMemorySettings
}
type agentMemoryResource struct{ client *client.IrisServiceClient }

var (
	_ resource.Resource                = (*agentMemoryResource)(nil)
	_ resource.ResourceWithConfigure   = (*agentMemoryResource)(nil)
	_ resource.ResourceWithImportState = (*agentMemoryResource)(nil)
)

func NewAgentMemoryResource() resource.Resource { return &agentMemoryResource{} }
func (r *agentMemoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iris_agent_memory"
}
func (r *agentMemoryResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = AgentMemoryResourceSchema(ctx)
}
func (r *agentMemoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredClient(req, "agent-memory", &resp.Diagnostics)
}
func (r *agentMemoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config AgentMemoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Create, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	data.BuiltInDetectors = config.BuiltInDetectors
	data.CustomDetectors = config.CustomDetectors
	settings, d := data.apiPatch(ctx, config, AgentMemoryModel{})
	resp.Diagnostics.Append(d...)
	customTypes, d := memoryTypesToAPI(ctx, config.CustomMemoryTypes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := agentMemoryCreateRequest{Database: memoryDatabase{DatabaseID: config.DatabaseID.ValueString()}, CustomMemoryTypes: customTypes, ExtractionStrategy: optionalString(config.ExtractionStrategy), agentMemorySettings: settings}
	var created agentMemoryResponse
	if err := r.client.Do(ctx, http.MethodPost, "/iris/v1/agent-memory/stores", nil, body, &created); err != nil {
		resp.Diagnostics.AddError("Failed to create Iris resource", err.Error())
		return
	}
	if created.StoreID == nil || *created.StoreID == "" {
		resp.Diagnostics.AddError("Missing Iris resource identifier", "The resource may have been created, but the response omitted its identifier. Inspect the service before retrying.")
		return
	}
	data.ID = types.StringPointerValue(created.StoreID)
	data.State = types.StringPointerValue(created.State)
	resp.Diagnostics.Append(r.refresh(ctx, &resp.State, &data, "created")...)
}
func (r *agentMemoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AgentMemoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Read, defaultReadTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	current, err := r.read(ctx, &data, false)
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
func (r *agentMemoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, config AgentMemoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	var previous AgentMemoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Update, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	data.BuiltInDetectors = config.BuiltInDetectors
	data.CustomDetectors = config.CustomDetectors
	body, d := data.apiPatch(ctx, config, previous)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if body == (agentMemorySettings{}) {
		// Local preferences and releasing management of a secret require no API write.
		previous.FlushOnDelete = data.FlushOnDelete
		previous.Timeouts = data.Timeouts
		previous.LLMAPIKey = types.StringNull()
		previous.LLMAPIKeyVersion = data.LLMAPIKeyVersion
		previous.EmbeddingAPIKey = types.StringNull()
		previous.EmbeddingAPIKeyVersion = data.EmbeddingAPIKeyVersion
		previous.ClassificationAPIKey = types.StringNull()
		previous.ClassificationAPIKeyVersion = data.ClassificationAPIKeyVersion
		resp.Diagnostics.Append(saveState(ctx, &resp.State, &previous)...)
		return
	}
	if err := r.client.Do(ctx, "PATCH", "/iris/v1/agent-memory/stores/"+escaped(data.ID), nil, body, nil); err != nil {
		resp.Diagnostics.AddError("Failed to update Iris resource", err.Error())
		return
	}
	resp.Diagnostics.Append(r.refresh(ctx, &resp.State, &data, "updated")...)
}
func (r *agentMemoryResource) refresh(ctx context.Context, state *tfsdk.State, data *AgentMemoryModel, action string) diag.Diagnostics {
	return readAfterWrite(ctx, state, data, action, false,
		func() (*agentMemoryResponse, error) { return r.read(ctx, data, true) },
		func(current *agentMemoryResponse) diag.Diagnostics { return data.setFromAPI(ctx, current) })
}

func (r *agentMemoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AgentMemoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	ctx, cancel := operationContext(ctx, data.Timeouts.Delete, defaultOperationTimeout, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	query := url.Values{}
	query.Set("flush", fmt.Sprint(data.FlushOnDelete.ValueBool()))
	if err := r.client.Do(ctx, http.MethodDelete, "/iris/v1/agent-memory/stores/"+escaped(data.ID), query, nil, nil); err != nil && !client.IrisNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Iris resource", err.Error())
	}
}
func (r *agentMemoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := importParts(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Iris import identifier", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("flush_on_delete"), false)...)
}
func (r *agentMemoryResource) read(ctx context.Context, data *AgentMemoryModel, wait bool) (*agentMemoryResponse, error) {
	fetch := func(ctx context.Context) (*agentMemoryResponse, error) {
		var current agentMemoryResponse
		if err := r.client.Do(ctx, http.MethodGet, "/iris/v1/agent-memory/stores/"+escaped(data.ID), nil, nil, &current); err != nil {
			return nil, err
		}
		if err := verifyID(data.ID, current.StoreID); err != nil {
			return nil, err
		}
		return &current, nil
	}
	if !wait {
		return fetch(ctx)
	}
	var last *agentMemoryResponse
	var mu sync.Mutex
	duration := defaultOperationTimeout
	if deadline, ok := ctx.Deadline(); ok {
		duration = time.Until(deadline)
	}
	const pollInterval = 5 * time.Second
	waiter := retry.StateChangeConf{Pending: []string{"PROVISIONING"}, Target: []string{"READY"}, Timeout: duration, PollInterval: pollInterval, NotFoundChecks: int(duration/pollInterval) + 1, Refresh: func() (any, string, error) {
		attempt, cancel := context.WithTimeout(ctx, time.Minute)
		current, err := fetch(attempt)
		cancel()
		if err != nil {
			if ctx.Err() == nil && retryableStoreRead(err) {
				var response *client.IrisHTTPError
				if errors.As(err, &response) && response.RetryAfter > 0 {
					timer := time.NewTimer(response.RetryAfter)
					defer timer.Stop()
					select {
					case <-ctx.Done():
						return nil, "", ctx.Err()
					case <-timer.C:
					}
				}
				return nil, "PROVISIONING", nil
			}
			return nil, "", err
		}
		mu.Lock()
		last = current
		mu.Unlock()
		state := ""
		if current.State != nil {
			state = *current.State
		}
		return current, state, nil
	}}
	_, err := waiter.WaitForStateContext(ctx)
	mu.Lock()
	defer mu.Unlock()
	return last, err
}

// Retry only reads during readiness; mutation errors remain visible to the caller.
func retryableStoreRead(err error) bool {
	var response *client.IrisHTTPError
	if errors.As(err, &response) {
		switch response.StatusCode {
		case 404, 408, 429, 500, 502, 503, 504, 520, 521, 522, 523, 524:
			return true
		}
		return false
	}
	// A body cut off or a connection closed mid-response says nothing about the
	// store, and the read is repeatable. Malformed JSON is deliberately not
	// listed: repeating the same read cannot fix it.
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	var network net.Error
	return errors.As(err, &network) && network.Timeout()
}
