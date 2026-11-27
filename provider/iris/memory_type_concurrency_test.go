package iris

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestQueuedMemoryTypeWritesRespectDeadline(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			var requests atomic.Int64
			r := testResource(t, "iris_agent_memory_type", func(_ http.ResponseWriter, _ *http.Request) { requests.Add(1) })
			unlock, err := lockMemoryTypes(context.Background(), "queued-store")
			require.NoError(t, err)
			defer unlock()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			config, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("profile"), "store_id": types.StringValue("queued-store"), "description": types.StringValue("Profile"), "extraction_prompt": types.StringValue("Extract"), "extraction_enabled": types.BoolValue(true)})
			var previous AgentMemoryTypeModel
			require.False(t, config.Get(context.Background(), &previous).HasError())

			previous.ID = types.StringValue("queued-store/profile")
			previous.Revision = types.StringValue("1")
			previous.ExtractionPrompt = types.StringValue("Before")
			state := modelState(t, r, &previous)
			done := make(chan diag.Diagnostics, 1)
			go func() {
				switch operation {
				case "create":
					resp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
					r.Create(ctx, resource.CreateRequest{Config: config, Plan: plan}, &resp)
					done <- resp.Diagnostics
				case "update":
					resp := resource.UpdateResponse{State: state}
					r.Update(ctx, resource.UpdateRequest{Config: config, Plan: plan, State: state}, &resp)
					done <- resp.Diagnostics
				case "delete":
					var resp resource.DeleteResponse
					r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)
					done <- resp.Diagnostics
				}
			}()
			select {
			case d := <-done:
				require.True(t, d.HasError(), d)
				require.Equal(t, "Failed to lock memory types", d.Errors()[0].Summary())
			case <-time.After(time.Second):
				t.Fatal("operation kept waiting for a lock after its deadline")
			}
			require.Zero(t, requests.Load(), "a canceled waiter must not issue any HTTP request")
		})
	}
}

func TestMemoryTypeLocksAreIndependentAndCancelable(t *testing.T) {
	unlock, err := lockMemoryTypes(context.Background(), "locked-store")
	require.NoError(t, err)
	other, err := lockMemoryTypes(context.Background(), "other-store")
	require.NoError(t, err, "different stores must not block one another")
	other()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, err = lockMemoryTypes(ctx, "locked-store")
	require.ErrorIs(t, err, context.DeadlineExceeded, "a queued waiter must give up at its deadline")
	unlock()
	again, err := lockMemoryTypes(context.Background(), "locked-store")
	require.NoError(t, err, "a canceled waiter must not keep the store locked")
	again()
}

func TestSiblingMemoryTypeWritesAreSerialized(t *testing.T) {
	ctx := context.Background()
	var registryRevision atomic.Int64
	var mu sync.Mutex
	records := map[string]map[string]any{}
	r := testResource(t, "iris_agent_memory_type", func(w http.ResponseWriter, req *http.Request) {
		name := req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]
		if req.Method != http.MethodGet {
			// Model the service's read/modify/CAS registry transaction, not an
			// HTTP-wide lock that would conceal concurrent sibling writes.
			version := registryRevision.Load()
			timer := time.NewTimer(20 * time.Millisecond)
			defer timer.Stop()
			select {
			case <-req.Context().Done():
				return
			case <-timer.C:
			}
			if !registryRevision.CompareAndSwap(version, version+1) {
				http.Error(w, "registry revision conflict", http.StatusPreconditionFailed)
				return
			}
		}
		mu.Lock()
		defer mu.Unlock()
		switch req.Method {
		case http.MethodPost:
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			name = body["name"].(string)
			body["revision"] = "1"
			records[name] = body
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			if body["expectedRevision"] != records[name]["revision"] {
				http.Error(w, "type revision conflict", http.StatusPreconditionFailed)
				return
			}
			revision, err := strconv.Atoi(records[name]["revision"].(string))
			require.NoError(t, err)
			strategy := records[name]["extractionStrategy"].(map[string]any)
			for key, value := range body["extractionStrategy"].(map[string]any) {
				strategy[key] = value
			}
			records[name]["revision"] = strconv.Itoa(revision + 1)
		case http.MethodDelete:
			if req.URL.Query().Get("expectedRevision") != records[name]["revision"] {
				http.Error(w, "type revision conflict", http.StatusPreconditionFailed)
				return
			}
			delete(records, name)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(records[name]))
	})
	parallel := func(fn func(int)) {
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := range 3 {
			wg.Add(1)
			go func() { defer wg.Done(); <-start; fn(i) }()
		}
		close(start)
		wg.Wait()
	}
	states := make([]tfsdk.State, 3)
	fields, d := memoryFieldsFromAPI(ctx, []memoryField{{Name: "topic", Type: "STRING", Description: "Topic"}})
	require.False(t, d.HasError())
	parallel(func(i int) {
		config, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue(fmt.Sprintf("profile_%d", i)), "store_id": types.StringValue("sibling-store"), "description": types.StringValue("Profile"), "field": fields, "extraction_prompt": types.StringValue("Extract"), "extraction_enabled": types.BoolValue(true)})
		response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
		r.Create(ctx, resource.CreateRequest{Config: config, Plan: plan}, &response)
		require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
		states[i] = response.State
	})
	if t.Failed() {
		return
	}
	parallel(func(i int) {
		var previous AgentMemoryTypeModel
		require.False(t, states[i].Get(ctx, &previous).HasError())
		config, _ := objects(t, r, map[string]attr.Value{"name": previous.Name, "store_id": previous.StoreID, "description": previous.Description, "field": fields, "extraction_prompt": types.StringValue("Updated"), "extraction_enabled": types.BoolValue(true)})
		planned := previous
		planned.ExtractionPrompt = types.StringValue("Updated")
		planned.Revision = types.StringUnknown()
		response := resource.UpdateResponse{State: states[i]}
		r.Update(ctx, resource.UpdateRequest{Config: config, Plan: tfsdk.Plan(modelState(t, r, &planned)), State: states[i]}, &response)
		require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
		states[i] = response.State
	})
	if t.Failed() {
		return
	}
	parallel(func(i int) {
		response := resource.DeleteResponse{}
		r.Delete(ctx, resource.DeleteRequest{State: states[i]}, &response)
		require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
	})
	require.Empty(t, records)
}
