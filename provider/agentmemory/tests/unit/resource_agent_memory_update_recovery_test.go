package unit_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryAcceptedUpdateReadFailureRecovers(t *testing.T) {
	for _, scenario := range []string{"http-error", "invalid-database-id"} {
		t.Run(scenario, func(t *testing.T) {
			var mu sync.Mutex
			creates, updates, shortTTL := 0, 0, 86400
			failRead := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "POST /memory-stores":
					creates++
					_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
				case "GET /tasks/create-task":
					_, _ = w.Write([]byte(`{"taskId":"create-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case "PATCH /memory-stores/store-1":
					updates++
					shortTTL = 172800
					failRead = true
					_, _ = w.Write([]byte(`{"taskId":"update-task"}`))
				case "GET /tasks/update-task":
					_, _ = w.Write([]byte(`{"taskId":"update-task","status":"processing-completed"}`))
				case "GET /memory-stores/store-1":
					if failRead {
						failRead = false
						if scenario == "http-error" {
							w.WriteHeader(500)
							_, _ = w.Write([]byte(`{"message":"simulated read failure"}`))
						} else {
							_, _ = w.Write([]byte(`{"storeId":"store-1","name":"update-recovery","databaseId":"not-a-number"}`))
						}
						return
					}
					writeStoreResponse(t, w, "update-recovery", shortTTL, 31536000, 300)
				case "DELETE /memory-stores/store-1":
					_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
				case "GET /tasks/delete-task":
					_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := func(ttl int) string {
				return testProviderConfig(t, server.URL) + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="update-recovery"
 database_id=123
 short_term_ttl_seconds=%d
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
}`, ttl)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: config(86400)},
				{Config: config(172800), ExpectError: regexp.MustCompile("Failed to read Agent Memory service after update|Unexpected Agent Memory database ID")},
				// A failed refresh must preserve identity so retry can read the accepted update.
				{Config: config(172800), Check: func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					if creates != 1 || updates != 1 {
						return fmt.Errorf("unexpected repeat mutation: creates=%d updates=%d", creates, updates)
					}
					return resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "short_term_ttl_seconds", "172800"))(state)
				}},
			}})
		})
	}
}
