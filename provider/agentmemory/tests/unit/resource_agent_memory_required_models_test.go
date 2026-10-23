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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryRequiredModelFields(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		for _, field := range []string{"provider", "model", "type", "api_key", "api_key_version"} {
			for _, value := range []string{"omitted", "null"} {
				t.Run(block+"/"+field+"/"+value, func(t *testing.T) {
					var calls atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) }))
					defer server.Close()
					config := selectorConfig("llm", "provider", `"fake-provider"`, "")
					start := strings.Index(config, block+" {\n")
					if start < 0 {
						t.Fatalf("fixture missing %s block", block)
					}
					offset := strings.Index(config[start:], "\n}\n")
					if offset < 0 {
						t.Fatalf("fixture missing closing brace for %s", block)
					}
					end := offset + start + 1
					part := config[start:end]
					pattern := regexp.MustCompile(`(?m)^` + field + `=[^\n]*\n`)
					replacement := ""
					if value == "null" {
						replacement = field + "=null\n"
					}
					changed := pattern.ReplaceAllString(part, replacement)
					if changed == part {
						t.Fatalf("fixture did not alter %s.%s", block, field)
					}
					config = config[:start] + changed + config[end:]
					resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + config, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")}}})
					if calls.Load() != 0 {
						t.Fatalf("invalid configuration reached API: %d calls", calls.Load())
					}
				})
			}
		}
	}
}

func TestAgentMemoryExistingModelsWithoutCredentials(t *testing.T) {
	var mu sync.Mutex
	creates, patches, deletes := 0, 0, 0
	exists := false
	llmModel := "gpt-4o-mini"
	ttl := 31536000
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			creates++
			exists = true
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			llm, _ := payload["llm"].(map[string]any)
			memory, _ := payload["longTermMemory"].(map[string]any)
			embedding, _ := memory["embedding"].(map[string]any)
			expected := []string{"FAKE-LLM", "FAKE-EMBEDDING"}
			for i, model := range []map[string]any{llm, embedding} {
				credentials, _ := model["credentials"].(map[string]any)
				if credentials["apiKey"] != expected[i] {
					t.Error("create did not send the configured model key")
				}
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			patches++
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			memory, _ := payload["longTermMemory"].(map[string]any)
			llm, _ := payload["llm"].(map[string]any)
			embedding, _ := memory["embedding"].(map[string]any)
			// Omitting credentials must omit the JSON member, not send null or an empty key.
			for _, model := range []map[string]any{llm, embedding} {
				if _, present := model["credentials"]; present {
					t.Error("existing model update sent credentials")
				}
			}
			if llm != nil {
				llmModel = llm["model"].(string)
			}
			if v, ok := memory["ttlSeconds"].(float64); ok {
				ttl = int(v)
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
			_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && exists:
			response := httptest.NewRecorder()
			writeStoreResponseWithModelConfig(t, response, "required-model-store", llmModel, "text-embedding-3-small")
			var store map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &store); err != nil {
				t.Error(err)
			}
			store["longTermMemory"].(map[string]any)["ttlSeconds"] = ttl
			_ = json.NewEncoder(w).Encode(store)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			deletes++
			exists = false
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	prefix := testProviderConfig(t, server.URL)
	full := prefix + agentMemoryModelConfig("required-model-store", "gpt-4o-mini", "text-embedding-3-small", "FAKE-LLM", "FAKE-EMBEDDING")
	noCredentials := regexp.MustCompile(`credentials \{[^}]*\}`).ReplaceAllString(full, "")
	changedModel := strings.Replace(noCredentials, "gpt-4o-mini", "gpt-4.1-mini", 1)
	changedTTL := strings.Replace(changedModel, "31536000", "172800", 1)
	afterImport := strings.Replace(changedTTL, "gpt-4.1-mini", "gpt-4.1", 1)
	expectedDeletes := 0
	check := func(want int) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if creates != 1 || deletes != expectedDeletes || patches != want {
				return fmt.Errorf("unexpected lifecycle: creates=%d patches=%d deletes=%d", creates, patches, deletes)
			}
			return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")(state)
		}
	}
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: full, Check: check(0)},
		{Config: noCredentials, Check: check(1)},
		{Config: noCredentials, PlanOnly: true},
		{Config: changedModel, Check: check(2)},
		{Config: changedTTL, Check: check(3)},
	}})
	mu.Lock()
	if deletes != 1 || exists {
		t.Error("initial lifecycle did not clean up its store")
	}
	// Seed an existing store for import into an empty Terraform state.
	exists = true
	expectedDeletes = 1
	mu.Unlock()
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{

		{Config: changedTTL, ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateId: "store-1", ImportStatePersist: true},
		{Config: changedTTL, PlanOnly: true},
		{Config: afterImport, Check: check(4)},
		{Config: afterImport, PlanOnly: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || deletes != 2 {
		t.Fatalf("unexpected create/cleanup counts %d/%d", creates, deletes)
	}
}

// A deferred sibling must not hide a field that is already known to be absent.
func TestAgentMemoryRequiredModelFieldsWithUnknownSibling(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		for _, fields := range [][]string{{"provider", "model"}, {"type", "api_key", "api_key_version"}} {
			for _, missing := range fields {
				for _, unknown := range fields {
					if missing == unknown {
						continue
					}
					for _, absent := range []string{"omitted", "null"} {
						t.Run(block+"/"+missing+"/"+absent+"/unknown_"+unknown, func(t *testing.T) {
							var calls atomic.Int32
							server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) }))
							defer server.Close()
							config := selectorConfig("llm", "provider", `"fake-provider"`, "")
							start := strings.Index(config, block+" {\n")
							end := start + strings.Index(config[start:], "\n}\n") + 1
							part := config[start:end]
							replacement := ""
							if absent == "null" {
								replacement = missing + "=null\n"
							}
							part = regexp.MustCompile(`(?m)^`+missing+`=[^\n]*\n`).ReplaceAllString(part, replacement)
							expression := "tostring(terraform_data.sibling.output)"
							if unknown == "api_key_version" {
								expression = "tonumber(terraform_data.sibling.output)"
							}
							part = regexp.MustCompile(`(?m)^`+unknown+`=[^\n]*\n`).ReplaceAllString(part, unknown+"="+expression+"\n")
							config = `resource "terraform_data" "sibling" { input="1" }` + config[:start] + part + config[end:]
							resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + config, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")}}})
							if calls.Load() != 0 {
								t.Fatal("invalid configuration reached API")
							}
						})
					}
				}
			}
		}
	}
}
