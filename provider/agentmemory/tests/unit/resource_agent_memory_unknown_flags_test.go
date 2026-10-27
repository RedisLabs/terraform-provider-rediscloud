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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

var enabledFlagCases = []struct{ name, block, wire, attribute string }{
	{"summarization", "summarization", "summarization", "summarization.0.enabled"},
	{"exclusions", "", "longTermMemoryExclusions", "long_term_memory_exclusions.0.enabled"},
	{"semantic", "semantic", "semantic", "long_term_memory_exclusions.0.semantic.0.enabled"},
	{"built-in", "built_in_detectors", "builtInDetectors", "long_term_memory_exclusions.0.built_in_detectors.0.enabled"},
	{"custom", "custom_detectors", "customDetectors", "long_term_memory_exclusions.0.custom_detectors.0.enabled"},
}

func enabledFlagConfig(name, block, expression, dependency string) string {
	attribute := ""
	if expression != "" {
		attribute = "enabled=" + expression
	}
	var setting string
	switch name {
	case "summarization":
		setting = "summarization {\n" + attribute + "\n}"
	case "exclusions":
		setting = "long_term_memory_exclusions {\n" + attribute + "\n}"
	default:
		setting = "long_term_memory_exclusions {\nenabled=true\n" + block + " {\n" + attribute + "\n}\n}"
	}
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="unknown-flags-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 %s
}
`, setting)
}

func TestAgentMemoryEnabledFlagValidation(t *testing.T) {
	for _, flag := range enabledFlagCases {
		for _, test := range []struct {
			name, expression, dependency string
			invalid                      bool
		}{
			{name: "true", expression: "true"}, {name: "false", expression: "false"},
			{name: "unknown", expression: "terraform_data.flag.output", dependency: `resource "terraform_data" "flag" { input=true }`},
			{name: "omitted", invalid: true}, {name: "null", expression: "null", invalid: true},
		} {
			t.Run(flag.name+"/"+test.name, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					http.Error(w, "plan must not call HTTP", 500)
				}))
				defer server.Close()
				defer func() {
					if requests.Load() != 0 {
						t.Error("validation made HTTP requests")
					}
				}()
				prefix := testProviderConfig(t, server.URL)
				step := resource.TestStep{Config: prefix + enabledFlagConfig(flag.name, flag.block, test.expression, test.dependency), PlanOnly: true, ExpectNonEmptyPlan: true}
				if test.invalid {
					step.ExpectError = regexp.MustCompile("Invalid Attribute Combination")
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
			})
		}
	}
}

func TestAgentMemoryEnabledFlagUnknownResolvesNull(t *testing.T) {
	for _, flag := range enabledFlagCases {
		t.Run(flag.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.Error(w, "null flag must not call HTTP", 500)
			}))
			defer server.Close()
			defer func() {
				if requests.Load() != 0 {
					t.Error("resolved null flag reached HTTP")
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			// A null input is known during plan; the conditional defers null until apply.
			config := prefix + enabledFlagConfig(flag.name, flag.block, `terraform_data.flag.output ? null : false`, `resource "terraform_data" "flag" { input=true }`)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config, ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
			}})
		})
	}
}

func TestAgentMemoryEnabledFlagsWithoutBlocks(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "plan must not call HTTP", 500)
	}))
	defer server.Close()
	defer func() {
		if requests.Load() != 0 {
			t.Error("validation made HTTP requests")
		}
	}()
	prefix := testProviderConfig(t, server.URL)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: prefix + `resource "rediscloud_agent_memory" "example" {
name="no-flags-store"
database_id=123
}`, PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
}

func TestAgentMemoryMissingEnabledWithUnknownThreshold(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "invalid flag must not call HTTP", 500)
	}))
	defer server.Close()
	defer func() {
		if requests.Load() != 0 {
			t.Error("validation made HTTP requests")
		}
	}()
	prefix := testProviderConfig(t, server.URL)
	config := prefix + `resource "terraform_data" "threshold" { input=20 }
resource "rediscloud_agent_memory" "example" {
 name="independent-flags-store"
 database_id=123
 summarization {
  enabled=true
  event_count {
   threshold=terraform_data.threshold.output
   retain_count=2
  }
 }
 long_term_memory_exclusions {}
}`
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
	}})
}

func TestAgentMemoryEnabledFlagLifecycle(t *testing.T) {
	for _, flag := range enabledFlagCases {
		t.Run(flag.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			var creates, updates, deletes int
			var observed []bool
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
					var policy map[string]any
					if flag.name == "summarization" {
						policy, _ = payload["summarization"].(map[string]any)
					} else {
						policy, _ = payload["longTermMemoryExclusions"].(map[string]any)
						if flag.name != "exclusions" {
							policy, _ = policy[flag.wire].(map[string]any)
						}
					}
					value, ok := policy["enabled"].(bool)
					if !ok {
						t.Error("resolved flag missing from request")
						w.WriteHeader(400)
						return
					}
					observed = append(observed, value)
					if r.Method == http.MethodPost {
						creates++
						store = payload
					} else {
						updates++
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
				if store != nil {
					t.Error("local fixture not removed")
				}
				if creates != 1 || updates != 1 || deletes != 1 {
					t.Errorf("expected one local create/update/delete; got %d/%d/%d", creates, updates, deletes)
				}
				if len(observed) != 2 || !observed[0] || observed[1] {
					t.Error("expected resolved true then false")
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			config := func(expr, dependency string) string {
				return prefix + enabledFlagConfig(flag.name, flag.block, expr, dependency)
			}
			on := config("terraform_data.flag.output", `resource "terraform_data" "flag" { input=true }`)
			off := config("terraform_data.flag.output", `resource "terraform_data" "flag" { input=false }`)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: on, Check: resource.TestCheckResourceAttr("rediscloud_agent_memory.example", flag.attribute, "true")},
				// Optional+Computed may reuse a prior value; configuration must still require the flag.
				{Config: config("", `resource "terraform_data" "flag" { input=true }`), PlanOnly: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
				{Config: off, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", flag.attribute, "false"))},
				{Config: off, PlanOnly: true},
			}})
		})
	}
}
