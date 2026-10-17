package agentmemory_test

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

func descriptionLengthConfig(descriptions []string, unknown bool) string {
	config := listLimitConfig("custom-types", len(descriptions), false)
	var values []string
	for _, description := range descriptions {
		values = append(values, fmt.Sprintf("%q", description))
	}
	for i, value := range values {
		expression := value
		if unknown {
			expression = fmt.Sprintf("terraform_data.descriptions.output[%d]", i)
		}
		config = strings.Replace(config, `description="Custom memory"`, "description="+expression, 1)
		config = strings.Replace(config, `description="Topic"`, "description="+expression, 1)
	}
	if unknown {
		config = fmt.Sprintf("resource \"terraform_data\" \"descriptions\" { input=[%s] }\n", strings.Join(values, ",")) + config
	}
	return config
}

func TestAgentMemoryDescriptionLengthByAPI(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown=%t", unknown), func(t *testing.T) {
			descriptions := []string{strings.Repeat("é", 201), strings.Repeat("界", 1024), "rejected-by-api"}
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
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
					entries := listLimitEntries(payload, "custom-types")
					if r.Method == http.MethodPost {
						creates++
					} else {
						patches++
						entries, _ = payload["addCustomMemoryTypes"].([]any)
						assert.Empty(t, payload["updateCustomMemoryTypeStrategies"])
					}
					if len(entries) != 1 {
						t.Error("expected exactly one new type")
						w.WriteHeader(400)
						return
					}
					item, _ := entries[0].(map[string]any)
					description, _ := item["description"].(string)
					observed = append(observed, description)
					assert.Equal(t, fmt.Sprintf("custom_%d", len(observed)-1), item["name"])
					fields, _ := item["fields"].([]any)
					if len(fields) != 1 {
						t.Error("missing field")
						w.WriteHeader(400)
						return
					}
					field, _ := fields[0].(map[string]any)
					assert.Equal(t, description, field["description"], "type and field descriptions must be transmitted verbatim")
					// Simulate expanded API limits while retaining server-side rejection.
					if description == "rejected-by-api" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"description rejected by API"}`))
						return
					}
					if store == nil {
						store = payload
					} else {
						existing := listLimitEntries(store, "custom-types")
						store["customMemoryTypes"] = append(existing, entries...)
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
			prefix := testProviderConfig(t, server.URL)
			config := func(count int) string { return prefix + descriptionLengthConfig(descriptions[:count], unknown) }
			check := func(count int) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 || state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store replaced or tainted: creates=%d deletes=%d", creates, deletes)
					}
					checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.#", fmt.Sprint(count))}
					for i := 0; i < count; i++ {
						checks = append(checks, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("custom_memory_types.%d.description", i), descriptions[i]), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("custom_memory_types.%d.fields.0.description", i), descriptions[i]))
					}
					return resource.ComposeTestCheckFunc(checks...)(state)
				}
			}
			redefined := prefix + descriptionLengthConfig([]string{"changed definition", descriptions[1]}, unknown)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(1), Check: check(1)},
				{Config: config(2), Check: check(2)},
				{Config: config(3), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: config(2), Check: check(2)},
				{Config: redefined, ExpectError: regexp.MustCompile("Unsupported Agent Memory custom memory type redefinition")},
				{Config: config(1), ExpectError: regexp.MustCompile("Unsupported Agent Memory custom memory type removal")},
				{Config: config(2), Check: check(2)},
				{Config: config(2), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, descriptions, observed)
			assert.Equal(t, 1, creates)
			assert.Equal(t, 2, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryDescriptionCreateRejectedByAPI(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	description := strings.Repeat("x", 201)
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
		entries := listLimitEntries(payload, "custom-types")
		if len(entries) != 1 {
			t.Error("missing type")
		} else {
			item, _ := entries[0].(map[string]any)
			assert.Equal(t, description, item["description"])
			fields, _ := item["fields"].([]any)
			if len(fields) != 1 {
				t.Error("missing field")
			} else {
				field, _ := fields[0].(map[string]any)
				assert.Equal(t, description, field["description"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"description rejected by API"}`))
	}))
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + descriptionLengthConfig([]string{description}, false), ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestAgentMemoryDescriptionRequired(t *testing.T) {
	for _, scope := range []string{"type", "field"} {
		for _, expression := range []string{`""`, "null"} {
			t.Run(scope+"/"+expression, func(t *testing.T) {
				config := descriptionLengthConfig([]string{"Description marker"}, false)
				if scope == "type" {
					config = strings.Replace(config, `description="Description marker"`, "description="+expression, 1)
				} else {
					index := strings.LastIndex(config, `description="Description marker"`)
					config = config[:index] + strings.Replace(config[index:], `description="Description marker"`, "description="+expression, 1)
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, "http://127.0.0.1:1") + config, PlanOnly: true, ExpectError: regexp.MustCompile("length|Missing Configuration for Required Attribute|Missing required argument")}}})
			})
		}
	}
}
