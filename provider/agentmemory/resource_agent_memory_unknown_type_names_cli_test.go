package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestCustomTypeUnknownNameLifecycle(t *testing.T) {
	for _, scenario := range []struct{ name, input, errorPattern string }{
		{name: "same-name", input: "preference:after"},
		{name: "resolved-rename", input: "renamed:after"},
		{name: "resolved-duplicate", input: "support:after", errorPattern: "Duplicate Agent Memory custom memory type"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var mu sync.Mutex
			var created map[string]any
			var creates, deletes, patches int
			lastID := ""
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					creates++
					lastID = fmt.Sprintf("store-%d", creates)
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": lastID}}})
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/"+lastID:
					if created == nil {
						http.NotFound(w, r)
						return
					}
					response := map[string]any{}
					for k, v := range created {
						response[k] = v
					}
					response["storeId"] = lastID
					response["databaseId"] = "123"
					response["status"] = "READY"
					response["extractionStrategy"] = "INSTRUCT"
					response["createdAt"] = "2026-10-16T12:00:00Z"
					response["endpoint"] = "https://memory.example.test"
					response["endpoints"] = []map[string]any{{"url": "https://memory.example.test", "provider": "AWS", "region": "us-east-1", "egressIps": []string{}, "isAccessible": true}}
					_ = json.NewEncoder(w).Encode(response)
				case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/"+lastID:
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					if len(payload) != 0 {
						t.Errorf("unchanged types must not generate API edits: %v", payload)
					}
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/"+lastID:
					created = nil
					deletes++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				t.Logf("LOCAL MOCK ONLY: creates=%d deletes=%d remaining=%t", creates, deletes, created != nil)
				expected := 2
				if scenario.errorPattern != "" {
					expected = 1
				}
				if created != nil || creates != expected || deletes != expected {
					t.Error("unexpected replacement counts or remaining local fixture")
				}
				if patches != 0 {
					t.Errorf("invalid resolved names reached API update: %d", patches)
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			config := func(input string) string {
				return prefix + fmt.Sprintf(`
resource "terraform_data" "source" { input=%q }
resource "rediscloud_agent_memory" "audit" {
 name="audit-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 custom_memory_types {
  name=split(":",tostring(terraform_data.source.output))[0]
  description="Shopping preferences"
  fields {
   name="topic"
   description="Preference topic"
   type="str"
  }
  extraction_strategy {
   prompt="Extract preferences"
   enabled=true
  }
 }
 custom_memory_types {
  name="support"
  description="Support requests"
  extraction_strategy {
   prompt="Extract support requests"
   enabled=true
  }
 }
}
`, input)
			}
			// Unknown existing names conservatively plan replacement, even when unchanged.
			initial := config("preference:before")
			changed := config(scenario.input)
			check := resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("rediscloud_agent_memory.audit", "id", "store-2"),
				resource.TestCheckResourceAttr("rediscloud_agent_memory.audit", "custom_memory_types.0.name", strings.Split(scenario.input, ":")[0]))
			steps := []resource.TestStep{{Config: initial, Check: resource.TestCheckResourceAttr("rediscloud_agent_memory.audit", "id", "store-1")}, {Config: changed, PlanOnly: true, ExpectNonEmptyPlan: true}}
			if scenario.errorPattern == "" {
				steps = append(steps, resource.TestStep{Config: changed, Check: check}, resource.TestStep{Config: changed, PlanOnly: true})
			} else {
				steps = append(steps, resource.TestStep{Config: changed, ExpectError: regexp.MustCompile(scenario.errorPattern)})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
		})
	}
}
