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

func TestAgentMemoryModelReplacementLifecycle(t *testing.T) {
	for _, tc := range []struct {
		name, before, after, errorPattern string
		afterVariables                    config.Variables
		replace                           bool
	}{
		{
			name:           "llm provider",
			before:         "testdata/lifecycle/models/selectors.tf",
			after:          "testdata/lifecycle/models/selectors.tf",
			afterVariables: config.Variables{"llm_provider": config.StringVariable("other-provider")},
			replace:        true,
		},
		{
			name:           "embedding provider",
			before:         "testdata/lifecycle/models/selectors.tf",
			after:          "testdata/lifecycle/models/selectors.tf",
			afterVariables: config.Variables{"embedding_provider": config.StringVariable("other-provider")},
			replace:        true,
		},
		{
			name:           "embedding model",
			before:         "testdata/lifecycle/models/selectors.tf",
			after:          "testdata/lifecycle/models/selectors.tf",
			afterVariables: config.Variables{"embedding_model": config.StringVariable("other-model")},
			replace:        true,
		},
		{
			name:           "llm model",
			before:         "testdata/lifecycle/models/selectors.tf",
			after:          "testdata/lifecycle/models/selectors.tf",
			afterVariables: config.Variables{"llm_model": config.StringVariable("other-model")},
		},
		{
			name:   "credential rotation",
			before: "testdata/lifecycle/models/selectors.tf",
			after:  "testdata/lifecycle/models/selectors.tf",
			afterVariables: config.Variables{
				"embedding_credentials_api_key":         config.StringVariable("FAKE-ROTATED"),
				"embedding_credentials_api_key_version": config.IntegerVariable(2),
				"llm_credentials_api_key":               config.StringVariable("FAKE-ROTATED"),
				"llm_credentials_api_key_version":       config.IntegerVariable(2),
			},
		},
		{
			name:         "rotation with omitted immutable selectors",
			before:       "testdata/lifecycle/models/selectors.tf",
			after:        "testdata/lifecycle/models/rotated_without_selectors.tf",
			errorPattern: "Invalid Attribute Combination",
		},
		{
			name:    "platform to customer models",
			before:  "testdata/lifecycle/models/managed.tf",
			after:   "testdata/lifecycle/models/selectors.tf",
			replace: true,
		},
		{
			name:    "customer to platform models",
			before:  "testdata/lifecycle/models/selectors.tf",
			after:   "testdata/lifecycle/models/managed.tf",
			replace: true,
		},
		{
			name:    "unknown provider resolving unchanged",
			before:  "testdata/lifecycle/models/selectors.tf",
			after:   "testdata/lifecycle/models/deferred_provider.tf",
			replace: true,
		},
		{
			name:           "unknown provider resolving changed",
			before:         "testdata/lifecycle/models/selectors.tf",
			after:          "testdata/lifecycle/models/deferred_provider.tf",
			afterVariables: config.Variables{"provider_input": config.StringVariable("other-provider")},
			replace:        true,
		},
		{
			name:         "replacement without credentials",
			before:       "testdata/lifecycle/models/selectors.tf",
			after:        "testdata/lifecycle/models/without_credentials.tf",
			errorPattern: "Missing Agent Memory model credentials",
		},
		{
			name:           "adding models without credentials",
			before:         "testdata/lifecycle/models/managed.tf",
			after:          "testdata/lifecycle/models/without_credentials.tf",
			afterVariables: config.Variables{"llm_provider": config.StringVariable("fake-provider")},
			errorPattern:   "Missing Agent Memory model credentials",
		},
		{
			name:   "adding models without immutable selectors",
			before: "testdata/lifecycle/models/managed.tf",
			after:  "testdata/lifecycle/models/rotated_without_selectors.tf",
			afterVariables: config.Variables{
				"embedding_credentials_api_key":         config.StringVariable("FAKE-STATIC"),
				"embedding_credentials_api_key_version": config.IntegerVariable(1),
				"llm_credentials_api_key":               config.StringVariable("FAKE-STATIC"),
				"llm_credentials_api_key_version":       config.IntegerVariable(1),
			},
			errorPattern: "Invalid Attribute Combination",
		},
		{
			name:         "prevent destroy",
			before:       "testdata/lifecycle/models/selectors.tf",
			after:        "testdata/lifecycle/models/protected.tf",
			errorPattern: "prevent_destroy|cannot be destroyed",
		},
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
				action, expectedID := plancheck.ResourceActionUpdate, "store-1"
				if tc.replace {
					action = plancheck.ResourceActionDestroyBeforeCreate
					expectedID = "store-2"
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
