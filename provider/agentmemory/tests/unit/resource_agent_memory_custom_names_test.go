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

func customNamesConfig(names []string, unknown bool) string {
	expressions := make([]string, len(names))
	for i, name := range names {
		expressions[i] = fmt.Sprintf("%q", name)
	}
	dependency := ""
	if unknown {
		// Adding a type must not invalidate the known names of existing types.
		for i, value := range expressions {
			dependency += fmt.Sprintf("resource \"terraform_data\" \"name_%d\" { input=%s }\n", i, value)
			expressions[i] = fmt.Sprintf("terraform_data.name_%d.output", i)
		}
	}
	var types []string
	for _, expression := range expressions {
		types = append(types, fmt.Sprintf(`custom_memory_types {
 name=%s
 description="Custom memory"
 fields {
  name=%s
  description="Custom field"
  type="str"
 }
}`, expression, expression))
	}
	return dependency + fmt.Sprintf(`resource "rediscloud_agent_memory" "example" {
 name="custom-names-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 long_term_memory_exclusions {
  enabled=true
  custom_detectors {
   enabled=true
   detectors {
    name=%s
    enabled=true
    action="redact"
    matcher {
     kind="regex"
     regex { pattern="secret-[0-9]+" }
    }
   }
  }
 }
 %s
}`, expressions[len(expressions)-1], strings.Join(types, "\n"))
}

func TestAgentMemoryCustomNameRulesByAPI(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown=%t", unknown), func(t *testing.T) {
			names := []string{"1.preferences/" + strings.Repeat("é", 65), "2.preferences/" + strings.Repeat("界", 65), "3.rejected/" + strings.Repeat("x", 65)}
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
					detectors := listLimitEntries(payload, "custom-detectors")
					if len(detectors) != 1 {
						t.Error("missing detector")
						w.WriteHeader(400)
						return
					}
					detector, _ := detectors[0].(map[string]any)
					name, _ := detector["name"].(string)
					observed = append(observed, name)
					types, _ := payload["customMemoryTypes"].([]any)
					if r.Method == http.MethodPost {
						creates++
					} else {
						patches++
						types, _ = payload["addCustomMemoryTypes"].([]any)
					}
					assert.Len(t, types, 1, "create/update must send exactly one new type")
					if len(types) != 1 {
						w.WriteHeader(400)
						return
					}
					memoryType, _ := types[0].(map[string]any)
					assert.Equal(t, name, memoryType["name"])
					fields, _ := memoryType["fields"].([]any)
					if len(fields) != 1 {
						t.Error("missing field")
						w.WriteHeader(400)
						return
					}
					field, _ := fields[0].(map[string]any)
					assert.Equal(t, name, field["name"], "all three names must be transmitted verbatim")
					// Simulate acceptance of expanded syntax and rejection of another name by the API.
					if name == names[2] {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"name rejected by API"}`))
						return
					}
					if store == nil {
						store = payload
					} else {
						existing, _ := store["customMemoryTypes"].([]any)
						store["customMemoryTypes"] = append(existing, types...)
						store["longTermMemoryExclusions"] = payload["longTermMemoryExclusions"]
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
			config := func(count int) string {
				return testProviderConfig(t, server.URL) + customNamesConfig(names[:count], unknown)
			}
			check := func(count int) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 || state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store replaced or tainted: creates=%d deletes=%d", creates, deletes)
					}
					checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.#", fmt.Sprint(count)), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.detectors.0.name", names[count-1])}
					for i := 0; i < count; i++ {
						checks = append(checks, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("custom_memory_types.%d.name", i), names[i]), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("custom_memory_types.%d.fields.0.name", i), names[i]))
					}
					return resource.ComposeTestCheckFunc(checks...)(state)
				}
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(1), Check: check(1)},
				{Config: config(2), Check: check(2)},
				{Config: config(3), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: config(2), Check: check(2)},
				{Config: config(2), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, names, observed)
			assert.Equal(t, 1, creates)
			assert.Equal(t, 2, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryCustomNameCreateRejectedByAPI(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	name := "1.rejected/" + strings.Repeat("x", 65)
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
		entries := listLimitEntries(payload, "custom-detectors")
		if len(entries) != 1 {
			t.Error("missing detector")
		} else {
			item, _ := entries[0].(map[string]any)
			assert.Equal(t, name, item["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"name rejected by API"}`))
	}))
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + customNamesConfig([]string{name}, false), ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestAgentMemoryCustomFieldNameRequired(t *testing.T) {
	for _, expression := range []string{`""`, "null"} {
		t.Run(expression, func(t *testing.T) {
			config := reservedTypeConfig(`"preference"`, "", "topic")
			config = strings.Replace(config, `name="topic"`, "name="+expression, 1)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, "http://127.0.0.1:1") + config, PlanOnly: true, ExpectError: regexp.MustCompile("length|Missing Configuration for Required Attribute|Missing required argument")}}})
		})
	}
}
