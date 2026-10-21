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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemorySummarizationStrategyLifecycle(t *testing.T) {
	for _, tc := range []struct{ name, expression, dependency, expected string }{
		{name: "omitted", expected: "FAKE-API-DEFAULT"},
		{name: "null", expression: "null", expected: "FAKE-API-DEFAULT"},
		{name: "current", expression: `"event_count"`, expected: "event_count"},
		{name: "future", expression: `"FAKE-FUTURE"`, expected: "FAKE-FUTURE"},
		{name: "unknown", expression: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="FAKE-RESOLVED" }`, expected: "FAKE-RESOLVED"},
		{name: "unknown null", expression: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input=null }`, expected: "FAKE-API-DEFAULT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			creates, patches, deletes := 0, 0, 0
			var store map[string]any
			var firstStrategy any
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
					summarization, _ := payload["summarization"].(map[string]any)
					if r.Method == http.MethodPost {
						creates++
						firstStrategy = summarization["triggerStrategy"]
					} else {
						patches++
					}
					if summarization["triggerStrategy"] == "FAKE-REJECTED" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"unsupported summarization strategy"}`))
						return
					}
					if store == nil {
						response := httptest.NewRecorder()
						writeStoreResponse(t, response, "store", 86400, 31536000, 300)
						if err := json.Unmarshal(response.Body.Bytes(), &store); err != nil {
							t.Error(err)
							w.WriteHeader(500)
							return
						}
					}
					if summarization["enabled"] == true && summarization["triggerStrategy"] == nil {
						summarization["triggerStrategy"] = "FAKE-API-DEFAULT"
					}
					if summarization["enabled"] == false {
						delete(summarization, "triggerStrategy")
					}
					store["summarization"] = summarization
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && store != nil:
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
			config := func(expression string, enabled bool) string {
				attr := ""
				if expression != "" {
					attr = "trigger_strategy=" + expression
				}
				return testProviderConfig(t, server.URL) + tc.dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 summarization {
  enabled=%t
  %s
 }
}
`, enabled, attr)
			}
			check := func(strategy string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 {
						return fmt.Errorf("store replaced: creates=%d deletes=%d", creates, deletes)
					}
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.trigger_strategy", strategy))(state)
				}
			}
			initial := config(tc.expression, true)
			updated := config(`"FAKE-FUTURE-UPDATE"`, true)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: initial, Check: check(tc.expected)},
				{Config: initial, PlanOnly: true},
				{Config: updated, Check: check("FAKE-FUTURE-UPDATE")},
				{Config: config(`"FAKE-REJECTED"`, true), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: updated, PlanOnly: true},
				{Config: config("", true), PlanOnly: true},
				{Config: config("", false), Check: resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "summarization.trigger_strategy")},
				{Config: config("", true), Check: check("FAKE-API-DEFAULT")},
				{Config: config("", true), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			if tc.expected == "FAKE-API-DEFAULT" {
				assert.Nil(t, firstStrategy, "provider must not inject a default into the request")
			} else {
				assert.Equal(t, tc.expected, firstStrategy)
			}
			assert.Equal(t, 1, creates)
			assert.Equal(t, 4, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemorySummarizationStrategyErrors(t *testing.T) {
	for _, tc := range []struct {
		name, expression, dependency, pattern string
		calls                                 int
	}{
		{name: "empty", expression: `""`, pattern: "at least 1"},
		{name: "resolved empty", expression: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="" }`, pattern: "at least 1"},
		{name: "API rejection", expression: `"FAKE-REJECTED"`, pattern: "Failed to create Agent Memory service", calls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/memory-stores" {
					t.Error("unexpected API operation")
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				summarization, _ := payload["summarization"].(map[string]any)
				assert.Equal(t, "FAKE-REJECTED", summarization["triggerStrategy"])
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"message":"unsupported summarization strategy"}`))
			}))
			defer server.Close()
			config := testProviderConfig(t, server.URL) + requiredEventCountConfig("trigger_strategy="+tc.expression, tc.dependency)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile(tc.pattern)}}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tc.calls, calls)
		})
	}
}
