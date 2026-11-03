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

func TestAgentMemoryCustomTypeReplacementLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, before, after, restore, errorPattern           string
		beforeVariables, afterVariables, restoreVariables    config.Variables
		replace, update, localUpdate, protect, hasDependency bool
	}{
		{
			name:   "unchanged",
			before: "testdata/lifecycle/custom_types/definition.tf",
			after:  "testdata/lifecycle/custom_types/definition.tf",
		},
		{
			name:   "add type",
			before: "testdata/lifecycle/custom_types/definition.tf",
			after:  "testdata/lifecycle/custom_types/two_types.tf",
			update: true,
		},
		{
			name:           "prompt update",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_extraction_strategy_prompt": config.StringVariable("Extract durable preferences")},
			update:         true,
		},
		{
			name:           "disable extraction",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_extraction_strategy_enabled": config.BoolVariable(false)},
			update:         true,
		},
		{
			name:   "add extraction strategy",
			before: "testdata/lifecycle/custom_types/without_strategy.tf",
			after:  "testdata/lifecycle/custom_types/definition.tf",
			update: true,
		},
		{
			name:    "remove type",
			before:  "testdata/lifecycle/custom_types/two_types.tf",
			after:   "testdata/lifecycle/custom_types/single_field.tf",
			replace: true,
		},
		{
			name:    "remove all types",
			before:  "testdata/lifecycle/custom_types/definition.tf",
			after:   "testdata/lifecycle/custom_types/none.tf",
			replace: true,
		},
		{
			name:           "rename type",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_name": config.StringVariable("renamed")},
			replace:        true,
		},
		{
			name:           "change description",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_description": config.StringVariable("Durable preferences")},
			replace:        true,
		},
		{
			name:            "add field",
			before:          "testdata/lifecycle/custom_types/single_field.tf",
			beforeVariables: config.Variables{"custom_memory_types_name": config.StringVariable("preference")},
			after:           "testdata/lifecycle/custom_types/definition.tf",
			replace:         true,
		},
		{
			name:           "remove field",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/single_field.tf",
			afterVariables: config.Variables{"custom_memory_types_name": config.StringVariable("preference")},
			replace:        true,
		},
		{
			name:           "rename field",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_fields_0_name": config.StringVariable("category")},
			replace:        true,
		},
		{
			name:           "change field type",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_fields_0_type": config.StringVariable("int")},
			replace:        true,
		},
		{
			name:           "change field description",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{"custom_memory_types_fields_0_description": config.StringVariable("Preference topic")},
			replace:        true,
		},
		{
			name:    "remove extraction strategy",
			before:  "testdata/lifecycle/custom_types/definition.tf",
			after:   "testdata/lifecycle/custom_types/without_strategy.tf",
			replace: true,
		},
		{
			name:        "reorder types",
			before:      "testdata/lifecycle/custom_types/two_types.tf",
			after:       "testdata/lifecycle/custom_types/two_types_reordered.tf",
			update:      true,
			localUpdate: true,
		},
		{
			name:   "reorder fields",
			before: "testdata/lifecycle/custom_types/definition.tf",
			after:  "testdata/lifecycle/custom_types/definition.tf",
			afterVariables: config.Variables{
				"custom_memory_types_fields_0_description": config.StringVariable("Confidence"),
				"custom_memory_types_fields_0_name":        config.StringVariable("confidence"),
				"custom_memory_types_fields_0_type":        config.StringVariable("float"),
				"custom_memory_types_fields_1_description": config.StringVariable("Topic"),
				"custom_memory_types_fields_1_name":        config.StringVariable("topic"),
				"custom_memory_types_fields_1_type":        config.StringVariable("str"),
			},
			update:      true,
			localUpdate: true,
		},
		{
			name:          "unknown existing name unchanged",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/deferred_name.tf",
			replace:       true,
			hasDependency: true,
		},
		{
			name:           "unknown existing name changed",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/deferred_name.tf",
			afterVariables: config.Variables{"value_input": config.StringVariable("renamed")},
			replace:        true,
			hasDependency:  true,
		},
		{
			name:          "unknown existing description unchanged",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/deferred_description.tf",
			replace:       true,
			hasDependency: true,
		},
		{
			name:           "unknown existing description changed",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/deferred_description.tf",
			afterVariables: config.Variables{"value_input": config.StringVariable("Changed")},
			replace:        true,
			hasDependency:  true,
		},
		{
			name:          "unknown existing field unchanged",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/deferred_field_type.tf",
			replace:       true,
			hasDependency: true,
		},
		{
			name:           "unknown existing field changed",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/deferred_field_type.tf",
			afterVariables: config.Variables{"value_input": config.StringVariable("int")},
			replace:        true,
			hasDependency:  true,
		},
		{
			name:          "unknown new type name",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/additional_deferred_name.tf",
			update:        true,
			hasDependency: true,
		},
		{
			name:          "unknown new field type",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/additional_deferred_field_type.tf",
			update:        true,
			hasDependency: true,
		},
		{
			name:          "unknown mutable prompt",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/deferred_prompt.tf",
			update:        true,
			hasDependency: true,
		},
		{
			name:          "unknown mutable enabled",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/deferred_enabled.tf",
			update:        true,
			hasDependency: true,
		},
		{
			name:         "duplicate names",
			before:       "testdata/lifecycle/custom_types/definition.tf",
			after:        "testdata/lifecycle/custom_types/duplicate_names.tf",
			errorPattern: "Duplicate Agent Memory custom memory type",
			restore:      "testdata/lifecycle/custom_types/definition.tf",
		},
		{
			name:           "unknown duplicate addition",
			before:         "testdata/lifecycle/custom_types/definition.tf",
			after:          "testdata/lifecycle/custom_types/additional_deferred_name.tf",
			afterVariables: config.Variables{"value_input": config.StringVariable("preference")},
			errorPattern:   "Duplicate Agent Memory custom memory type",
			restore:        "testdata/lifecycle/custom_types/restore_after_duplicate.tf",
			hasDependency:  true,
		},
		{
			name:         "removal prevent destroy",
			before:       "testdata/lifecycle/custom_types/definition.tf",
			after:        "testdata/lifecycle/custom_types/protected_empty.tf",
			protect:      true,
			errorPattern: "prevent_destroy|cannot be destroyed",
			restore:      "testdata/lifecycle/custom_types/definition.tf",
		},
		{
			name:         "redefinition prevent destroy",
			before:       "testdata/lifecycle/custom_types/definition.tf",
			after:        "testdata/lifecycle/custom_types/protected_definition.tf",
			protect:      true,
			errorPattern: "prevent_destroy|cannot be destroyed",
			restore:      "testdata/lifecycle/custom_types/definition.tf",
		},
		{
			name:          "unknown name prevent destroy",
			before:        "testdata/lifecycle/custom_types/definition.tf",
			after:         "testdata/lifecycle/custom_types/protected_deferred_name.tf",
			protect:       true,
			errorPattern:  "prevent_destroy|cannot be destroyed",
			restore:       "testdata/lifecycle/custom_types/definition.tf",
			hasDependency: true,
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
					store["databaseId"] = "123"
					store["status"] = "READY"
					store["extractionStrategy"] = "INSTRUCT"
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": lastID}}})
				case r.Method == http.MethodGet && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.NotContains(t, payload, "customMemoryTypes")
					entries, _ := store["customMemoryTypes"].([]any)
					additions, _ := payload["addCustomMemoryTypes"].([]any)
					entries = append(entries, additions...)
					updates, _ := payload["updateCustomMemoryTypeStrategies"].([]any)
					for _, u := range updates {
						update := u.(map[string]any)
						found := false
						for _, v := range entries {
							entry := v.(map[string]any)
							if entry["name"] != update["typeName"] {
								continue
							}
							found = true
							extraction, _ := entry["extractionStrategy"].(map[string]any)
							if extraction == nil {
								extraction = map[string]any{}
							}
							for _, key := range []string{"prompt", "enabled"} {
								if value, ok := update[key]; ok {
									extraction[key] = value
								}
							}
							entry["extractionStrategy"] = extraction
						}
						assert.True(t, found, "strategy update must reference an existing type")
					}
					if len(entries) > 0 {
						store["customMemoryTypes"] = entries
					}
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
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
				ConfigFile:      config.StaticFile(tc.before),
				ConfigVariables: tc.beforeVariables,
				Check:           resource.TestCheckResourceAttr(address, "id", "store-1"),
			}}
			if tc.errorPattern != "" {
				step := resource.TestStep{
					ConfigFile:      config.StaticFile(tc.after),
					ConfigVariables: tc.afterVariables,
					ExpectError:     regexp.MustCompile(tc.errorPattern),
				}
				if tc.protect || !tc.hasDependency {
					step.PlanOnly = true
				}
				steps = append(steps, step, resource.TestStep{
					ConfigFile:      config.StaticFile(tc.restore),
					ConfigVariables: tc.restoreVariables,
					PlanOnly:        true,
				})
			} else {
				action, expectedID := plancheck.ResourceActionNoop, "store-1"
				if tc.replace {
					action, expectedID = plancheck.ResourceActionDestroyBeforeCreate, "store-2"
				} else if tc.update {
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
			if tc.update && !tc.localUpdate {
				expectedPatches = 1
			}
			assert.Equal(t, expectedCreates, creates)
			assert.Equal(t, expectedCreates, deletes)
			assert.Equal(t, expectedPatches, patches)
			assert.Nil(t, store)
		})
	}
}
