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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func unknownKeyConfig(block, expression, dependency string) string {
	llmKey, embeddingKey := `"FAKE-STATIC"`, `"FAKE-STATIC"`
	if block == "llm" {
		llmKey = expression
	} else {
		embeddingKey = expression
	}
	config := dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="unknown-key-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 llm {
  provider="fake-provider"
  model="fake-llm"
  credentials {
   type="apiKey"
   api_key=%s
   api_key_version = 1
  }
 }
 embedding {
  provider="fake-provider"
  model="fake-embedding"
  credentials {
   type="apiKey"
   api_key=%s
   api_key_version = 1
  }
 }
}
`, llmKey, embeddingKey)
	// Keep an omitted attribute distinct from an explicitly configured null.
	return strings.ReplaceAll(config, "   api_key=OMIT_KEY\n", "")
}

func TestAgentMemoryModelAPIKeyValidation(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		for _, test := range []struct {
			name, expression, dependency string
			invalid                      bool
		}{
			{name: "known", expression: `"FAKE-KNOWN"`},
			{name: "unknown", expression: `tostring(terraform_data.key.output)`, dependency: `resource "terraform_data" "key" { input="FAKE-GENERATED" }`},
			{name: "omitted", expression: `OMIT_KEY`, invalid: true},
			{name: "null", expression: `null`, invalid: true},
			{name: "empty", expression: `""`, invalid: true},
			{name: "whitespace", expression: `"   "`, invalid: true},
		} {
			t.Run(block+"/"+test.name, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					http.Error(w, "no HTTP calls expected", 500)
				}))
				defer server.Close()
				defer func() {
					if requests.Load() != 0 {
						t.Errorf("plan validation made %d HTTP requests", requests.Load())
					}
				}()
				prefix := testProviderConfig(t, server.URL)
				step := resource.TestStep{Config: prefix + unknownKeyConfig(block, test.expression, test.dependency), PlanOnly: true, ExpectNonEmptyPlan: true}
				if test.invalid {
					step.ExpectError = regexp.MustCompile("Missing Agent Memory (model|write-only) API key|Invalid Attribute Combination")
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
			})
		}
	}
}

func TestAgentMemoryModelAPIKeyUnknownResolvesInvalid(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		for _, test := range []struct{ name, input, expression string }{
			// A null input is already known during plan; this conditional resolves to null only at apply.
			{"null", "true", `terraform_data.key.output ? null : "FAKE-ONLY"`},
			{"empty", `""`, `tostring(terraform_data.key.output)`},
			{"whitespace", `"   "`, `tostring(terraform_data.key.output)`},
		} {
			t.Run(block+"/"+test.name, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					http.Error(w, "invalid resolved key must not reach HTTP", 500)
				}))
				defer server.Close()
				defer func() {
					if requests.Load() != 0 {
						t.Errorf("invalid key reached HTTP %d times", requests.Load())
					}
				}()
				prefix := testProviderConfig(t, server.URL)
				config := prefix + unknownKeyConfig(block, test.expression, fmt.Sprintf(`resource "terraform_data" "key" { input=%s }`, test.input))
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
					{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config, ExpectError: regexp.MustCompile("Missing Agent Memory (model|write-only) API key|Invalid Attribute Combination")},
				}})
			})
		}
	}
}

func TestAgentMemoryModelAPIKeyUnknownCreateAndUpdate(t *testing.T) {
	for _, block := range []string{"llm", "embedding"} {
		t.Run(block, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			var creates, updates, deletes int
			var observed []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case (r.Method == http.MethodPost && r.URL.Path == "/memory-stores") || (r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1"):
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					var model map[string]any
					if block == "llm" {
						model, _ = payload["llm"].(map[string]any)
					} else if memory, ok := payload["longTermMemory"].(map[string]any); ok {
						model, _ = memory["embedding"].(map[string]any)
					}
					credentials, _ := model["credentials"].(map[string]any)
					key, _ := credentials["apiKey"].(string)
					if strings.TrimSpace(key) == "" {
						t.Error("unknown/invalid key sent to API")
						w.WriteHeader(400)
						return
					}
					observed = append(observed, key)
					// The actual API does not return model secrets in GET responses.
					if llm, ok := payload["llm"].(map[string]any); ok {
						delete(llm, "credentials")
					}
					if memory, ok := payload["longTermMemory"].(map[string]any); ok {
						if embedding, ok := memory["embedding"].(map[string]any); ok {
							delete(embedding, "credentials")
						}
					}
					if r.Method == http.MethodPost {
						creates++
						store = payload
					} else {
						updates++
						for k, v := range payload {
							store[k] = v
						}
					}
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
					if store == nil {
						http.NotFound(w, r)
						return
					}
					store["storeId"] = "store-1"
					store["databaseId"] = "123"
					store["status"] = "READY"
					store["extractionStrategy"] = "INSTRUCT"
					store["createdAt"] = "2026-10-16T12:00:00Z"
					store["endpoint"] = "https://memory.example.test"
					store["endpoints"] = []map[string]any{{"url": "https://memory.example.test", "provider": "AWS", "region": "us-east-1", "egressIps": []string{}, "isAccessible": true}}
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
			defer func() {
				mu.Lock()
				defer mu.Unlock()
				if store != nil {
					t.Error("local fixture not removed")
				}
				if creates != 1 || updates != 1 || deletes != 1 {
					t.Errorf("expected one local create/update/delete; got %d/%d/%d", creates, updates, deletes)
				}
				if len(observed) != 2 || observed[0] != "FAKE-FIRST" || observed[1] != "FAKE-ROTATED" {
					t.Error("API did not receive the resolved keys in order")
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			config := func(value string) string {
				cfg := prefix + unknownKeyConfig(block, `tostring(terraform_data.key.output)`, fmt.Sprintf(`resource "terraform_data" "key" { input=%q }`, value))
				if value == "FAKE-ROTATED" {
					cfg = strings.ReplaceAll(cfg, "api_key_version = 1", "api_key_version = 2")
				}
				return cfg
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config("FAKE-FIRST"), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", block+".credentials.api_key"))},
				{Config: config("FAKE-ROTATED"), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", block+".credentials.api_key"))},
				{Config: config("FAKE-ROTATED"), PlanOnly: true},
			}})
		})
	}
}
