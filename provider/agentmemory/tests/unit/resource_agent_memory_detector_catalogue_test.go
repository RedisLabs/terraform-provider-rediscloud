package unit_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func catalogueConfig(field, expression, dependency string) string {
	name := `"ticket"`
	if field == "name" {
		name = expression
	}
	config := reservedDetectorConfig(name, dependency, "catalogue-placeholder")
	if field == "id" {
		config = strings.Replace(config, `id="catalogue-placeholder"`, "id="+expression, 1)
	}
	return config
}

func catalogueValue(payload map[string]any, field string) string {
	group := "built-in"
	if field == "name" {
		group = "custom-detectors"
	}
	items := listLimitEntries(payload, group)
	if len(items) != 1 {
		return ""
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := item[field].(string)
	return value
}

func TestAgentMemoryDetectorCatalogueValidation(t *testing.T) {
	for _, tc := range []struct{ name, expression, dependency, wantError string }{
		{name: "known", expression: `"email"`},
		{name: "future", expression: `"future-detector"`},
		{name: "unknown", expression: `tostring(terraform_data.value.output)`, dependency: `resource "terraform_data" "value" { input="future-detector" }`},
		{name: "empty", expression: `""`, wantError: "non-whitespace"},
		{name: "whitespace", expression: `"   "`, wantError: "non-whitespace"},
		{name: "null", expression: `null`, wantError: "Missing Configuration for Required Attribute|Missing required argument"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) }))
			defer server.Close()
			step := resource.TestStep{Config: testProviderConfig(t, server.URL) + catalogueConfig("id", tc.expression, tc.dependency), PlanOnly: true, ExpectNonEmptyPlan: true}
			if tc.wantError != "" {
				step.ExpectError = regexp.MustCompile(tc.wantError)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
			if calls.Load() != 0 {
				t.Fatal("planning reached API")
			}
		})
	}
}

func TestAgentMemoryDetectorCatalogueLifecycle(t *testing.T) {
	for _, field := range []string{"id", "name"} {
		for _, deferred := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/deferred=%t", field, deferred), func(t *testing.T) {
				var mu sync.Mutex
				var store map[string]any
				creates, deletes, patches, rejections := 0, 0, 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
						var payload map[string]any
						if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
							t.Error(err)
							w.WriteHeader(500)
							return
						}
						value := catalogueValue(payload, field)
						if value == "" {
							t.Error("configured detector value missing from request")
							w.WriteHeader(500)
							return
						}
						if value == "rejected-detector" {
							rejections++
							w.WriteHeader(400)
							_, _ = w.Write([]byte(`{"message":"detector ID or reserved name rejected by API"}`))
							return
						}
						if r.Method == http.MethodPost {
							creates++
							store = payload
						} else {
							patches++
							store["longTermMemoryExclusions"] = payload["longTermMemoryExclusions"]
						}
						_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
					case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
						_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
					case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && store != nil:
						store["storeId"] = "store-1"
						store["databaseId"] = "123"
						store["status"] = "READY"
						store["extractionStrategy"] = "INSTRUCT"
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
				config := func(value string) string {
					expression, dependency := fmt.Sprintf("%q", value), ""
					if deferred {
						dependency = fmt.Sprintf(`resource "terraform_data" "value" { input=%q }`, value)
						expression = "tostring(terraform_data.value.output)"
					}
					return testProviderConfig(t, server.URL) + catalogueConfig(field, expression, dependency)
				}
				check := func(value string) resource.TestCheckFunc {
					return func(state *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if creates != 1 || deletes != 0 || catalogueValue(store, field) != value {
							return fmt.Errorf("store replaced or API value changed unexpectedly")
						}
						return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")(state)
					}
				}
				// Acceptance of a future ID or reserved name is simulated, not a claim about the live catalogue.
				initial, updated := "future-detector", "future-detector-next"
				if field == "name" {
					initial = "email"
					updated = "phone"
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
					{Config: config(initial), Check: check(initial)},
					{Config: config(updated), Check: check(updated)},
					{Config: config(updated), PlanOnly: true},
					{Config: config("rejected-detector"), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
					// A failed apply may already have updated the terraform_data dependency.
					{Config: config(updated), Check: check(updated)},
					{Config: config(updated), PlanOnly: true},
					{Config: config(updated), ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateId: "store-1", ImportStateVerify: true},
				}})
				mu.Lock()
				defer mu.Unlock()
				if creates != 1 || deletes != 1 || patches != 1 || rejections != 1 || store != nil {
					t.Fatalf("unexpected lifecycle: creates=%d deletes=%d patches=%d rejections=%d", creates, deletes, patches, rejections)
				}
			})
		}
	}
}

func TestAgentMemoryDetectorCatalogueCreateRejected(t *testing.T) {
	for _, field := range []string{"id", "name"} {
		for _, deferred := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/deferred=%t", field, deferred), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != http.MethodPost || r.URL.Path != "/memory-stores" {
						t.Error("unexpected API request")
					}
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if catalogueValue(payload, field) != "rejected-detector" {
						t.Error("configured value changed before API")
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(400)
					_, _ = w.Write([]byte(`{"message":"detector ID or reserved name rejected by API"}`))
				}))
				defer server.Close()
				expression, dependency := `"rejected-detector"`, ""
				if deferred {
					expression = "tostring(terraform_data.value.output)"
					dependency = `resource "terraform_data" "value" { input="rejected-detector" }`
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + catalogueConfig(field, expression, dependency), ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
				if calls.Load() != 1 {
					t.Fatalf("expected only the rejected create, got %d requests", calls.Load())
				}
			})
		}
	}
}
