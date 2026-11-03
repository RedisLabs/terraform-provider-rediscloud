package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemorySummarizationStrategyLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, fixture, expected string
		strategy, source        config.Variable
		deferred                bool
	}{
		{
			name:     "omitted",
			expected: "FAKE-API-DEFAULT",
			fixture:  "testdata/lifecycle/summarization/summarization_strategy_omitted.tf",
		},
		{
			name:     "null",
			expected: "FAKE-API-DEFAULT",
			fixture:  "testdata/lifecycle/summarization/summarization_strategy.tf",
		},
		{
			name:     "current",
			expected: "event_count",
			fixture:  "testdata/lifecycle/summarization/summarization_strategy.tf",
			strategy: config.StringVariable("event_count"),
		},
		{
			name:     "future",
			expected: "FAKE-FUTURE",
			fixture:  "testdata/lifecycle/summarization/summarization_strategy.tf",
			strategy: config.StringVariable("FAKE-FUTURE"),
		},
		{
			name:     "unknown",
			expected: "FAKE-RESOLVED",
			fixture:  "testdata/lifecycle/summarization/summarization_strategy_deferred.tf",
			source:   config.StringVariable("FAKE-RESOLVED"),
			deferred: true,
		},
		{
			name:     "unknown null",
			expected: "FAKE-API-DEFAULT",
			fixture:  "testdata/lifecycle/summarization/summarization_strategy_deferred.tf",
			deferred: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			creates, patches, deletes := 0, 0, 0
			var store map[string]any
			var firstStrategy any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					summarization, _ := payload["summarization"].(map[string]any)
					if r.Method == http.MethodPost {
						creates++
						firstStrategy = summarization["triggerStrategy"]
					} else {
						patches++
					}
					if summarization["triggerStrategy"] == "FAKE-REJECTED" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"unsupported summarization strategy"}`))
						return
					}
					if store == nil {
						response := httptest.NewRecorder()
						writeStoreResponse(t, response, "store", 86400, 31536000, 300)
						if err := json.Unmarshal(response.Body.Bytes(), &store); err != nil {
							t.Error(err)
							w.WriteHeader(500)
							return
						}
					}
					if summarization["enabled"] == true && summarization["triggerStrategy"] == nil {
						summarization["triggerStrategy"] = "FAKE-API-DEFAULT"
					}
					if summarization["enabled"] == false {
						delete(summarization, "triggerStrategy")
					}
					store["summarization"] = summarization
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && store != nil:
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			testProviderConfig(t, server.URL)
			updatedFixture := "testdata/lifecycle/summarization/summarization_strategy.tf"
			omittedFixture := "testdata/lifecycle/summarization/summarization_strategy_omitted.tf"
			if tc.deferred {
				updatedFixture = "testdata/lifecycle/summarization/summarization_strategy_with_source.tf"
				omittedFixture = "testdata/lifecycle/summarization/summarization_strategy_omitted_with_source.tf"
			}
			check := func(strategy string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 {
						return fmt.Errorf("store replaced: creates=%d deletes=%d", creates, deletes)
					}
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.trigger_strategy", strategy))(state)
				}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile: config.StaticFile(tc.fixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": tc.strategy,
						"source":   tc.source,
					})},
					Check: check(tc.expected),
				},
				{
					ConfigFile: config.StaticFile(tc.fixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": tc.strategy,
						"source":   tc.source,
					})},
					PlanOnly: true,
				},
				{
					ConfigFile: config.StaticFile(updatedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": config.StringVariable("FAKE-FUTURE-UPDATE"),
						"source":   tc.source,
					})},
					Check: check("FAKE-FUTURE-UPDATE"),
				},
				{
					ConfigFile: config.StaticFile(updatedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": config.StringVariable("FAKE-REJECTED"),
						"source":   tc.source,
					})},
					ExpectError: regexp.MustCompile("Failed to update Agent Memory service"),
				},
				{
					ConfigFile: config.StaticFile(updatedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": config.StringVariable("FAKE-FUTURE-UPDATE"),
						"source":   tc.source,
					})},
					PlanOnly: true,
				},
				{
					ConfigFile: config.StaticFile(omittedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": nil,
						"source":   tc.source,
					})},
					PlanOnly: true,
				},
				{
					ConfigFile: config.StaticFile(omittedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(false),
						"strategy": nil,
						"source":   tc.source,
					})},
					Check: resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "summarization.0.trigger_strategy"),
				},
				{
					ConfigFile: config.StaticFile(omittedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": nil,
						"source":   tc.source,
					})},
					Check: check("FAKE-API-DEFAULT"),
				},
				{
					ConfigFile: config.StaticFile(omittedFixture),
					ConfigVariables: config.Variables{"settings": config.ObjectVariable(map[string]config.Variable{
						"enabled":  config.BoolVariable(true),
						"strategy": nil,
						"source":   tc.source,
					})},
					PlanOnly: true,
				},
			}})
			mu.Lock()
			defer mu.Unlock()
			if tc.expected == "FAKE-API-DEFAULT" {
				assert.Nil(t, firstStrategy, "provider must not inject a default into the request")
			} else {
				assert.Equal(t, tc.expected, firstStrategy)
			}
			assert.Equal(t, 1, creates)
			assert.Equal(t, 4, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemorySummarizationStrategyErrors(t *testing.T) {
	for _, tc := range []struct {
		name, expression, dependency, pattern string
		calls                                 int
	}{
		{name: "empty", expression: `""`, pattern: `unsupported\s+summarization\s+strategy`, calls: 1},
		{name: "resolved empty", expression: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="" }`, pattern: `unsupported\s+summarization\s+strategy`, calls: 1},
		{name: "API rejection", expression: `"FAKE-REJECTED"`, pattern: "Failed to create Agent Memory service", calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/memory-stores" {
					t.Error("unexpected API operation")
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				summarization, _ := payload["summarization"].(map[string]any)
				expected := ""
				if tc.name == "API rejection" {
					expected = "FAKE-REJECTED"
				}
				assert.Equal(t, expected, summarization["triggerStrategy"])
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"message":"unsupported summarization strategy"}`))
			}))
			defer server.Close()
			config := testProviderConfig(t, server.URL) + requiredEventCountConfig("trigger_strategy="+tc.expression, tc.dependency)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile(tc.pattern)}}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.calls, calls)
		})
	}
}
