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

var listLimitCases = []struct {
	name, attribute            string
	initial, updated, rejected int
}{
	{"built-in", "long_term_memory_exclusions.built_in_detectors.detectors", 4, 5, 6},
	{"custom-detectors", "long_term_memory_exclusions.custom_detectors.detectors", 33, 34, 35},
	{"custom-types", "custom_memory_types", 4, 5, 6},
}

func listLimitConfig(kind string, count int, unknown bool) string {
	ids := []string{"credit-card", "email", "ip-address", "phone", "us-ssn"}
	var blocks, names []string
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("custom_%d", i)
		if kind == "built-in" {
			name = ids[i%len(ids)]
		}
		names = append(names, fmt.Sprintf("%q", name))
		expression := fmt.Sprintf("%q", name)
		if unknown {
			expression = fmt.Sprintf("terraform_data.names.output[%d]", i)
			if kind == "custom-types" {
				expression = fmt.Sprintf("terraform_data.type_name_%d.output", i)
			}
		}
		switch kind {
		case "built-in":
			blocks = append(blocks, fmt.Sprintf("detectors {\nid=%s\nenabled=true\naction=\"redact\"\n}", expression))
		case "custom-detectors":
			blocks = append(blocks, fmt.Sprintf(`detectors {
 name=%s
 enabled=true
 action="redact"
 matcher {
  kind="regex"
  regex { pattern="secret-[0-9]+" }
 }
}`, expression))
		case "custom-types":
			blocks = append(blocks, fmt.Sprintf(`custom_memory_types {
 name=%s
 description="Custom memory"
 fields {
  name="topic"
  description="Topic"
  type="str"
 }
}`, expression))
		}
	}
	settings := strings.Join(blocks, "\n")
	if kind != "custom-types" {
		block := "custom_detectors"
		if kind == "built-in" {
			block = "built_in_detectors"
		}
		settings = fmt.Sprintf("long_term_memory_exclusions {\nenabled=true\n%s {\nenabled=true\n%s\n}\n}", block, settings)
	}
	dependency := ""
	if unknown {
		if kind == "custom-types" {
			// An appended type should not make existing immutable names unknown.
			for i, value := range names {
				dependency += fmt.Sprintf("resource \"terraform_data\" \"type_name_%d\" { input=%s }\n", i, value)
			}
		} else {
			dependency = fmt.Sprintf("resource \"terraform_data\" \"names\" { input=[%s] }\n", strings.Join(names, ","))
		}
	}
	return dependency + fmt.Sprintf(`resource "rediscloud_agent_memory" "example" {
 name="list-limits-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 %s
}`, settings)
}

func listLimitEntries(payload map[string]any, kind string) []any {
	if kind == "custom-types" {
		entries, _ := payload["customMemoryTypes"].([]any)
		return entries
	}
	exclusions, _ := payload["longTermMemoryExclusions"].(map[string]any)
	block := "customDetectors"
	if kind == "built-in" {
		block = "builtInDetectors"
	}
	config, _ := exclusions[block].(map[string]any)
	entries, _ := config["detectors"].([]any)
	return entries
}

func TestAgentMemoryListLimitsByAPI(t *testing.T) {
	for _, c := range listLimitCases {
		for _, unknown := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unknown=%t", c.name, unknown), func(t *testing.T) {
				var mu sync.Mutex
				var store map[string]any
				creates, patches, deletes := 0, 0, 0
				var observed []int
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
						entries := listLimitEntries(payload, c.name)
						var existing []any
						if r.Method == http.MethodPost {
							creates++
						} else {
							patches++
							if c.name == "custom-types" {
								existing = listLimitEntries(store, c.name)
								entries, _ = payload["addCustomMemoryTypes"].([]any)
								assert.Len(t, entries, 1, "update must send only the added memory type")
							}
						}
						combined := append(append([]any{}, existing...), entries...)
						observed = append(observed, len(combined))
						expectedKind := "name"
						if c.name == "built-in" {
							expectedKind = "id"
						}
						ids := []string{"credit-card", "email", "ip-address", "phone", "us-ssn"}
						for i, entry := range combined {
							item, ok := entry.(map[string]any)
							if !ok {
								t.Error("invalid list item")
								w.WriteHeader(400)
								return
							}
							expected := fmt.Sprintf("custom_%d", i)
							if c.name == "built-in" {
								expected = ids[i%len(ids)]
							}
							assert.Equal(t, expected, item[expectedKind], "entries must retain identity and order")
						}
						// Simulate expanded custom-list limits; built-in IDs still use the current catalog.
						if len(combined) >= c.rejected {
							w.WriteHeader(400)
							_, _ = w.Write([]byte(`{"message":"list rejected by API"}`))
							return
						}
						if store == nil {
							store = payload
						} else if c.name == "custom-types" {
							store["customMemoryTypes"] = combined
						} else {
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
					return testProviderConfig(t, server.URL) + listLimitConfig(c.name, count, unknown)
				}
				check := func(count int) resource.TestCheckFunc {
					return func(state *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if creates != 1 || deletes != 0 || state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
							return fmt.Errorf("store replaced or tainted: creates=%d deletes=%d", creates, deletes)
						}
						return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", c.attribute+".#", fmt.Sprint(count)))(state)
					}
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
					{Config: config(c.initial), Check: check(c.initial)},
					{Config: config(c.updated), Check: check(c.updated)},
					{Config: config(c.rejected), ExpectError: regexp.MustCompile("Failed to update Agent Memory service")},
					{Config: config(c.updated), Check: check(c.updated)},
					{Config: config(c.updated), PlanOnly: true},
				}})
				mu.Lock()
				defer mu.Unlock()
				assert.Equal(t, []int{c.initial, c.updated, c.rejected}, observed)
				assert.Equal(t, 1, creates)
				assert.Equal(t, 2, patches)
				assert.Equal(t, 1, deletes)
				assert.Nil(t, store)
			})
		}
	}
}

func TestAgentMemoryListCreateRejectedByAPI(t *testing.T) {
	for _, c := range listLimitCases {
		t.Run(c.name, func(t *testing.T) {
			var mu sync.Mutex
			calls := 0
			// Each request exceeds the former local maximum, including the built-in list.
			count := c.initial
			if c.name == "built-in" {
				count = c.rejected
			}
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
				assert.Len(t, listLimitEntries(payload, c.name), count)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"message":"list rejected by API"}`))
			}))
			defer server.Close()
			config := testProviderConfig(t, server.URL) + listLimitConfig(c.name, count, false)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory service")}}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, 1, calls, "rejected creation must not trigger reads or deletes")
		})
	}
}
