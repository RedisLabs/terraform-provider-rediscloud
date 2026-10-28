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

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemorySemanticPromptLimitsByAPI(t *testing.T) {
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
					semantic, _ := policy["semantic"].(map[string]any)
					prompt, ok := semantic["prompt"].(string)
					if !ok {
						t.Error("request omitted semantic prompt")
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
						_, _ = w.Write([]byte(`{"message":"semantic prompt rejected"}`))
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
			testProviderConfig(t, server.URL)
			fixture := "testdata/lifecycle/exclusions/semantic_prompt"
			if unknown {
				fixture += "_deferred"
			}
			fixture += ".tf"
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
					return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.0.semantic.0.prompt", prompt))(state)
				}
			}
			longPrompt := strings.Repeat("é", 2001)
			longerPrompt := strings.Repeat("界", 3000)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"prompt": config.StringVariable(longPrompt)},
					Check:           check(longPrompt),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"prompt": config.StringVariable(longerPrompt)},
					Check:           check(longerPrompt),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"prompt": config.StringVariable("rejected-by-api")},
					ExpectError:     regexp.MustCompile("Failed to update Agent Memory service"),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"prompt": config.StringVariable(longerPrompt)},
					Check:           check(longerPrompt),
				},
				{
					ConfigFile:      config.StaticFile(fixture),
					ConfigVariables: config.Variables{"prompt": config.StringVariable(longerPrompt)},
					PlanOnly:        true,
				},
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

func TestAgentMemorySemanticPromptCreateRejectedByAPI(t *testing.T) {
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
		semantic, _ := policy["semantic"].(map[string]any)
		assert.Equal(t, strings.Repeat("x", 2001), semantic["prompt"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"semantic prompt rejected"}`))
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + semanticPromptConfig(fmt.Sprintf("%q", strings.Repeat("x", 2001)), "")
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls, "rejected create must not produce a tracked store or delete call")
}

func semanticPromptConfig(expression, dependency string) string {
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="semantic-prompt-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 long_term_memory_exclusions {
  enabled=true
  semantic {
   enabled=true
   prompt=%s
  }
 }
}
`, expression)
}
