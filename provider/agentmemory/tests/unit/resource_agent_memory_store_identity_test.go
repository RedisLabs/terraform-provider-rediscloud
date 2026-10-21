package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryCreateOnlyReplacementLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, database, strategy, dependency, errorPattern string
		replace, rename, preventDestroy                    bool
	}{
		{name: "unchanged", database: "123", strategy: `"INSTRUCT"`},
		{name: "database change", database: "456", strategy: `"INSTRUCT"`, replace: true},
		{name: "unknown database change", database: "terraform_data.value.output", strategy: `"INSTRUCT"`, dependency: `resource "terraform_data" "value" { input=456 }`, replace: true},
		{name: "unknown database unchanged", database: "terraform_data.value.output", strategy: `"INSTRUCT"`, dependency: `resource "terraform_data" "value" { input=123 }`, replace: true},
		{name: "strategy change", database: "123", strategy: `"FAKE-FUTURE-STRATEGY"`, replace: true},
		{name: "unknown strategy change", database: "123", strategy: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="FAKE-FUTURE-STRATEGY" }`, replace: true},
		{name: "unknown strategy unchanged", database: "123", strategy: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="INSTRUCT" }`, replace: true},
		{name: "omitted strategy", database: "123"},
		{name: "omitted strategy with rename", database: "123", rename: true},
		{name: "database prevent destroy", database: "456", strategy: `"INSTRUCT"`, preventDestroy: true, errorPattern: "prevent_destroy|cannot be destroyed"},
		{name: "strategy prevent destroy", database: "123", strategy: `"FAKE-FUTURE-STRATEGY"`, preventDestroy: true, errorPattern: "prevent_destroy|cannot be destroyed"},
		{name: "unknown database prevent destroy", database: "terraform_data.value.output", strategy: `"INSTRUCT"`, dependency: `resource "terraform_data" "value" { input=123 }`, preventDestroy: true, errorPattern: "prevent_destroy|cannot be destroyed"},
		{name: "unknown strategy prevent destroy", database: "123", strategy: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="INSTRUCT" }`, preventDestroy: true, errorPattern: "prevent_destroy|cannot be destroyed"},
		{name: "empty strategy prevent destroy", database: "123", strategy: `""`, preventDestroy: true, errorPattern: "prevent_destroy|cannot be destroyed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
			lastID := ""
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					require.NoError(t, json.NewDecoder(r.Body).Decode(&store))
					creates++
					lastID = fmt.Sprintf("store-%d", creates)
					store["storeId"] = lastID
					store["databaseId"] = fmt.Sprint(store["databaseId"])
					store["status"] = "READY"
					if _, ok := store["extractionStrategy"]; !ok {
						store["extractionStrategy"] = "INSTRUCT"
					}
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.NotContains(t, payload, "databaseId")
					assert.NotContains(t, payload, "extractionStrategy")
					for key, value := range payload {
						store[key] = value
					}
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": lastID}}})
				case r.Method == http.MethodGet && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodDelete && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := func(name, database, strategy, dependency string, protect bool) string {
				strategyLine, lifecycle := "", ""
				if strategy != "" {
					strategyLine = "extraction_strategy=" + strategy
				}
				if protect {
					lifecycle = "lifecycle { prevent_destroy=true }"
				}
				return testProviderConfig(t, server.URL) + dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name=%q
 database_id=%s
 %s
 %s
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
}
`, name, database, strategyLine, lifecycle)
			}
			const address = "rediscloud_agent_memory.example"
			initial := config("store", "123", `"INSTRUCT"`, "", false)
			name := "store"
			if tc.rename {
				name = "renamed"
			}
			changed := config(name, tc.database, tc.strategy, tc.dependency, tc.preventDestroy)
			steps := []resource.TestStep{{Config: initial, Check: resource.TestCheckResourceAttr(address, "id", "store-1")}}
			if tc.errorPattern != "" {
				steps = append(steps, resource.TestStep{Config: changed, PlanOnly: true, ExpectError: regexp.MustCompile(tc.errorPattern)}, resource.TestStep{Config: initial, PlanOnly: true})
			} else {
				action, expectedID := plancheck.ResourceActionNoop, "store-1"
				if tc.replace {
					action, expectedID = plancheck.ResourceActionDestroyBeforeCreate, "store-2"
				} else if tc.rename {
					action = plancheck.ResourceActionUpdate
				}
				steps = append(steps, resource.TestStep{Config: changed, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)}}, Check: resource.TestCheckResourceAttr(address, "id", expectedID)}, resource.TestStep{Config: changed, PlanOnly: true})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
			mu.Lock()
			defer mu.Unlock()
			expectedCreates, expectedPatches := 1, 0
			if tc.replace {
				expectedCreates = 2
			}
			if tc.rename {
				expectedPatches = 1
			}
			assert.Equal(t, expectedCreates, creates)
			assert.Equal(t, expectedPatches, patches)
			assert.Equal(t, expectedCreates, deletes)
			assert.Nil(t, store)
		})
	}
}
