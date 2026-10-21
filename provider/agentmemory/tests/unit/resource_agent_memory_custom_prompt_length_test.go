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

func TestAgentMemoryCustomPromptLimitsByAPI(t *testing.T) {
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
					var strategy map[string]any
					if r.Method == http.MethodPost {
						entries, _ := payload["customMemoryTypes"].([]any)
						if len(entries) != 1 {
							t.Error("missing custom type")
							w.WriteHeader(400)
							return
						}
						item, _ := entries[0].(map[string]any)
						assert.Equal(t, "preference", item["name"])
						strategy, _ = item["extractionStrategy"].(map[string]any)
					} else {
						assert.Empty(t, payload["addCustomMemoryTypes"])
						entries, _ := payload["updateCustomMemoryTypeStrategies"].([]any)
						if len(entries) != 1 {
							t.Error("missing strategy update")
							w.WriteHeader(400)
							return
						}
						strategy, _ = entries[0].(map[string]any)
						assert.Equal(t, "preference", strategy["typeName"])
					}
					prompt, ok := strategy["prompt"].(string)
					if !ok {
						t.Error("request omitted custom prompt")
						w.WriteHeader(400)
						return
					}
					observed = append(observed, prompt)
					if r.Method == http.MethodPost {
						creates++
					} else {
						patches++
					}
					// Simulate an expanded API limit while retaining server-side validation.
					if prompt == "rejected-by-api" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"custom extraction prompt rejected"}`))
						return
					}
					if store == nil {
						store = payload
					} else {
						entries, _ := store["customMemoryTypes"].([]any)
						item, _ := entries[0].(map[string]any)
						existing, _ := item["extractionStrategy"].(map[string]any)
						existing["prompt"] = prompt
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
			config := func(prompt string) string {
				expression, dependency := fmt.Sprintf("%q", prompt), ""
				if unknown {
					expression = "terraform_data.prompt.output"
					dependency = fmt.Sprintf(`resource "terraform_data" "prompt" { input=%q }`, prompt)
				}
				return testProviderConfig(t, server.URL) + customPromptConfig(expression, dependency)
			}
			check := func(prompt string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 {
						return fmt.Errorf("store replaced: creates=%d deletes=%d", creates, deletes)
					}
					if state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store tainted")
					}
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.0.extraction_strategy.prompt", prompt))(state)
				}
			}
			longPrompt := strings.Repeat("é", 10001)
			longerPrompt := strings.Repeat("界", 20000)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(longPrompt), Check: check(longPrompt)},
				{Config: config(longerPrompt), Check: check(longerPrompt)},
				{Config: config("rejected-by-api"), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: config(longerPrompt), Check: check(longerPrompt)},
				{Config: config(longerPrompt), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, []string{longPrompt, longerPrompt, "rejected-by-api"}, observed, "prompts must be transmitted unchanged")
			assert.Equal(t, 1, creates)
			assert.Equal(t, 2, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryCustomPromptCreateRejectedByAPI(t *testing.T) {
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
		entries, _ := payload["customMemoryTypes"].([]any)
		if len(entries) != 1 {
			t.Error("missing custom type")
			w.WriteHeader(400)
			return
		}
		item, _ := entries[0].(map[string]any)
		strategy, _ := item["extractionStrategy"].(map[string]any)
		assert.Equal(t, strings.Repeat("x", 10001), strategy["prompt"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"custom extraction prompt rejected"}`))
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + customPromptConfig(fmt.Sprintf("%q", strings.Repeat("x", 10001)), "")
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls, "rejected create must not produce a tracked store or delete call")
}

func customPromptConfig(expression, dependency string) string {
	return optionalStrategyConfig("extraction_strategy {\nprompt="+expression+"\nenabled=true\n}", dependency)
}

func TestAgentMemoryCustomPromptResolvedEmpty(t *testing.T) {
	prefix := testProviderConfig(t, "http://127.0.0.1:1")
	dependency := `resource "terraform_data" "flag" { input=true }`
	config := prefix + customPromptConfig(`terraform_data.flag.output ? "" : "Extract preferences"`, dependency)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
		{Config: config, ExpectError: regexp.MustCompile("length")},
		{Config: prefix + customPromptConfig(`"Extract preferences"`, dependency), PlanOnly: true, ExpectNonEmptyPlan: true},
	}})
}
