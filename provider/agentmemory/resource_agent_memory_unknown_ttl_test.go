package agentmemory_test

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

func unknownTTLConfig(expression, dependency string, customerModels bool) string {
	models := ""
	if customerModels {
		for _, block := range []string{"llm", "embedding"} {
			models += fmt.Sprintf("%s {\nprovider=\"fake-provider\"\nmodel=\"fake-model\"\ncredentials {\ntype=\"apiKey\"\napi_key=\"FAKE-STATIC\"\napi_key_version=1\n}\n}\n", block)
		}
	}
	config := dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="unknown-ttl-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=%s
 extraction_cadence_seconds=300
 %s
}
`, expression, models)
	return strings.ReplaceAll(config, " long_term_ttl_seconds=OMIT_TTL\n", "")
}

func TestAgentMemoryUnknownLongTermTTLValidation(t *testing.T) {
	for _, customerModels := range []bool{false, true} {
		mode := "platform"
		if customerModels {
			mode = "customer"
		}
		for _, c := range []struct{ name, expression, dependency, errorPattern string }{
			{name: "known", expression: "86400"},
			{name: "previous minimum", expression: "1"},
			{name: "previous maximum", expression: "31536000"},
			{name: "unknown", expression: "terraform_data.ttl.output", dependency: `resource "terraform_data" "ttl" { input=86400 }`},
			{name: "omitted", expression: "OMIT_TTL"},
			{name: "null", expression: "null"},
			{name: "zero", expression: "0"},
			{name: "negative", expression: "-1"},
			{name: "oversized", expression: "31536001"},
			{name: "fraction", expression: "1.5", errorPattern: "integer|Int64|whole"},
		} {
			t.Run(mode+"/"+c.name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					http.Error(w, "plan must not call HTTP", 500)
				}))
				defer server.Close()
				defer func() {
					if calls.Load() != 0 {
						t.Error("plan reached HTTP")
					}
				}()
				config := testProviderConfig(t, server.URL) + unknownTTLConfig(c.expression, c.dependency, customerModels)
				pattern := c.errorPattern
				if customerModels && (c.name == "omitted" || c.name == "null") {
					pattern = "Missing Agent Memory long-term TTL"
				}
				step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
				if pattern != "" {
					step.ExpectError = regexp.MustCompile(pattern)
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
			})
		}
	}
}

func TestAgentMemoryUnknownLongTermTTLResolvesInvalid(t *testing.T) {
	for _, c := range []struct{ name, input, expression, errorPattern string }{
		{"null", "true", "terraform_data.ttl.output ? null : 86400", "Missing Agent Memory long-term TTL"},
		{"zero", "0", "terraform_data.ttl.output", "Failed to create Agent Memory service"},
		{"negative", "-1", "terraform_data.ttl.output", "Failed to create Agent Memory service"},
		{"oversized", "31536001", "terraform_data.ttl.output", "Failed to create Agent Memory service"},
		{"fraction", "1.5", "terraform_data.ttl.output", "integer|Int64|whole"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/memory-stores" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var payload struct {
					LongTermMemory struct {
						TTLSeconds *int `json:"ttlSeconds"`
					} `json:"longTermMemory"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Error(err)
				}
				if payload.LongTermMemory.TTLSeconds == nil {
					t.Error("resolved TTL missing from request")
				} else if fmt.Sprint(*payload.LongTermMemory.TTLSeconds) != c.input {
					t.Errorf("resolved TTL changed: got %d, want %s", *payload.LongTermMemory.TTLSeconds, c.input)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"TTL rejected by the API"}`))
			}))
			defer server.Close()
			defer func() {
				expectedCalls := int32(0)
				if c.errorPattern == "Failed to create Agent Memory service" {
					expectedCalls = 1
				}
				if calls.Load() != expectedCalls {
					t.Errorf("unexpected API calls: got %d, want %d", calls.Load(), expectedCalls)
				}
			}()
			config := testProviderConfig(t, server.URL) + unknownTTLConfig(c.expression, fmt.Sprintf(`resource "terraform_data" "ttl" { input=%s }`, c.input), true)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config, ExpectError: regexp.MustCompile(c.errorPattern)},
			}})
		})
	}
}

func TestAgentMemoryUnknownLongTermTTLLifecycle(t *testing.T) {
	var mu sync.Mutex
	var store map[string]any
	creates, patches, deletes, rejectedPatches := 0, 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores" || r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			memory, ok := payload["longTermMemory"].(map[string]any)
			if !ok {
				t.Error("longTermMemory missing")
				w.WriteHeader(400)
				return
			}
			ttl, ok := memory["ttlSeconds"].(float64)
			if ok && r.Method == http.MethodPatch && ttl == 0 {
				rejectedPatches++
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"TTL rejected by the API"}`))
				return
			}
			expected := float64(86400)
			if r.Method == http.MethodPatch {
				expected = 172800
			}
			if !ok || ttl != expected {
				t.Errorf("resolved TTL not preserved: %v", memory)
				w.WriteHeader(400)
				return
			}
			if r.Method == http.MethodPost {
				creates++
				store = payload
			} else {
				patches++
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
		if creates != 1 || patches != 1 || deletes != 1 || rejectedPatches != 1 || store != nil {
			t.Errorf("unexpected fixture counts: creates=%d patches=%d deletes=%d rejectedPatches=%d remaining=%t", creates, patches, deletes, rejectedPatches, store != nil)
		}
	}()
	prefix := testProviderConfig(t, server.URL)
	config := func(ttl int) string {
		return prefix + unknownTTLConfig("terraform_data.ttl.output", fmt.Sprintf(`resource "terraform_data" "ttl" { input=%d }`, ttl), true)
	}
	first, changed := config(86400), config(172800)
	// On update, omission must retain the known TTL rather than become a missing-create error.
	omitted := prefix + unknownTTLConfig("OMIT_TTL", `resource "terraform_data" "ttl" { input=172800 }`, true)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: first, Check: resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_ttl_seconds", "86400")},
		{Config: changed, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_ttl_seconds", "172800"))},
		{Config: changed, PlanOnly: true},
		{Config: omitted, PlanOnly: true},
		{Config: changed, PlanOnly: true},
		{Config: config(0), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
	}})
}
