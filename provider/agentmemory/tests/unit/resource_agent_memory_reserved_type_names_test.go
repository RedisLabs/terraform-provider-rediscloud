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

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func reservedTypeConfig(name, dependency, fieldName string) string {
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="reserved-type-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 custom_memory_types {
  name=%s
  description="Custom preferences"
  fields {
   name=%q
   description="Preference topic"
   type="str"
  }
  extraction_strategy {
   prompt="Extract preferences"
   enabled=true
  }
 }
}
`, name, fieldName)
}

func TestAgentMemoryReservedTypeNameValidation(t *testing.T) {
	cases := []struct{ name, expression, fieldName, errorPattern string }{
		{name: "custom-name", expression: `"preference"`},
		{name: "maximum-length", expression: fmt.Sprintf("%q", strings.Repeat("a", 64))},
		{name: "above-former-length-deferred-to-api", expression: fmt.Sprintf("%q", strings.Repeat("a", 65))},
		{name: "empty-name", expression: `""`},
		{name: "null-name", expression: "null", errorPattern: "Missing Configuration for Required Attribute|Missing required argument"},
		{name: "format-deferred-to-api", expression: `"1preference"`},
	}
	for _, name := range []string{"semantic", "episodic", "message", "session_summary_view"} {
		cases = append(cases,
			struct{ name, expression, fieldName, errorPattern string }{name: "reserved-" + name, expression: fmt.Sprintf("%q", name), errorPattern: "none of"},
			struct{ name, expression, fieldName, errorPattern string }{name: "custom-suffix-" + name, expression: fmt.Sprintf("%q", name+"_custom")},
			// Built-in type names are not reserved as structured field names by this contract.
			struct{ name, expression, fieldName, errorPattern string }{name: "field-" + name, expression: `"preference"`, fieldName: name},
		)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "plan must not call HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("plan reached HTTP")
				}
			}()
			fieldName := c.fieldName
			if fieldName == "" {
				fieldName = "topic"
			}
			config := testProviderConfig(t, server.URL) + reservedTypeConfig(c.expression, "", fieldName)
			step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(c.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryReservedTypeNameUnknown(t *testing.T) {
	for _, name := range []string{"semantic", "episodic", "message", "session_summary_view"} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "reserved name must not call HTTP", 400)
			}))
			defer server.Close()
			defer func() {
				if calls.Load() != 0 {
					t.Error("reserved resolved name reached HTTP")
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			dependency := fmt.Sprintf(`resource "terraform_data" "name" { input=%q }`, name)
			config := prefix + reservedTypeConfig("tostring(terraform_data.name.output)", dependency, "topic")
			// Restore valid configuration so schema validation does not block harness cleanup.
			restored := prefix + reservedTypeConfig(`"preference"`, dependency, "topic")
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config, ExpectError: regexp.MustCompile("none of")},
				{Config: restored, PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}

func TestAgentMemoryReservedTypeNameValidLifecycle(t *testing.T) {
	for _, name := range []string{"preference", "semantic_custom"} {
		t.Run(name, func(t *testing.T) {
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
						t.Error("expected one custom type")
						w.WriteHeader(400)
						return
					}
					item, ok := items[0].(map[string]any)
					if !ok || item["name"] != name {
						t.Error("resolved valid name was not preserved")
						w.WriteHeader(400)
						return
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
				if creates != 1 || deletes != 1 || store != nil {
					t.Errorf("fixture cleanup failed: creates=%d deletes=%d remaining=%t", creates, deletes, store != nil)
				}
			}()
			testProviderConfig(t, server.URL)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile:      config.StaticFile("testdata/lifecycle/custom_types/custom_type_deferred_name.tf"),
					ConfigVariables: config.Variables{"name": config.StringVariable(name)},
					Check:           resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.name", name)),
				},
				{
					ConfigFile:      config.StaticFile("testdata/lifecycle/custom_types/custom_type_deferred_name.tf"),
					ConfigVariables: config.Variables{"name": config.StringVariable(name)},
					PlanOnly:        true,
				},
			}})
		})
	}
}
