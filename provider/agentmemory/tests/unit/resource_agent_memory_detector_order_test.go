package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func detectorOrderConfig(kind string, names []string, action string) string {
	var detectors strings.Builder
	group := "built_in_detectors"
	for _, name := range names {
		if kind == "built-in" {
			fmt.Fprintf(&detectors, "detectors {\nid=%q\nenabled=true\naction=%q\n}\n", name, action)
		} else {
			group = "custom_detectors"
			fmt.Fprintf(&detectors, "detectors {\nname=%q\nenabled=true\naction=%q\nmatcher {\nkind=\"regex\"\nregex { pattern=\"ticket-[0-9]+\" }\n}\n}\n", name, action)
		}
	}
	return fmt.Sprintf(`resource "rediscloud_agent_memory" "example" {
 name="detector-order-store"
 database_id=123
 long_term_memory_exclusions {
  enabled=true
  %s {
   enabled=true
   %s
  }
 }
}`, group, detectors.String())
}

func TestAgentMemoryDetectorOrderLifecycle(t *testing.T) {
	for _, kind := range []string{"built-in", "custom-detectors"} {
		t.Run(kind, func(t *testing.T) {
			var mu sync.Mutex
			var store map[string]any
			creates, deletes, reads := 0, 0, 0
			group, identity := "built_in_detectors", "id"
			names := []string{"credit-card", "email", "phone"}
			if kind == "custom-detectors" {
				group = "custom_detectors"
				identity = "name"
				names = []string{"ticket", "tenant", "account"}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "POST /memory-stores", "PATCH /memory-stores/store-1":
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
						w.WriteHeader(500)
						return
					}
					if r.Method == http.MethodPost {
						creates++
						store = payload
					} else {
						store["longTermMemoryExclusions"] = payload["longTermMemoryExclusions"]
					}
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case "GET /tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case "GET /memory-stores/store-1":
					if store == nil {
						http.NotFound(w, r)
						return
					}
					// Change response order without changing the stored detector configuration.
					raw, err := json.Marshal(store)
					if err != nil {
						t.Error(err)
						w.WriteHeader(500)
						return
					}
					var response map[string]any
					if err := json.Unmarshal(raw, &response); err != nil {
						t.Error(err)
						w.WriteHeader(500)
						return
					}
					reads++
					entries := listLimitEntries(response, kind)
					if reads%2 == 1 {
						for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
							entries[i], entries[j] = entries[j], entries[i]
						}
					}
					response["storeId"], response["databaseId"], response["status"] = "store-1", "123", "READY"
					_ = json.NewEncoder(w).Encode(response)
				case "DELETE /memory-stores/store-1":
					deletes++
					store = nil
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := func(values []string, action string) string {
				return testProviderConfig(t, server.URL) + detectorOrderConfig(kind, values, action)
			}
			expectedDeletes := 0
			check := func(values []string, action string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || deletes != expectedDeletes {
						return fmt.Errorf("store was replaced")
					}
					prefix := "long_term_memory_exclusions.0." + group + ".0.detectors"
					checks := []resource.TestCheckFunc{resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", prefix+".#", fmt.Sprint(len(values)))}
					for i, name := range values {
						checks = append(checks, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("%s.%d.%s", prefix, i, identity), name), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", fmt.Sprintf("%s.%d.action", prefix, i), action))
					}
					return resource.ComposeTestCheckFunc(checks...)(state)
				}
			}
			reordered := []string{names[2], names[0], names[1]}
			var importedOrder []string
			var importStore map[string]any
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(names[:2], "redact"), Check: check(names[:2], "redact")},
				{Config: config(names[:2], "redact"), PlanOnly: true},
				{Config: config(names, "redact"), Check: check(names, "redact")},
				{Config: config(reordered, "drop"), Check: check(reordered, "drop")},
				{Config: config(reordered, "drop"), PlanOnly: true},
				{Config: config(names[:2], "drop"), Check: resource.ComposeTestCheckFunc(check(names[:2], "drop"), func(*terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					importStore = store
					return nil
				})},
			}})
			mu.Lock()
			if deletes != 1 || store != nil {
				t.Error("initial lifecycle did not clean up its store")
			}
			// Import the fixture into a fresh Terraform state without a removed block.
			store = importStore
			expectedDeletes = 1
			mu.Unlock()
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{

				{Config: config(names[:2], "drop"), ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateId: "store-1", ImportStatePersist: true, ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("unexpected imported resource count")
					}
					for i := range names[:2] {
						attribute := fmt.Sprintf("long_term_memory_exclusions.0.%s.0.detectors.%d.%s", group, i, identity)
						value := states[0].Attributes[attribute]
						if value == "" {
							return fmt.Errorf("import omitted detector identity")
						}
						importedOrder = append(importedOrder, value)
					}
					return nil
				}},
				{RefreshState: true, Check: func(state *terraform.State) error { return check(importedOrder, "drop")(state) }},
				{Config: config(names[:2], "drop"), Check: check(names[:2], "drop")},
				{Config: config(names[:2], "drop"), PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, 1, creates)
			require.Equal(t, 2, deletes)
			require.Nil(t, store)
		})
	}
}
