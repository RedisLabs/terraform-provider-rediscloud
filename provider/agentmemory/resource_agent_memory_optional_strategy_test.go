package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func optionalStrategyConfig(strategy, dependency string) string {
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="optional-strategy-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 custom_memory_types {
  name="preference"
  description="User preferences"
  %s
 }
}
`, strategy)
}

func TestAgentMemoryOptionalStrategyValidation(t *testing.T) {
	for _, test := range []struct{ name, strategy, dependency, errorPattern string }{
		{name: "absent"},
		{name: "empty-block", strategy: `extraction_strategy {}`, errorPattern: `(?s)(Invalid Attribute Combination|Missing Configuration for Required Attribute|Missing required argument)`},
		{name: "null-prompt", strategy: `extraction_strategy { prompt=null }`, errorPattern: `(?s)(Invalid Attribute Combination|Missing Configuration for Required Attribute|Missing required argument)`},
		{name: "empty-prompt", strategy: `extraction_strategy { prompt="" }`, errorPattern: `length`},
		{name: "oversized-prompt", strategy: fmt.Sprintf(`extraction_strategy { prompt=%q }`, strings.Repeat("x", 10001)), errorPattern: `length`},
		{name: "prompt", strategy: `extraction_strategy { prompt="Extract preferences" }`},
		{name: "maximum-prompt", strategy: fmt.Sprintf(`extraction_strategy { prompt=%q }`, strings.Repeat("x", 10000))},
		{name: "enabled-true", strategy: "extraction_strategy {\nprompt=\"Extract preferences\"\nenabled=true\n}"},
		{name: "enabled-false", strategy: "extraction_strategy {\nprompt=\"Extract preferences\"\nenabled=false\n}"},
		{name: "enabled-only", strategy: `extraction_strategy { enabled=false }`, errorPattern: `(?s)(Invalid Attribute Combination|Missing Configuration for Required Attribute|Missing required argument)`},
		{name: "unknown-prompt", strategy: `extraction_strategy { prompt=terraform_data.prompt.output }`, dependency: `resource "terraform_data" "prompt" { input="Extract preferences" }`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "validation must not call HTTP", 500)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("validation reached HTTP")
				}
			}()
			config := testProviderConfig(t, server.URL) + optionalStrategyConfig(test.strategy, test.dependency)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if test.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(test.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryOptionalStrategyUnknownResolvesNull(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "null prompt must not call HTTP", 500)
	}))
	defer server.Close()
	defer func() {
		if calls.Load() != 0 {
			t.Error("resolved null prompt reached HTTP")
		}
	}()
	// A conditional keeps null unknown during plan, then resolves it before create.
	config := testProviderConfig(t, server.URL) + optionalStrategyConfig(`extraction_strategy { prompt=terraform_data.flag.output ? null : "Extract preferences" }`, `resource "terraform_data" "flag" { input=true }`)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
		{Config: config, ExpectError: regexp.MustCompile(`(?s)(Invalid Attribute Combination|Missing Configuration for Required Attribute|Missing required argument)`)},
	}})
}

func TestAgentMemoryOptionalStrategyLifecycle(t *testing.T) {
	for _, test := range []struct{ name, strategy, dependency string }{
		{name: "absent"},
		{name: "default-enabled", strategy: `extraction_strategy { prompt="Extract preferences" }`},
		{name: "disabled", strategy: "extraction_strategy {\nprompt=\"Extract preferences\"\nenabled=false\n}"},
		{name: "resolved-prompt", strategy: `extraction_strategy { prompt=terraform_data.prompt.output }`, dependency: `resource "terraform_data" "prompt" { input="Extract preferences" }`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, deletes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					if err := json.NewDecoder(r.Body).Decode(&store); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					items, ok := store["customMemoryTypes"].([]any)
					if !ok || len(items) != 1 {
						t.Error("expected one custom memory type")
						w.WriteHeader(400)
						return
					}
					memoryType, ok := items[0].(map[string]any)
					if !ok {
						t.Error("invalid custom memory type")
						w.WriteHeader(400)
						return
					}
					strategy, present := memoryType["extractionStrategy"]
					if test.name == "absent" {
						if present {
							t.Error("absent strategy must be omitted from request")
						}
					} else {
						policy, ok := strategy.(map[string]any)
						if !ok || policy["prompt"] != "Extract preferences" {
							t.Error("resolved prompt missing")
							w.WriteHeader(400)
							return
						}
						if test.name == "disabled" {
							if policy["enabled"] != false {
								t.Error("explicit false not preserved")
							}
						} else {
							if _, ok := policy["enabled"]; ok {
								t.Error("omitted enabled must use API default")
							}
							// Simulate the documented API default in the read response.
							policy["enabled"] = true
						}
					}
					creates++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
					if store == nil {
						http.NotFound(w, r)
						return
					}
					store["storeId"] = "store-1"
					store["databaseId"] = "123"
					store["status"] = "READY"
					store["extractionStrategy"] = "INSTRUCT"
					store["createdAt"] = "2026-10-16T12:00:00Z"
					store["endpoint"] = "https://memory.example.test"
					store["endpoints"] = []map[string]any{{"url": "https://memory.example.test", "provider": "AWS", "region": "us-east-1", "egressIps": []string{}, "isAccessible": true}}
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
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				if store != nil || creates != 1 || deletes != 1 {
					t.Errorf("fixture cleanup/counts failed: create=%d delete=%d remaining=%t", creates, deletes, store != nil)
				}
			}()
			config := testProviderConfig(t, server.URL) + optionalStrategyConfig(test.strategy, test.dependency)
			checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")}
			if test.name != "absent" {
				checks = append(checks, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.extraction_strategy.prompt", "Extract preferences"))
				enabled := "true"
				if test.name == "disabled" {
					enabled = "false"
				}
				checks = append(checks, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.extraction_strategy.enabled", enabled))
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, Check: resource.ComposeTestCheckFunc(checks...)},
				{Config: config, PlanOnly: true},
			}})
		})
	}
}
