package unit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/agentmemory"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

type readinessFixture struct {
	api                              *testhelpers.FakeAPI
	store                            *api.Store
	statuses                         []string
	reads, creates, updates, deletes int
	failure                          int
	failureAfter                     int
	failureCount                     int
	postDelay                        time.Duration
	deadline                         time.Time
	changedDeadline                  bool
	keyExists                        bool
	keyBeforeReady                   bool
	readySeen                        bool
}

func newReadinessFixture(t *testing.T, statuses ...string) *readinessFixture {
	t.Helper()
	f := &readinessFixture{api: testhelpers.NewFakeAPI(), statuses: statuses}
	f.store = &api.Store{
		StoreID:            redis.String("store-1"),
		Name:               redis.String("store"),
		DatabaseID:         redis.String("123"),
		Status:             redis.String(api.StatusReady),
		Endpoint:           redis.String("https://memory.example.invalid"),
		ShortMemory:        &api.ShortMemoryConfig{TTLSeconds: redis.Int(86400)},
		LongTermMemory:     &api.LongTermMemoryStatus{TTLSeconds: redis.Int(31536000)},
		ExtractionStrategy: redis.String("INSTRUCT"),
		ExtractionCadence:  &api.ExtractionCadenceConfig{ActiveIntervalSeconds: redis.Int(300)},
	}
	f.api.Handle("POST /memory-stores", func(req *http.Request) (any, error) {
		var request api.CreateStore
		if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
			return nil, err
		}
		if request.Name != nil {
			f.store.Name = request.Name
		}
		f.creates++
		f.reads = 0
		f.readySeen = false
		f.deadline, _ = req.Context().Deadline()
		if f.postDelay > 0 {
			select {
			case <-time.After(f.postDelay):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}
		return map[string]any{"taskId": "create-task"}, nil
	})
	f.api.Handle("PATCH /memory-stores/store-1", func(req *http.Request) (any, error) {
		var update api.UpdateStore
		if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
			return nil, err
		}
		if update.Name != nil {
			f.store.Name = update.Name
		}
		f.updates++
		f.reads = 0
		f.readySeen = false
		f.deadline, _ = req.Context().Deadline()
		return map[string]any{"taskId": "update-task"}, nil
	})
	f.api.Handle("GET /memory-stores/store-1", func(req *http.Request) (any, error) {
		if f.store == nil {
			return nil, testhelpers.NotFound()
		}
		deadline, _ := req.Context().Deadline()
		if !f.deadline.IsZero() && !deadline.Equal(f.deadline) {
			f.changedDeadline = true
		}
		f.reads++
		if f.failure != 0 && f.reads > f.failureAfter && (f.failureCount == 0 || f.reads <= f.failureAfter+f.failureCount) {
			return nil, testhelpers.StatusError{Code: f.failure, Body: map[string]any{"description": "fixture read failure"}}
		}
		status := f.statuses[min(f.reads-1, len(f.statuses)-1)]
		f.readySeen = f.readySeen || status == api.StatusReady
		store := *f.store
		store.Status = redis.String(status)
		return store, nil
	})
	f.api.Handle("DELETE /memory-stores/store-1", func(*http.Request) (any, error) {
		f.deletes++
		f.store = nil
		return map[string]any{"taskId": "delete-task"}, nil
	})
	for _, name := range []string{"create-task", "update-task", "delete-task"} {
		f.api.Handle("GET /tasks/"+name, func(*http.Request) (any, error) {
			value := map[string]any{"taskId": name, "status": "processing-completed"}
			if name == "create-task" {
				value["response"] = map[string]any{"resource": map[string]any{"storeId": "store-1"}}
			}
			return value, nil
		})
	}
	return f
}

func readinessResource(t *testing.T, f *readinessFixture, duration string) (resource.Resource, tfsdk.Plan, tfsdk.State) {
	t.Helper()
	ctx := t.Context()
	r := agentmemory.NewAgentMemoryResource()
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	require.False(t, schema.Diagnostics.HasError(), schema.Diagnostics)
	r.(resource.ResourceWithConfigure).Configure(ctx, resource.ConfigureRequest{ProviderData: testhelpers.NewAPIClient(t, f.api)}, &resource.ConfigureResponse{})
	model := agentmemory.AgentMemoryResourceModel{
		ID:                       types.StringValue("store-1"),
		Name:                     types.StringValue("store"),
		DatabaseID:               types.Int64Value(123),
		ShortTermTTLSeconds:      types.Int64Value(86400),
		LongTermTTLSeconds:       types.Int64Value(31536000),
		ExtractionStrategy:       types.StringValue("INSTRUCT"),
		ExtractionCadenceSeconds: types.Int64Value(300),
		Status:                   types.StringValue(api.StatusReady),
		Endpoint:                 types.StringValue("https://memory.example.invalid"),
		Endpoints:                types.ListNull(schema.Schema.Attributes["endpoints"].GetType().(attr.TypeWithElementType).ElementType()),
	}
	typ := schema.Schema.Blocks["timeouts"].Type().(agentmemory.MemoryTimeoutsListType).ElemType.(timeouts.Type)
	values := map[string]attr.Value{}
	for name := range typ.AttrTypes {
		values[name] = types.StringNull()
	}
	values["create"], values["update"] = types.StringValue(duration), types.StringValue(duration)
	timeoutList, diags := agentmemory.NewMemoryTimeoutsList(ctx, []timeouts.Value{{Object: types.ObjectValueMust(typ.AttrTypes, values)}})
	require.False(t, diags.HasError(), diags)
	model.Timeouts = timeoutList
	plan, state := tfsdk.Plan{Schema: schema.Schema}, tfsdk.State{Schema: schema.Schema}
	require.False(t, state.Set(ctx, &model).HasError())
	model.Name = types.StringValue("updated")
	require.False(t, plan.Set(ctx, &model).HasError())
	return r, plan, state
}

