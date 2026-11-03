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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryCreateOnlyReplacementLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, before, after, errorPattern string
		afterVariables                    config.Variables
		replace, rename                   bool
	}{
		{
			name:   "unchanged",
			before: "testdata/lifecycle/stores/identity.tf",
			after:  "testdata/lifecycle/stores/identity.tf",
		},
		{
			name:           "database change",
			before:         "testdata/lifecycle/stores/identity.tf",
			after:          "testdata/lifecycle/stores/identity.tf",
			afterVariables: config.Variables{"database_id": config.IntegerVariable(456)},
			replace:        true,
		},
		{
			name:    "unknown database change",
			before:  "testdata/lifecycle/stores/identity.tf",
			after:   "testdata/lifecycle/stores/deferred_database.tf",
			replace: true,
		},
		{
			name:           "unknown database unchanged",
			before:         "testdata/lifecycle/stores/identity.tf",
			after:          "testdata/lifecycle/stores/deferred_database.tf",
			afterVariables: config.Variables{"value_input": config.IntegerVariable(123)},
			replace:        true,
		},
		{
			name:           "strategy change",
			before:         "testdata/lifecycle/stores/identity.tf",
			after:          "testdata/lifecycle/stores/identity.tf",
			afterVariables: config.Variables{"extraction_strategy": config.StringVariable("FAKE-FUTURE-STRATEGY")},
			replace:        true,
		},
		{
			name:    "unknown strategy change",
			before:  "testdata/lifecycle/stores/identity.tf",
			after:   "testdata/lifecycle/stores/deferred_strategy.tf",
			replace: true,
		},
		{
			name:           "unknown strategy unchanged",
			before:         "testdata/lifecycle/stores/identity.tf",
			after:          "testdata/lifecycle/stores/deferred_strategy.tf",
			afterVariables: config.Variables{"value_input": config.StringVariable("INSTRUCT")},
			replace:        true,
		},
		{
			name:   "omitted strategy",
			before: "testdata/lifecycle/stores/identity.tf",
			after:  "testdata/lifecycle/stores/omitted_strategy.tf",
		},
		{
			name:           "omitted strategy with rename",
			before:         "testdata/lifecycle/stores/identity.tf",
			after:          "testdata/lifecycle/stores/omitted_strategy.tf",
			afterVariables: config.Variables{"name": config.StringVariable("renamed")},
			rename:         true,
		},
		{
			name:         "database prevent destroy",
			before:       "testdata/lifecycle/stores/identity.tf",
			after:        "testdata/lifecycle/stores/protected_identity.tf",
			errorPattern: "prevent_destroy|cannot be destroyed",
		},
		{
			name:   "strategy prevent destroy",
			before: "testdata/lifecycle/stores/identity.tf",
			after:  "testdata/lifecycle/stores/protected_identity.tf",
			afterVariables: config.Variables{
				"database_id":         config.IntegerVariable(123),
				"extraction_strategy": config.StringVariable("FAKE-FUTURE-STRATEGY"),
			},
			errorPattern: "prevent_destroy|cannot be destroyed",
		},
		{
			name:         "unknown database prevent destroy",
			before:       "testdata/lifecycle/stores/identity.tf",
			after:        "testdata/lifecycle/stores/protected_deferred_database.tf",
			errorPattern: "prevent_destroy|cannot be destroyed",
		},
		{
			name:         "unknown strategy prevent destroy",
			before:       "testdata/lifecycle/stores/identity.tf",
			after:        "testdata/lifecycle/stores/protected_deferred_strategy.tf",
			errorPattern: "prevent_destroy|cannot be destroyed",
		},
		{
			name:   "empty strategy prevent destroy",
			before: "testdata/lifecycle/stores/identity.tf",
			after:  "testdata/lifecycle/stores/protected_identity.tf",
			afterVariables: config.Variables{
				"database_id":         config.IntegerVariable(123),
				"extraction_strategy": config.StringVariable(""),
			},
			errorPattern: "prevent_destroy|cannot be destroyed",
		},
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
			testProviderConfig(t, server.URL)
			const address = "rediscloud_agent_memory.example"
			steps := []resource.TestStep{{
				ConfigFile: config.StaticFile(tc.before),
				Check:      resource.TestCheckResourceAttr(address, "id", "store-1"),
			}}
			if tc.errorPattern != "" {
				steps = append(steps, resource.TestStep{
					ConfigFile:      config.StaticFile(tc.after),
					ConfigVariables: tc.afterVariables,
					PlanOnly:        true,
					ExpectError:     regexp.MustCompile(tc.errorPattern),
				}, resource.TestStep{
					ConfigFile: config.StaticFile(tc.before),
					PlanOnly:   true,
				})
			} else {
				action, expectedID := plancheck.ResourceActionNoop, "store-1"
				if tc.replace {
					action, expectedID = plancheck.ResourceActionDestroyBeforeCreate, "store-2"
				} else if tc.rename {
					action = plancheck.ResourceActionUpdate
				}
				steps = append(steps, resource.TestStep{
					ConfigFile:       config.StaticFile(tc.after),
					ConfigVariables:  tc.afterVariables,
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)}},
					Check:            resource.TestCheckResourceAttr(address, "id", expectedID),
				}, resource.TestStep{
					ConfigFile:      config.StaticFile(tc.after),
					ConfigVariables: tc.afterVariables,
					PlanOnly:        true,
				})
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
