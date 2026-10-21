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

var selectorCases = []struct{ block, field, value string }{
	{"llm", "provider", "fake-provider"},
	{"llm", "model", "fake-llm"},
	{"embedding", "provider", "fake-provider"},
	{"embedding", "model", "fake-embedding"},
}

func selectorConfig(block, field, expression, dependency string) string {
	models := ""
	for _, name := range []string{"llm", "embedding"} {
		model := "fake-llm"
		if name == "embedding" {
			model = "fake-embedding"
		}
		providerExpr, modelExpr := `"fake-provider"`, fmt.Sprintf("%q", model)
		if name == block {
			if field == "provider" {
				providerExpr = expression
			} else {
				modelExpr = expression
			}
		}
		models += fmt.Sprintf("%s {\nprovider=%s\nmodel=%s\ncredentials {\ntype=\"apiKey\"\napi_key=\"FAKE-STATIC\"\napi_key_version=1\n}\n}\n", name, providerExpr, modelExpr)
	}
	models = strings.ReplaceAll(models, "provider=OMIT_SELECTOR\n", "")
	models = strings.ReplaceAll(models, "model=OMIT_SELECTOR\n", "")
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="selector-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 %s
}
`, models)
}

func TestAgentMemoryUnknownSelectorValidation(t *testing.T) {
	for _, selector := range selectorCases {
		for _, c := range []struct {
			name, expression, dependency string
			invalid                      bool
		}{
			{name: "known", expression: fmt.Sprintf("%q", selector.value)},
			{name: "unknown", expression: "tostring(terraform_data.selector.output)", dependency: fmt.Sprintf(`resource "terraform_data" "selector" { input=%q }`, selector.value)},
			{name: "omitted", expression: "OMIT_SELECTOR", invalid: true},
			{name: "null", expression: "null", invalid: true},
			{name: "empty", expression: `""`, invalid: true},
			{name: "whitespace", expression: `"   "`, invalid: true},
		} {
			t.Run(selector.block+"/"+selector.field+"/"+c.name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					http.Error(w, "validation must not call HTTP", 500)
				}))
				defer server.Close()
				defer func() {
					if calls.Load() != 0 {
						t.Error("plan reached HTTP")
					}
				}()
				config := testProviderConfig(t, server.URL) + selectorConfig(selector.block, selector.field, c.expression, c.dependency)
				step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
				if c.invalid {
					step.ExpectError = regexp.MustCompile("Missing Agent Memory model")
					if c.name == "omitted" || c.name == "null" {
						step.ExpectError = regexp.MustCompile("Invalid Attribute Combination")
					}
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
			})
		}
	}
}

func TestAgentMemoryUnknownSelectorResolvesInvalid(t *testing.T) {
	for _, selector := range selectorCases {
		for _, c := range []struct{ name, input, expression string }{
			{"null", "true", `terraform_data.selector.output ? null : "fake"`},
			{"empty", `""`, `tostring(terraform_data.selector.output)`},
			{"whitespace", `"   "`, `tostring(terraform_data.selector.output)`},
		} {
			t.Run(selector.block+"/"+selector.field+"/"+c.name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					http.Error(w, "invalid selector must not call HTTP", 500)
				}))
				defer server.Close()
				defer func() {
					if calls.Load() != 0 {
						t.Error("invalid selector reached HTTP")
					}
				}()
				config := testProviderConfig(t, server.URL) + selectorConfig(selector.block, selector.field, c.expression, fmt.Sprintf(`resource "terraform_data" "selector" { input=%s }`, c.input))
				pattern := "Missing Agent Memory model"
				if c.name == "null" {
					pattern = "Invalid Attribute Combination"
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
					{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config, ExpectError: regexp.MustCompile(pattern)},
				}})
			})
		}
	}
}

func TestAgentMemoryUnknownSelectorLifecycle(t *testing.T) {
	for _, selector := range selectorCases {
		t.Run(selector.block+"/"+selector.field, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, deletes := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					if err := json.NewDecoder(r.Body).Decode(&store); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					creates++
					var model map[string]any
					if selector.block == "llm" {
						model, _ = store["llm"].(map[string]any)
					} else {
						memory, _ := store["longTermMemory"].(map[string]any)
						model, _ = memory["embedding"].(map[string]any)
					}
					if model[selector.field] != selector.value {
						t.Error("resolved selector missing from create request")
					}
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					if len(payload) != 0 {
						t.Errorf("unchanged selectors generated API changes: %v", payload)
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
				expected := 1
				if selector.field == "provider" || selector.block == "embedding" {
					expected = 2
				}
				if creates != expected || deletes != expected || store != nil {
					t.Errorf("fixture not cleaned up: creates=%d deletes=%d remaining=%t", creates, deletes, store != nil)
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			config := func(suffix string) string {
				return prefix + selectorConfig(selector.block, selector.field, `split(":",tostring(terraform_data.selector.output))[0]`, fmt.Sprintf(`resource "terraform_data" "selector" { input=%q }`, selector.value+":"+suffix))
			}
			first, changed := config("before"), config("after")
			check := resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", selector.block+"."+selector.field, selector.value))
			// Configured model blocks require explicit selectors on updates as well.
			omitted := prefix + selectorConfig(selector.block, selector.field, "OMIT_SELECTOR", fmt.Sprintf(`resource "terraform_data" "selector" { input=%q }`, selector.value+":after"))
			steps := []resource.TestStep{
				{Config: first, Check: check},
				{Config: changed, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: changed, Check: check},
				{Config: changed, PlanOnly: true},
				{Config: omitted, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
				{Config: changed, PlanOnly: true},
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
		})
	}
}
