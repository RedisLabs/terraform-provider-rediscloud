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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryUnknownListsLifecycle(t *testing.T) {
	for _, kind := range []string{"memory_types", "fields", "built_in", "custom"} {
		t.Run(kind, func(t *testing.T) {
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
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/"+lastID:
					if store == nil {
						http.NotFound(w, r)
						return
					}
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.Contains(t, payload, "longTermMemoryExclusions")
					assert.NotContains(t, payload, "addCustomMemoryTypes")
					store["longTermMemoryExclusions"] = payload["longTermMemoryExclusions"]
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodDelete && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", 500)
				}
			}))
			defer server.Close()
			testProviderConfig(t, server.URL)
			action := plancheck.ResourceActionUpdate
			if kind == "memory_types" || kind == "fields" {
				action = plancheck.ResourceActionDestroyBeforeCreate
			}
			check := func(count string) resource.TestCheckFunc {
				attr := map[string]string{"memory_types": "custom_memory_types.#", "fields": "custom_memory_types.0.fields.#", "built_in": "long_term_memory_exclusions.0.built_in_detectors.0.detectors.#", "custom": "long_term_memory_exclusions.0.custom_detectors.0.detectors.#"}[kind]
				return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", attr, count)
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(kind), "items": config.ListVariable(config.StringVariable("first"))},
						PlanOnly:        true, ExpectNonEmptyPlan: true,
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(kind), "items": config.ListVariable(config.StringVariable("first"))},
						Check:           check("1"),
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(kind), "items": config.ListVariable(config.StringVariable("first"))},
						PlanOnly:        true,
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(kind), "items": config.ListVariable(config.StringVariable("first"), config.StringVariable("second"))},
						PlanOnly:        true, ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectResourceAction("rediscloud_agent_memory.example", action)}},
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(kind), "items": config.ListVariable(config.StringVariable("first"), config.StringVariable("second"))},
						Check:           check("2"),
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(kind), "items": config.ListVariable(config.StringVariable("first"), config.StringVariable("second"))},
						PlanOnly:        true,
					},
				},
			})
			mu.Lock()
			defer mu.Unlock()
			assert.Nil(t, store)
			assert.Equal(t, creates, deletes)
			if kind == "memory_types" || kind == "fields" {
				assert.Equal(t, 2, creates)
				assert.Zero(t, patches)
			} else {
				assert.Equal(t, 1, creates)
				assert.Equal(t, 1, patches)
			}
		})
	}
}

func TestAgentMemoryUnknownListsResolveInvalid(t *testing.T) {
	for _, tc := range []struct {
		kind         string
		items        []config.Variable
		errorPattern string
	}{
		{"memory_types", []config.Variable{config.StringVariable("duplicate"), config.StringVariable("duplicate")}, "Duplicate Agent Memory custom memory type"},
		{"fields", []config.Variable{config.StringVariable("")}, `(?s)Invalid Attribute Value Length.*custom_memory_types\[0\]\.fields\[0\]\.name`},
		{"built_in", []config.Variable{config.StringVariable(" ")}, `(?s)Invalid Attribute Value Match.*built_in_detectors\[0\]\.detectors\[0\]\.id`},
		{"custom", []config.Variable{config.StringVariable("duplicate"), config.StringVariable("duplicate")}, "Duplicate Agent Memory custom detector name"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Error(w, "invalid resolved lists must not call the API", 500)
			}))
			defer server.Close()
			testProviderConfig(t, server.URL)
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(tc.kind), "items": config.ListVariable(tc.items...)},
						PlanOnly:        true, ExpectNonEmptyPlan: true,
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(tc.kind), "items": config.ListVariable(tc.items...)},
						ExpectError:     regexp.MustCompile(tc.errorPattern),
					},
					{
						ConfigFile:      config.StaticFile("testdata/unknown_lists.tf"),
						ConfigVariables: config.Variables{"kind": config.StringVariable(tc.kind), "items": config.ListVariable(config.StringVariable("valid"))},
						PlanOnly:        true, ExpectNonEmptyPlan: true,
					},
				},
			})
			assert.Zero(t, calls.Load())
		})
	}
}
