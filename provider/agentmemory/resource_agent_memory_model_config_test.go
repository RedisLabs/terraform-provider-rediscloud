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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryModelReplacementLifecycle(t *testing.T) {
	base := selectorConfig("llm", "provider", `"fake-provider"`, "")
	platform := `
resource "rediscloud_agent_memory" "example" {
 name="selector-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
}
`
	rotated := strings.ReplaceAll(base, "FAKE-STATIC", "FAKE-ROTATED")
	omitted := strings.ReplaceAll(rotated, "provider=\"fake-provider\"\n", "")
	omitted = strings.ReplaceAll(omitted, "model=\"fake-embedding\"\n", "")
	for _, tc := range []struct {
		name, before, after, errorPattern string
		replace                           bool
	}{
		{name: "llm provider", before: base, after: selectorConfig("llm", "provider", `"other-provider"`, ""), replace: true},
		{name: "embedding provider", before: base, after: selectorConfig("embedding", "provider", `"other-provider"`, ""), replace: true},
		{name: "embedding model", before: base, after: selectorConfig("embedding", "model", `"other-model"`, ""), replace: true},
		{name: "llm model", before: base, after: selectorConfig("llm", "model", `"other-model"`, "")},
		{name: "credential rotation", before: base, after: rotated},
		{name: "rotation with omitted immutable selectors", before: base, after: omitted},
		{name: "platform to customer models", before: platform, after: base, replace: true},
		{name: "customer to platform models", before: base, after: platform, replace: true},
		{name: "unknown provider resolving unchanged", before: base, after: selectorConfig("llm", "provider", `tostring(terraform_data.provider.output)`, `resource "terraform_data" "provider" { input="fake-provider" }`), replace: true},
		{name: "unknown provider resolving changed", before: base, after: selectorConfig("llm", "provider", `tostring(terraform_data.provider.output)`, `resource "terraform_data" "provider" { input="other-provider" }`), replace: true},
		{name: "replacement without credentials", before: base, after: regexp.MustCompile(`credentials \{[^}]*\}\n`).ReplaceAllString(selectorConfig("llm", "provider", `"other-provider"`, ""), ""), errorPattern: "Missing Agent Memory model credentials"},
		{name: "adding models without credentials", before: platform, after: regexp.MustCompile(`credentials \{[^}]*\}\n`).ReplaceAllString(base, ""), errorPattern: "Missing Agent Memory model credentials"},
		{name: "prevent destroy", before: base, after: strings.Replace(selectorConfig("llm", "provider", `"other-provider"`, ""), `database_id=123`, `database_id=123
 lifecycle { prevent_destroy=true }`, 1), errorPattern: "prevent_destroy|cannot be destroyed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
			var merge func(map[string]any, map[string]any)
			merge = func(dst, src map[string]any) {
				for key, value := range src {
					nested, nestedOK := value.(map[string]any)
					current, currentOK := dst[key].(map[string]any)
					if nestedOK && currentOK {
						merge(current, nested)
					} else {
						dst[key] = value
					}
				}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					require.NoError(t, json.NewDecoder(r.Body).Decode(&store))
					creates++
					store["storeId"] = fmt.Sprintf("store-%d", creates)
					store["databaseId"] = "123"
					store["status"] = "READY"
					store["extractionStrategy"] = "INSTRUCT"
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+store["storeId"].(string):
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					merge(store, payload)
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					id := "store-1"
					if store != nil {
						id = store["storeId"].(string)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": id}}})
				case r.Method == http.MethodGet && store != nil && r.URL.Path == "/memory-stores/"+store["storeId"].(string):
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodDelete && store != nil && r.URL.Path == "/memory-stores/"+store["storeId"].(string):
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			prefix := testProviderConfig(t, server.URL)
			const address = "rediscloud_agent_memory.example"
			steps := []resource.TestStep{{Config: prefix + tc.before, Check: resource.TestCheckResourceAttr(address, "id", "store-1")}}
			if tc.errorPattern != "" {
				steps = append(steps, resource.TestStep{Config: prefix + tc.after, PlanOnly: true, ExpectError: regexp.MustCompile(tc.errorPattern)}, resource.TestStep{Config: prefix + tc.before, PlanOnly: true})
			} else {
				action, expectedID := plancheck.ResourceActionUpdate, "store-1"
				if tc.replace {
					action = plancheck.ResourceActionDestroyBeforeCreate
					expectedID = "store-2"
				}
				steps = append(steps, resource.TestStep{Config: prefix + tc.after, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)}}, Check: resource.TestCheckResourceAttr(address, "id", expectedID)}, resource.TestStep{Config: prefix + tc.after, PlanOnly: true})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
			mu.Lock()
			defer mu.Unlock()
			expectedCreates, expectedPatches := 1, 0
			if tc.replace {
				expectedCreates = 2
			} else if tc.errorPattern == "" {
				expectedPatches = 1
			}
			assert.Equal(t, expectedCreates, creates)
			assert.Equal(t, expectedPatches, patches)
			assert.Equal(t, expectedCreates, deletes)
			assert.Nil(t, store)
		})
	}
}
