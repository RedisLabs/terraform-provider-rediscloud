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
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryCustomTypeReplacementLifecycle(t *testing.T) {
	const topic = `fields {
name="topic"
type="str"
description="Topic"
}`
	const confidence = `fields {
name="confidence"
type="float"
description="Confidence"
}`
	const strategy = `extraction_strategy {
prompt="Extract preferences"
enabled=true
}`
	block := func(name, fields, strategy string) string {
		return fmt.Sprintf(`custom_memory_types {
 name=%q
 description="User preferences"
 %s
 %s
}
`, name, fields, strategy)
	}
	base := block("preference", topic+"\n"+confidence, strategy)
	support := block("support", topic, strategy)
	unknownName := strings.Replace(base, `name="preference"`, `name=terraform_data.value.output`, 1)
	unknownDescription := strings.Replace(base, `description="User preferences"`, `description=terraform_data.value.output`, 1)
	unknownFieldType := strings.Replace(base, `type="str"`, `type=terraform_data.value.output`, 1)
	unknownAddition := strings.Replace(support, `name="support"`, `name=terraform_data.value.output`, 1)
	for _, tc := range []struct {
		name, before, after, dependency, errorPattern string
		replace, update, protect                      bool
	}{
		{name: "unchanged", before: base, after: base},
		{name: "add type", before: base, after: base + support, update: true},
		{name: "prompt update", before: base, after: strings.Replace(base, "Extract preferences", "Extract durable preferences", 1), update: true},
		{name: "disable extraction", before: base, after: strings.Replace(base, "enabled=true", "enabled=false", 1), update: true},
		{name: "add extraction strategy", before: block("preference", topic+"\n"+confidence, ""), after: base, update: true},
		{name: "remove type", before: base + support, after: support, replace: true},
		{name: "remove all types", before: base, replace: true},
		{name: "rename type", before: base, after: strings.Replace(base, `name="preference"`, `name="renamed"`, 1), replace: true},
		{name: "change description", before: base, after: strings.Replace(base, "User preferences", "Durable preferences", 1), replace: true},
		{name: "add field", before: block("preference", topic, strategy), after: base, replace: true},
		{name: "remove field", before: base, after: block("preference", topic, strategy), replace: true},
		{name: "rename field", before: base, after: strings.Replace(base, `name="topic"`, `name="category"`, 1), replace: true},
		{name: "change field type", before: base, after: strings.Replace(base, `type="str"`, `type="int"`, 1), replace: true},
		{name: "change field description", before: base, after: strings.Replace(base, `description="Topic"`, `description="Preference topic"`, 1), replace: true},
		{name: "remove extraction strategy", before: base, after: block("preference", topic+"\n"+confidence, ""), replace: true},
		{name: "reorder types", before: base + support, after: support + base, update: true},
		{name: "reorder fields", before: base, after: block("preference", confidence+"\n"+topic, strategy), update: true},
		{name: "unknown existing name unchanged", before: base, after: unknownName, dependency: `resource "terraform_data" "value" { input="preference" }`, replace: true},
		{name: "unknown existing name changed", before: base, after: unknownName, dependency: `resource "terraform_data" "value" { input="renamed" }`, replace: true},
		{name: "unknown existing description unchanged", before: base, after: unknownDescription, dependency: `resource "terraform_data" "value" { input="User preferences" }`, replace: true},
		{name: "unknown existing description changed", before: base, after: unknownDescription, dependency: `resource "terraform_data" "value" { input="Changed" }`, replace: true},
		{name: "unknown existing field unchanged", before: base, after: unknownFieldType, dependency: `resource "terraform_data" "value" { input="str" }`, replace: true},
		{name: "unknown existing field changed", before: base, after: unknownFieldType, dependency: `resource "terraform_data" "value" { input="int" }`, replace: true},
		{name: "unknown new type name", before: base, after: base + unknownAddition, dependency: `resource "terraform_data" "value" { input="support" }`, update: true},
		{name: "unknown new field type", before: base, after: base + strings.Replace(support, `type="str"`, `type=terraform_data.value.output`, 1), dependency: `resource "terraform_data" "value" { input="str" }`, update: true},
		{name: "unknown mutable prompt", before: base, after: strings.Replace(base, `prompt="Extract preferences"`, `prompt=terraform_data.value.output`, 1), dependency: `resource "terraform_data" "value" { input="Extract durable preferences" }`, update: true},
		{name: "unknown mutable enabled", before: base, after: strings.Replace(base, "enabled=true", "enabled=terraform_data.value.output", 1), dependency: `resource "terraform_data" "value" { input=false }`, update: true},
		{name: "duplicate names", before: base, after: base + base, errorPattern: "Duplicate Agent Memory custom memory type"},
		{name: "unknown duplicate addition", before: base, after: base + unknownAddition, dependency: `resource "terraform_data" "value" { input="preference" }`, errorPattern: "Duplicate Agent Memory custom memory type"},
		{name: "removal prevent destroy", before: base, protect: true, errorPattern: "prevent_destroy|cannot be destroyed"},
		{name: "redefinition prevent destroy", before: base, after: strings.Replace(base, "User preferences", "Changed", 1), protect: true, errorPattern: "prevent_destroy|cannot be destroyed"},
		{name: "unknown name prevent destroy", before: base, after: unknownName, dependency: `resource "terraform_data" "value" { input="preference" }`, protect: true, errorPattern: "prevent_destroy|cannot be destroyed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, patches, deletes := 0, 0, 0
			lastID := ""
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					require.NoError(t, json.NewDecoder(r.Body).Decode(&store))
					creates++
					lastID = fmt.Sprintf("store-%d", creates)
					store["storeId"] = lastID
					store["databaseId"] = "123"
					store["status"] = "READY"
					store["extractionStrategy"] = "INSTRUCT"
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": lastID}}})
				case r.Method == http.MethodGet && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					_ = json.NewEncoder(w).Encode(store)
				case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.NotContains(t, payload, "customMemoryTypes")
					entries, _ := store["customMemoryTypes"].([]any)
					additions, _ := payload["addCustomMemoryTypes"].([]any)
					entries = append(entries, additions...)
					updates, _ := payload["updateCustomMemoryTypeStrategies"].([]any)
					for _, u := range updates {
						update := u.(map[string]any)
						found := false
						for _, v := range entries {
							entry := v.(map[string]any)
							if entry["name"] != update["typeName"] {
								continue
							}
							found = true
							extraction, _ := entry["extractionStrategy"].(map[string]any)
							if extraction == nil {
								extraction = map[string]any{}
							}
							for _, key := range []string{"prompt", "enabled"} {
								if value, ok := update[key]; ok {
									extraction[key] = value
								}
							}
							entry["extractionStrategy"] = extraction
						}
						assert.True(t, found, "strategy update must reference an existing type")
					}
					if len(entries) > 0 {
						store["customMemoryTypes"] = entries
					}
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodDelete && store != nil && r.URL.Path == "/memory-stores/"+lastID:
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := func(types, dependency string, protect bool) string {
				lifecycle := ""
				if protect {
					lifecycle = "lifecycle { prevent_destroy=true }"
				}
				return testProviderConfig(t, server.URL) + dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="custom-type-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 %s
 %s
}
`, types, lifecycle)
			}
			initial, changed := config(tc.before, "", false), config(tc.after, tc.dependency, tc.protect)
			const address = "rediscloud_agent_memory.example"
			steps := []resource.TestStep{{Config: initial, Check: resource.TestCheckResourceAttr(address, "id", "store-1")}}
			if tc.errorPattern != "" {
				step := resource.TestStep{Config: changed, ExpectError: regexp.MustCompile(tc.errorPattern)}
				if tc.protect || tc.dependency == "" {
					step.PlanOnly = true
				}
				restore := initial
				if tc.dependency != "" && !tc.protect {
					restore = config(tc.before, tc.dependency, false)
				}
				steps = append(steps, step, resource.TestStep{Config: restore, PlanOnly: true})
			} else {
				action, expectedID := plancheck.ResourceActionNoop, "store-1"
				if tc.replace {
					action, expectedID = plancheck.ResourceActionDestroyBeforeCreate, "store-2"
				} else if tc.update {
					action = plancheck.ResourceActionUpdate
				}
				steps = append(steps, resource.TestStep{Config: changed, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)}}, Check: resource.TestCheckResourceAttr(address, "id", expectedID)}, resource.TestStep{Config: changed, PlanOnly: true})
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps})
			mu.Lock()
			defer mu.Unlock()
			expectedCreates, expectedPatches := 1, 0
			if tc.replace {
				expectedCreates = 2
			}
			if tc.update {
				expectedPatches = 1
			}
			assert.Equal(t, expectedCreates, creates)
			assert.Equal(t, expectedCreates, deletes)
			assert.Equal(t, expectedPatches, patches)
			assert.Nil(t, store)
		})
	}
}
