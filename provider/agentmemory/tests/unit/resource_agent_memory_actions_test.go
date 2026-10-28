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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func detectorActionsConfig(expression, dependency string) string {
	config := reservedDetectorConfig(`"tenant_id"`, dependency, "email")
	action := ""
	if expression != "" {
		action = "action=" + expression
	}
	return strings.ReplaceAll(config, `action="redact"`, action)
}

func detectorActions(payload map[string]any) []map[string]any {
	var items []map[string]any
	for _, kind := range []string{"built-in", "custom-detectors"} {
		entries := listLimitEntries(payload, kind)
		if len(entries) != 1 {
			return nil
		}
		item, ok := entries[0].(map[string]any)
		if !ok {
			return nil
		}
		items = append(items, item)
	}
	return items
}

func TestAgentMemoryDetectorActionsByAPI(t *testing.T) {
	for _, test := range []struct {
		name          string
		unknown, null bool
	}{
		{name: "known"}, {
			name:    "unknown",
			unknown: true,
		}, {
			name: "null",
			null: true,
		}, {
			name:    "unknown-null",
			unknown: true,
			null:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
			var observed []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					items := detectorActions(payload)
					if len(items) != 2 {
						t.Error("missing detector entries")
						w.WriteHeader(400)
						return
					}
					action, _ := items[0]["action"].(string)
					if r.Method == http.MethodPost {
						creates++
						for _, item := range items {
							_, present := item["action"]
							assert.False(t, present, "omitted/null actions must not be sent")
						}
					} else {
						patches++
						assert.NotEmpty(t, action)
						assert.Equal(t, action, items[1]["action"])
					}
					observed = append(observed, action)
					if action == "rejected-by-api" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"action rejected by API"}`))
						return
					}
					// Simulate an API default and acceptance of a new action.
					if action == "" {
						for _, item := range items {
							item["action"] = "redact"
						}
					}
					if store == nil {
						store = payload
					} else {
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
			testProviderConfig(t, server.URL)
			fixture := "testdata/lifecycle/detectors/detector_actions.tf"
			initialFixture := "testdata/lifecycle/detectors/detector_actions_omitted.tf"
			if test.null {
				initialFixture = "testdata/lifecycle/detectors/detector_actions_null.tf"
			}
			if test.unknown {
				fixture = "testdata/lifecycle/detectors/detector_actions_deferred.tf"
				if test.null {
					initialFixture = "testdata/lifecycle/detectors/detector_actions_deferred_null.tf"
				}
			}

			check := func(action string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 || state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store replaced or tainted: creates=%d deletes=%d", creates, deletes)
					}
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.built_in_detectors.0.detectors.0.action", action), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.custom_detectors.0.detectors.0.action", action))(state)
				}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile: config.StaticFile(initialFixture),
					Check:      check("redact"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"action": config.StringVariable("mask")},
					Check:           check("mask"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"action": config.StringVariable("drop")},
					Check:           check("drop"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"action": config.StringVariable("redact")},
					Check:           check("redact"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"action": config.StringVariable("rejected-by-api")},
					ExpectError:     regexp.MustCompile("Failed to update Agent Memory service"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"action": config.StringVariable("redact")},
					Check:           check("redact"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"action": config.StringVariable("redact")},
					PlanOnly:        true,
				},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{"", "mask", "drop", "redact", "rejected-by-api"}, observed)
			assert.Equal(t, 1, creates)
			assert.Equal(t, 4, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryDetectorActionCreateRejectedByAPI(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/memory-stores", r.URL.Path)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		items := detectorActions(payload)
		assert.Len(t, items, 2)
		for _, item := range items {
			assert.Equal(t, "rejected-by-api", item["action"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"action rejected by API"}`))
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + detectorActionsConfig(`"rejected-by-api"`, "")
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestAgentMemoryDetectorEmptyAction(t *testing.T) {
	for _, kind := range []string{"built-in", "custom-detectors"} {
		for _, unknown := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unknown=%t", kind, unknown), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					items := listLimitEntries(payload, kind)
					assert.Len(t, items, 1)
					if len(items) == 1 {
						assert.Equal(t, "", items[0].(map[string]any)["action"])
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"message":"empty action rejected by API"}`))
				}))
				defer server.Close()
				defer func() { assert.Equal(t, int32(1), calls.Load()) }()
				dependency, expression := "", `""`
				if unknown {
					dependency = `resource "terraform_data" "action" { input="" }`
					expression = "tostring(terraform_data.action.output)"
				}
				config := reservedDetectorConfig(`"tenant_id"`, dependency, "email")
				if kind == "built-in" {
					config = strings.Replace(config, `action="redact"`, "action="+expression, 1)
				} else {
					index := strings.LastIndex(config, `action="redact"`)
					config = config[:index] + strings.Replace(config[index:], `action="redact"`, "action="+expression, 1)
				}
				prefix := testProviderConfig(t, server.URL)
				steps := []resource.TestStep{{Config: prefix + config, ExpectError: regexp.MustCompile(`empty\s+action\s+rejected\s+by\s+API`)}}
				if unknown {
					steps = []resource.TestStep{{Config: prefix + config, PlanOnly: true, ExpectNonEmptyPlan: true}, {Config: prefix + config, ExpectError: regexp.MustCompile(`empty\s+action\s+rejected\s+by\s+API`)}, {Config: prefix + reservedDetectorConfig(`"tenant_id"`, dependency, "email"), PlanOnly: true, ExpectNonEmptyPlan: true}}
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
			})
		}
	}
}