func TestAgentMemoryWaitsForReadyAfterTask(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			f := newReadinessFixture(t, api.StatusProvisioning, api.StatusProvisioning, api.StatusReady)
			r, plan, state := readinessResource(t, f, "500ms")
			var result tfsdk.State
			if operation == "create" {
				resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
				r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &resp)
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
				result = resp.State
			} else {
				resp := resource.UpdateResponse{State: state}
				r.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
				result = resp.State
			}
			var model agentmemory.AgentMemoryResourceModel
			require.False(t, result.Get(t.Context(), &model).HasError())
			assert.Equal(t, api.StatusReady, model.Status.ValueString())
			f.api.WithHandlersPaused(func() {
				assert.Equal(t, 3, f.reads)
				assert.False(t, f.changedDeadline)
				assert.Equal(t, 1, f.creates+f.updates)
			})
		})
	}
}

func TestAgentMemoryReadinessFailurePreservesIdentity(t *testing.T) {
	scenarios := []struct {
		name         string
		status       string
		failure      int
		failureAfter int
	}{
		{name: "timeout", status: api.StatusProvisioning},
		{name: "unavailable", status: api.StatusUnavailable},
		{name: "not-found", status: api.StatusProvisioning, failure: http.StatusNotFound},
		{name: "forbidden", status: api.StatusProvisioning, failure: http.StatusForbidden},
		{name: "read-error", status: api.StatusProvisioning, failure: http.StatusInternalServerError},
		{name: "read-error-after-provisioning", status: api.StatusProvisioning, failure: http.StatusInternalServerError, failureAfter: 1},
	}
	for _, operation := range []string{"create", "update"} {
		for _, scenario := range scenarios {
			t.Run(operation+"/"+scenario.name, func(t *testing.T) {
				f := newReadinessFixture(t, scenario.status)
				f.failure, f.failureAfter = scenario.failure, scenario.failureAfter
				f.postDelay = 10 * time.Millisecond
				r, plan, state := readinessResource(t, f, "60ms")
				var result tfsdk.State
				if operation == "create" {
					resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
					r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &resp)
					require.True(t, resp.Diagnostics.HasError(), resp.Diagnostics)
					result = resp.State
				} else {
					resp := resource.UpdateResponse{State: state}
					r.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
					require.True(t, resp.Diagnostics.HasError(), resp.Diagnostics)
					result = resp.State
				}
				require.False(t, result.Raw.IsNull())
				require.True(t, result.Raw.IsFullyKnown())
				var model agentmemory.AgentMemoryResourceModel
				require.False(t, result.Get(context.Background(), &model).HasError())
				assert.Equal(t, "store-1", model.ID.ValueString())
				wantStatus := scenario.status
				if scenario.failure != 0 && scenario.failureAfter == 0 && operation == "update" {
					wantStatus = api.StatusReady
				}
				assert.Equal(t, wantStatus, model.Status.ValueString())
				f.api.WithHandlersPaused(func() {
					assert.False(t, f.changedDeadline)
					assert.Equal(t, 1, f.creates+f.updates)
					assert.Zero(t, f.deletes)
					if scenario.failure != 0 && scenario.failure != http.StatusNotFound {
						assert.Equal(t, scenario.failureAfter+1, f.reads)
					}
					if scenario.failure == http.StatusNotFound {
						assert.Greater(t, f.reads, 1)
					}
				})
			})
		}
	}
}

