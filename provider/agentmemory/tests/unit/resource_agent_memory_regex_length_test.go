package unit_test

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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryRegexPatternLimitsByAPI(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown=%t", unknown), func(t *testing.T) {
			var mu sync.Mutex
			creates, patches, deletes := 0, 0, 0
			var store map[string]any
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
					policy, _ := payload["longTermMemoryExclusions"].(map[string]any)
					custom, _ := policy["customDetectors"].(map[string]any)
					entries, _ := custom["detectors"].([]any)
					if len(entries) != 1 {
						t.Error("missing custom detector")
						w.WriteHeader(400)
						return
					}
					detector, _ := entries[0].(map[string]any)
					matcher, _ := detector["matcher"].(map[string]any)
					regex, _ := matcher["regex"].(map[string]any)
					pattern, ok := regex["pattern"].(string)
					if !ok {
						t.Error("request omitted regex pattern")
						w.WriteHeader(400)
						return
					}
					observed = append(observed, pattern)
					if r.Method == http.MethodPost {
						creates++
					} else {
						patches++
					}
					// Simulate an expanded API limit while retaining server-side validation.
					if pattern == "rejected-by-api" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"regex pattern rejected"}`))
						return
					}
					if store == nil {
						store = payload
					} else {
						store["longTermMemoryExclusions"] = policy
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
			config := func(pattern string) string {
				expression, dependency := fmt.Sprintf("%q", pattern), ""
				if unknown {
					expression = "terraform_data.pattern.output"
					dependency = fmt.Sprintf(`resource "terraform_data" "pattern" { input=%q }`, pattern)
				}
				return testProviderConfig(t, server.URL) + emptyRegexConfig(expression, dependency, true)
			}
			check := func(pattern string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 {
						return fmt.Errorf("store replaced: creates=%d deletes=%d", creates, deletes)
					}
					if state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store tainted")
					}
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.detectors.0.matcher.regex.pattern", pattern))(state)
				}
			}
			longPattern := strings.Repeat("é", 513)
			longerPattern := strings.Repeat("界", 1024)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(longPattern), Check: check(longPattern)},
				{Config: config(longerPattern), Check: check(longerPattern)},
				{Config: config("rejected-by-api"), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: config(longerPattern), Check: check(longerPattern)},
				{Config: config(longerPattern), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{longPattern, longerPattern, "rejected-by-api"}, observed, "patterns must be transmitted unchanged")
			assert.Equal(t, 1, creates)
			assert.Equal(t, 2, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryRegexPatternCreateRejectedByAPI(t *testing.T) {
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
		policy, _ := payload["longTermMemoryExclusions"].(map[string]any)
		custom, _ := policy["customDetectors"].(map[string]any)
		entries, _ := custom["detectors"].([]any)
		if len(entries) != 1 {
			t.Error("missing custom detector")
			w.WriteHeader(400)
			return
		}
		detector, _ := entries[0].(map[string]any)
		matcher, _ := detector["matcher"].(map[string]any)
		regex, _ := matcher["regex"].(map[string]any)
		assert.Equal(t, strings.Repeat("x", 513), regex["pattern"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"regex pattern rejected"}`))
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + emptyRegexConfig(fmt.Sprintf("%q", strings.Repeat("x", 513)), "", true)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls, "rejected create must not produce a tracked store or delete call")
}
