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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryEventCountLimitsByAPI(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown=%t", unknown), func(t *testing.T) {
			var mu sync.Mutex
			creates, patches, deletes := 0, 0, 0
			var store map[string]any
			var observed [][2]int
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
					policy, _ := payload["summarization"].(map[string]any)
					counts, _ := policy["eventCount"].(map[string]any)
					threshold, hasThreshold := counts["threshold"].(float64)
					retain, hasRetain := counts["retainCount"].(float64)
					if !hasThreshold || !hasRetain {
						t.Error("request omitted an event-count field")
						w.WriteHeader(400)
						return
					}
					observed = append(observed, [2]int{int(threshold), int(retain)})
					if r.Method == http.MethodPost {
						creates++
					} else {
						patches++
					}
					// Simulate an expanded upper limit; the API still rejects negative counts.
					if retain < 0 {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"retainCount must not be negative"}`))
						return
					}
					if store == nil {
						store = payload
					} else {
						store["summarization"] = policy
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
			fixture := "testdata/lifecycle/summarization/summarization_event_count_limits"
			if unknown {
				fixture += "_deferred"
			}
			fixture += ".tf"
			check := func(threshold, retain int) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 {
						return fmt.Errorf("store replaced: creates=%d deletes=%d", creates, deletes)
					}
					if state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store tainted")
					}
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.event_count.0.threshold", fmt.Sprint(threshold)), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.event_count.0.retain_count", fmt.Sprint(retain)))(state)
				}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile: config.StaticFile(fixture),
					ConfigVariables: config.Variables{
						"threshold": config.IntegerVariable(10001),
						"retain":    config.IntegerVariable(10000),
					},
					Check: check(10001, 10000),
				},
				{
					ConfigFile: config.StaticFile(fixture),
					ConfigVariables: config.Variables{
						"threshold": config.IntegerVariable(20001),
						"retain":    config.IntegerVariable(15001),
					},
					Check: check(20001, 15001),
				},
				{
					ConfigFile: config.StaticFile(fixture),
					ConfigVariables: config.Variables{
						"threshold": config.IntegerVariable(1),
						"retain":    config.IntegerVariable(0),
					},
					Check: check(1, 0),
				},
				{
					ConfigFile: config.StaticFile(fixture),
					ConfigVariables: config.Variables{
						"threshold": config.IntegerVariable(20),
						"retain":    config.IntegerVariable(-1),
					},
					ExpectError: regexp.MustCompile("Failed to update Agent Memory service"),
				},
				{
					ConfigFile: config.StaticFile(fixture),
					ConfigVariables: config.Variables{
						"threshold": config.IntegerVariable(1),
						"retain":    config.IntegerVariable(0),
					},
					Check: check(1, 0),
				},
				{
					ConfigFile: config.StaticFile(fixture),
					ConfigVariables: config.Variables{
						"threshold": config.IntegerVariable(1),
						"retain":    config.IntegerVariable(0),
					},
					PlanOnly: true,
				},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, [][2]int{{10001, 10000}, {20001, 15001}, {1, 0}, {20, -1}}, observed, "request values must be transmitted unchanged")
			assert.Equal(t, 1, creates)
			assert.Equal(t, 3, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryEventCountCreateRejectedByAPI(t *testing.T) {
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
		policy, _ := payload["summarization"].(map[string]any)
		counts, _ := policy["eventCount"].(map[string]any)
		assert.Equal(t, float64(20), counts["threshold"])
		assert.Equal(t, float64(-1), counts["retainCount"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"retainCount must not be negative"}`))
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + requiredEventCountConfig("event_count {\nthreshold=20\nretain_count=-1\n}", "")
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls, "rejected create must not produce a tracked store or delete call")
}