func TestAgentMemoryReadinessRetriesPostTaskNotFound(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			f := newReadinessFixture(t, api.StatusProvisioning, api.StatusProvisioning, api.StatusProvisioning, api.StatusReady)
			f.failure, f.failureCount = http.StatusNotFound, 2
			r, plan, state := readinessResource(t, f, "500ms")
			var result tfsdk.State
			if operation == "create" {
				resp := resource.CreateResponse{State: tfsdk.State{Schema: state.Schema}}
				r.Create(t.Context(), resource.CreateRequest{Plan: plan}, &resp)
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
				result = resp.State
			} else {
				resp := resource.UpdateResponse{State: state}
				r.Update(t.Context(), resource.UpdateRequest{Plan: plan, State: state}, &resp)
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
				result = resp.State
			}
			var model agentmemory.AgentMemoryResourceModel
			require.False(t, result.Get(t.Context(), &model).HasError())
			assert.Equal(t, "store-1", model.ID.ValueString())
			assert.Equal(t, api.StatusReady, model.Status.ValueString())
			f.api.WithHandlersPaused(func() {
				assert.Equal(t, 4, f.reads)
				assert.False(t, f.changedDeadline)
				assert.Equal(t, 1, f.creates+f.updates)
				assert.Zero(t, f.deletes)
			})
		})
	}
}

func (f *readinessFixture) registerDependentKey() {
	f.api.Handle("POST /memory-stores/store-1/api-keys", func(*http.Request) (any, error) {
		if !f.readySeen {
			f.keyBeforeReady = true
			return nil, testhelpers.StatusError{Code: http.StatusConflict, Body: map[string]any{"description": "store is not ready"}}
		}
		f.keyExists = true
		return map[string]any{"apiKey": "FAKE-READINESS-SECRET"}, nil
	})
	f.api.Handle("GET /memory-stores/store-1/api-keys", func(*http.Request) (any, error) {
		keys := []map[string]any{}
		if f.keyExists {
			keys = append(keys, map[string]any{"apiKeyId": "key-1", "keyName": "runtime", "createdAt": 123, "obfuscatedToken": "masked"})
		}
		return map[string]any{"apiKeys": keys}, nil
	})
	f.api.Handle("DELETE /memory-stores/store-1/api-keys/key-1", func(*http.Request) (any, error) {
		f.keyExists = false
		return map[string]any{}, nil
	})
}

func TestAgentMemoryReadinessTerraformLifecycle(t *testing.T) {
	f := newReadinessFixture(t, api.StatusProvisioning, api.StatusProvisioning, api.StatusReady)
	f.failure, f.failureCount = http.StatusNotFound, 1
	f.registerDependentKey()
	testingresource.UnitTest(t, testingresource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(testhelpers.NewAPIClient(t, f.api), agentmemory.NewAgentMemoryResource, agentmemory.NewAgentMemoryAPIKeyResource),
		Steps: []testingresource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/readiness.tf"),
				Check: testingresource.ComposeTestCheckFunc(
					testingresource.TestCheckResourceAttr("test_agent_memory.example", "status", api.StatusReady),
					testingresource.TestCheckResourceAttr("test_agent_memory_api_key.dependent", "id", "key-1"),
				),
			},
			{
				ConfigFile:      config.StaticFile("testdata/readiness.tf"),
				ConfigVariables: config.Variables{"name": config.StringVariable("updated")},
				Check:           testingresource.TestCheckResourceAttr("test_agent_memory.example", "status", api.StatusReady),
			},
			{
				ConfigFile:      config.StaticFile("testdata/readiness.tf"),
				ConfigVariables: config.Variables{"name": config.StringVariable("updated")},
				PlanOnly:        true,
			},
		},
	})
	f.api.WithHandlersPaused(func() {
		assert.False(t, f.keyBeforeReady)
		assert.Equal(t, 1, f.creates)
		assert.Equal(t, 1, f.updates)
		assert.Equal(t, 1, f.deletes)
		assert.False(t, f.keyExists)
		assert.Nil(t, f.store)
	})
}

func TestAgentMemoryReadinessCreateErrorKeepsTaintedIdentity(t *testing.T) {
	f := newReadinessFixture(t, api.StatusUnavailable)
	f.registerDependentKey()
	testingresource.UnitTest(t, testingresource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(testhelpers.NewAPIClient(t, f.api), agentmemory.NewAgentMemoryResource, agentmemory.NewAgentMemoryAPIKeyResource),
		Steps: []testingresource.TestStep{
			{
				ConfigFile:  config.StaticFile("testdata/readiness.tf"),
				ExpectError: regexp.MustCompile("Agent Memory service failed to become ready"),
			},
			{
				ConfigFile: config.StaticFile("testdata/readiness.tf"),
				PreConfig: func() {
					f.api.WithHandlersPaused(func() {
						f.statuses = []string{api.StatusReady}
						f.reads = 0
					})
				},
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: testingresource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("test_agent_memory.example", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
	f.api.WithHandlersPaused(func() {
		assert.False(t, f.keyBeforeReady)
		assert.Equal(t, 1, f.creates)
		assert.Equal(t, 1, f.deletes)
		assert.Nil(t, f.store)
	})
}
