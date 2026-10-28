package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func requiredEventCountConfig(block, dependency string) string {
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="event-count-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 summarization {
  enabled=true
  %s
 }
}
`, block)
}

func TestAgentMemoryEventCountRequiredFields(t *testing.T) {
	const missing = "Invalid Attribute Combination"
	const invalid = "Invalid summarization event count"
	for _, c := range []struct{ name, block, dependency, errorPattern string }{
		{name: "absent-block"},
		{name: "empty-block", block: `event_count {}`, errorPattern: missing},
		{name: "threshold-only", block: `event_count { threshold=20 }`, errorPattern: missing},
		{name: "retain-only", block: `event_count { retain_count=10 }`, errorPattern: missing},
		{name: "null-threshold", block: "event_count {\nthreshold=null\nretain_count=10\n}", errorPattern: missing},
		{name: "null-retain", block: "event_count {\nthreshold=20\nretain_count=null\n}", errorPattern: missing},
		{name: "both-null", block: "event_count {\nthreshold=null\nretain_count=null\n}", errorPattern: missing},
		{name: "valid-values", block: "event_count {\nthreshold=20\nretain_count=10\n}"},
		{name: "retain-zero", block: "event_count {\nthreshold=1\nretain_count=0\n}"},
		{name: "maximum-threshold", block: "event_count {\nthreshold=10000\nretain_count=9999\n}"},
		{name: "equal-values", block: "event_count {\nthreshold=10\nretain_count=10\n}", errorPattern: invalid},
		{name: "reversed-values", block: "event_count {\nthreshold=10\nretain_count=20\n}", errorPattern: invalid},
		{name: "negative-value-deferred-to-api", block: "event_count {\nthreshold=20\nretain_count=-1\n}"},
		{name: "above-former-limit-deferred-to-api", block: "event_count {\nthreshold=10001\nretain_count=10\n}"},
		{name: "unknown-threshold", block: "event_count {\nthreshold=terraform_data.value.output\nretain_count=10\n}", dependency: `resource "terraform_data" "value" { input=20 }`},
		{name: "unknown-retain", block: "event_count {\nthreshold=20\nretain_count=terraform_data.value.output\n}", dependency: `resource "terraform_data" "value" { input=10 }`},
		{name: "both-unknown", block: "event_count {\nthreshold=terraform_data.values.output.threshold\nretain_count=terraform_data.values.output.retain\n}", dependency: `resource "terraform_data" "values" { input={threshold=20,retain=10} }`},
		{name: "unknown-threshold-missing-retain", block: `event_count { threshold=terraform_data.value.output }`, dependency: `resource "terraform_data" "value" { input=20 }`, errorPattern: missing},
		{name: "unknown-retain-missing-threshold", block: `event_count { retain_count=terraform_data.value.output }`, dependency: `resource "terraform_data" "value" { input=10 }`, errorPattern: missing},
	} {
		t.Run(c.name, func(t *testing.T) {
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
			config := testProviderConfig(t, server.URL) + requiredEventCountConfig(c.block, c.dependency)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(c.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryEventCountRequiredLifecycle(t *testing.T) {
	for _, c := range []struct {
		name, errorPattern string
		threshold, retain  config.Variable
	}{
		{
			name:      "valid-update",
			threshold: config.IntegerVariable(30),
			retain:    config.IntegerVariable(0),
		},
		{
			name:         "resolved-invalid-comparison",
			threshold:    config.IntegerVariable(5),
			retain:       config.IntegerVariable(10),
			errorPattern: "Invalid summarization event count",
		},
		{
			name:         "resolved-null-threshold",
			threshold:    nil,
			retain:       config.IntegerVariable(10),
			errorPattern: "Invalid Attribute Combination",
		},
		{
			name:         "resolved-null-retain",
			threshold:    config.IntegerVariable(20),
			retain:       nil,
			errorPattern: "Invalid Attribute Combination",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores" || r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					policy, ok := payload["summarization"].(map[string]any)
					if !ok {
						t.Error("missing summarization")
						w.WriteHeader(400)
						return
					}
					counts, ok := policy["eventCount"].(map[string]any)
					if !ok {
						t.Error("missing eventCount")
						w.WriteHeader(400)
						return
					}
					threshold, hasThreshold := counts["threshold"].(float64)
					retain, hasRetain := counts["retainCount"].(float64)
					if !hasThreshold || !hasRetain || threshold <= retain {
						t.Errorf("invalid values reached HTTP: %v", counts)
						w.WriteHeader(400)
						return
					}
					if r.Method == http.MethodPost {
						creates++
						store = payload
						if threshold != 20 || retain != 10 {
							t.Error("unexpected initial values")
						}
					} else {
						patches++
						if threshold != 30 || retain != 0 {
							t.Error("resolved update values not preserved")
						}
						for k, v := range payload {
							store[k] = v
						}
					}
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
				expectedPatches := 0
				if c.errorPattern == "" {
					expectedPatches = 1
				}
				if store != nil || creates != 1 || deletes != 1 || patches != expectedPatches {
					t.Errorf("unexpected fixture counts: creates=%d patches=%d deletes=%d remaining=%t", creates, patches, deletes, store != nil)
				}
			}()
			testProviderConfig(t, server.URL)
			steps := []resource.TestStep{{
				ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_deferred_event_counts.tf"),
				ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
					"threshold": config.IntegerVariable(20),
					"retain":    config.IntegerVariable(10),
				})},
			}, {
				ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_deferred_event_counts.tf"),
				ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
					"threshold": c.threshold,
					"retain":    c.retain,
				})},
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			}}
			if c.errorPattern != "" {
				steps = append(steps, resource.TestStep{
					ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_deferred_event_counts.tf"),
					ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
						"threshold": c.threshold,
						"retain":    c.retain,
					})},
					ExpectError: regexp.MustCompile(c.errorPattern),
				})
			} else {
				steps = append(steps, resource.TestStep{
					ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_deferred_event_counts.tf"),
					ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
						"threshold": c.threshold,
						"retain":    c.retain,
					})},
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
						resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.event_count.0.threshold", "30"),
						resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.event_count.0.retain_count", "0"),
					),
				}, resource.TestStep{
					ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_deferred_event_counts.tf"),
					ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
						"threshold": c.threshold,
						"retain":    c.retain,
					})},
					PlanOnly: true,
				})
				// Prior state must not satisfy fields omitted from the new configuration.
				for _, fixture := range []string{"testdata/lifecycle/summarization/summarization_event_counts_missing_retain.tf", "testdata/lifecycle/summarization/summarization_event_counts_missing_threshold.tf"} {
					steps = append(steps, resource.TestStep{
						ConfigFile: config.StaticFile(fixture),
						ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
							"threshold": c.threshold,
							"retain":    c.retain,
						})},
						PlanOnly:    true,
						ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
					})
				}
				// Restore valid configuration so the harness can plan fixture destruction.
				steps = append(steps, resource.TestStep{
					ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_deferred_event_counts.tf"),
					ConfigVariables: config.Variables{"values": config.ObjectVariable(map[string]config.Variable{
						"threshold": c.threshold,
						"retain":    c.retain,
					})},
					PlanOnly: true,
				})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
		})
	}
}
