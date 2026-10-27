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

var credentialTypeCases = []struct{ block, field, value string }{
	{"llm", "type", "apiKey"},
	{"embedding", "type", "apiKey"},
}

func credentialTypeConfig(block, field, expression, dependency string) string {
	models := ""
	for _, name := range []string{"llm", "embedding"} {
		typeExpression := `"apiKey"`
		if name == block {
			typeExpression = expression
		}
		models += fmt.Sprintf("%s {\nprovider=\"fake-provider\"\nmodel=\"fake-model\"\ncredentials {\ntype=%s\napi_key=\"FAKE-STATIC\"\napi_key_version=1\n}\n}\n", name, typeExpression)
	}
	models = strings.ReplaceAll(models, "type=OMIT_SELECTOR\n", "")
	return dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="credential-type-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 %s
}
`, models)
}

func TestAgentMemoryUnknownCredentialTypeValidation(t *testing.T) {
	for _, selector := range credentialTypeCases {
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
			{name: "API-owned value", expression: `"unsupported"`},
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
				config := testProviderConfig(t, server.URL) + credentialTypeConfig(selector.block, selector.field, c.expression, c.dependency)
				step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
				if c.invalid {
					step.ExpectError = regexp.MustCompile("Missing Agent Memory model credential type")
					if c.name == "omitted" || c.name == "null" {
						step.ExpectError = regexp.MustCompile("Invalid Attribute Combination")
					}
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
			})
		}
	}
}

func TestAgentMemoryUnknownCredentialTypeResolvesInvalid(t *testing.T) {
	for _, selector := range credentialTypeCases {
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
				config := testProviderConfig(t, server.URL) + credentialTypeConfig(selector.block, selector.field, c.expression, fmt.Sprintf(`resource "terraform_data" "selector" { input=%s }`, c.input))
				pattern := "Missing Agent Memory model credential type"
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

func TestAgentMemoryUnknownCredentialTypeLifecycle(t *testing.T) {
	selectors := append([]struct{ block, field, value string }{}, credentialTypeCases...)
	selectors = append(selectors,
		struct{ block, field, value string }{"llm", "type", "FAKE-KEY-TYPE"},
		struct{ block, field, value string }{"embedding", "type", "FAKE-KEY-TYPE"},
	)
	for _, selector := range selectors {
		t.Run(selector.block+"/"+selector.field+"/"+selector.value, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, deletes, rejectedPatches := 0, 0, 0
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
					credentials, _ := model["credentials"].(map[string]any)
					if credentials[selector.field] != selector.value {
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
						var model map[string]any
						if selector.block == "llm" {
							model, _ = payload["llm"].(map[string]any)
						} else {
							memory, _ := payload["longTermMemory"].(map[string]any)
							model, _ = memory["embedding"].(map[string]any)
						}
						credentials, _ := model["credentials"].(map[string]any)
						if credentials["type"] != "unsupported" || credentials["apiKey"] != "FAKE-STATIC" {
							t.Error("changed credential type was not preserved in update request")
						}
						rejectedPatches++
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"message":"Credential type rejected"}`))
						return
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
				if creates != 1 || deletes != 1 || rejectedPatches != 1 || store != nil {
					t.Errorf("fixture not cleaned up: creates=%d deletes=%d rejectedPatches=%d remaining=%t", creates, deletes, rejectedPatches, store != nil)
				}
			}()
			prefix := testProviderConfig(t, server.URL)
			config := func(suffix string) string {
				return prefix + credentialTypeConfig(selector.block, selector.field, `split(":",tostring(terraform_data.selector.output))[0]`, fmt.Sprintf(`resource "terraform_data" "selector" { input=%q }`, selector.value+":"+suffix))
			}
			first, changed := config("before"), config("after")
			check := resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", selector.block+".0.credentials.0."+selector.field, selector.value))
			// A configured credentials block requires its type on updates as well.
			omitted := prefix + credentialTypeConfig(selector.block, selector.field, "OMIT_SELECTOR", fmt.Sprintf(`resource "terraform_data" "selector" { input=%q }`, selector.value+":after"))
			steps := []resource.TestStep{
				{Config: first, Check: check},
				{Config: changed, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: changed, Check: check},
				{Config: changed, PlanOnly: true},
				{Config: omitted, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid Attribute Combination")},
				{Config: changed, PlanOnly: true},
			}
			invalid := prefix + credentialTypeConfig(selector.block, selector.field,
				`tostring(terraform_data.selector.output)`, `resource "terraform_data" "selector" { input="unsupported" }`)
			steps = append(steps,
				resource.TestStep{Config: invalid, PlanOnly: true, ExpectNonEmptyPlan: true},
				resource.TestStep{Config: invalid, ExpectError: regexp.MustCompile(`(?s)Failed to update Agent Memory service.*Credential\s+type\s+rejected`)})
			// Restore the accepted type after the API rejects the update.
			restored := prefix + credentialTypeConfig(selector.block, selector.field,
				fmt.Sprintf("%q", selector.value), `resource "terraform_data" "selector" { input="unsupported" }`)
			steps = append(steps, resource.TestStep{Config: restored, PlanOnly: true})
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
		})
	}
}
