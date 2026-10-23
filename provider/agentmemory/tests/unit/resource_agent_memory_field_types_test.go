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

func fieldTypesConfig(fieldTypes []string, unknown bool) string {
	config := listLimitConfig("custom-types", len(fieldTypes), false)
	var values []string
	for _, fieldType := range fieldTypes {
		values = append(values, fmt.Sprintf("%q", fieldType))
	}
	parts := strings.Split(config, `type="str"`)
	for i, value := range values {
		expression := value
		if unknown {
			expression = fmt.Sprintf("terraform_data.field_type_%d.output", i)
		}
		parts[i] += "type=" + expression
	}
	config = strings.Join(parts, "")
	if unknown {
		// Independent dependencies keep existing definitions known when a type is added.
		for i, value := range values {
			config = fmt.Sprintf("resource \"terraform_data\" \"field_type_%d\" { input=%s }\n", i, value) + config
		}
	}
	return config
}

func TestAgentMemoryFieldTypesByAPI(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprintf("unknown=%t", unknown), func(t *testing.T) {
			fieldTypes := []string{"str", "future-scalar", "future-list", "rejected-by-api"}
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
					assert.Equal(t, fmt.Sprintf("custom_%d", len(observed)), item["name"])
					fields, _ := item["fields"].([]any)
					if len(fields) != 1 {
						t.Error("missing field")
						w.WriteHeader(400)
						return
					}
					field, _ := fields[0].(map[string]any)
					fieldType, _ := field["type"].(string)
					observed = append(observed, fieldType)
					// Simulate new API-supported types while retaining server-side rejection.
					if fieldType == "rejected-by-api" {
						w.WriteHeader(400)
						_, _ = w.Write([]byte(`{"message":"fieldType rejected by API"}`))
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
			config := func(count int) string { return prefix + fieldTypesConfig(fieldTypes[:count], unknown) }
			check := func(count int) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != 0 || state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("store replaced or tainted: creates=%d deletes=%d", creates, deletes)
					}
					checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "custom_memory_types.#", fmt.Sprint(count))}
					for i := 0; i < count; i++ {
						checks = append(checks, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("custom_memory_types.%d.fields.0.type", i), fieldTypes[i]))
					}
					return resource.ComposeTestCheckFunc(checks...)(state)
				}
			}
			redefined := prefix + fieldTypesConfig([]string{"changed-field-type", fieldTypes[1], fieldTypes[2]}, unknown)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(1), Check: check(1)},
				{Config: config(2), Check: check(2)},
				{Config: config(3), Check: check(3)},
				{Config: config(4), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
				{Config: config(3), Check: check(3)},
				{Config: redefined, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config(1), PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: config(3), Check: check(3)},
				{Config: config(3), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, fieldTypes, observed)
			assert.Equal(t, 1, creates)
			assert.Equal(t, 3, patches)
			assert.Equal(t, 1, deletes)
			assert.Nil(t, store)
		})
	}
}

func TestAgentMemoryFieldTypeCreateRejectedByAPI(t *testing.T) {
	var mu sync.Mutex
	calls := 0
	fieldType := "rejected-by-api"
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
			fields, _ := item["fields"].([]any)
			if len(fields) != 1 {
				t.Error("missing field")
			} else {
				field, _ := fields[0].(map[string]any)
				assert.Equal(t, fieldType, field["type"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"fieldType rejected by API"}`))
	}))
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: testProviderConfig(t, server.URL) + fieldTypesConfig([]string{fieldType}, false), ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, calls)
}

func TestAgentMemoryFieldTypeValidation(t *testing.T) {
	for _, c := range []struct{ expression, errorPattern string }{
		{expression: `"str"`}, {expression: `"int"`}, {expression: `"float"`}, {expression: `"bool"`},
		{expression: `"list[str]"`}, {expression: `"list[float]"`}, {expression: `"object"`},
		{expression: `"future-scalar"`}, {expression: `""`, errorPattern: "length"},
		{expression: "null", errorPattern: "Missing Configuration for Required Attribute|Missing required argument"},
	} {
		t.Run(c.expression, func(t *testing.T) {
			config := strings.Replace(listLimitConfig("custom-types", 1, false), `type="str"`, "type="+c.expression, 1)
			step := resource.TestStep{Config: testProviderConfig(t, "http://127.0.0.1:1") + config, PlanOnly: true, ExpectNonEmptyPlan: true}
			if c.errorPattern != "" {
				step.ExpectError = regexp.MustCompile(c.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
		})
	}
}

func TestAgentMemoryFieldTypeResolvedMissing(t *testing.T) {
	for _, c := range []struct{ expression, errorPattern string }{
		{expression: `""`, errorPattern: "length"},
		{expression: "null", errorPattern: "Missing Configuration for Required Attribute|Missing required argument"},
	} {
		t.Run(c.expression, func(t *testing.T) {
			dependency := fmt.Sprintf(`resource "terraform_data" "value" { input={type=%s} }`, c.expression)
			config := strings.Replace(listLimitConfig("custom-types", 1, false), `type="str"`, "type=tostring(terraform_data.value.output.type)", 1)
			prefix := testProviderConfig(t, "http://127.0.0.1:1") + dependency + "\n"
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: prefix + config, PlanOnly: true, ExpectNonEmptyPlan: true},
				{Config: prefix + config, ExpectError: regexp.MustCompile(c.errorPattern)},
				{Config: prefix + listLimitConfig("custom-types", 1, false), PlanOnly: true, ExpectNonEmptyPlan: true},
			}})
		})
	}
}
