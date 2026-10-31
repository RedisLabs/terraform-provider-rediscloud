package unit_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
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
			storeDeleted := false
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
					if storeDeleted {
						http.NotFound(w, r)
						return
					}
					if failRead {
						failRead = false
						if scenario == "http-error" {
							w.WriteHeader(500)
							_, _ = w.Write([]byte(`{"message":"simulated read failure"}`))
						} else {
							_, _ = w.Write([]byte(`{"storeId":"store-1","name":"update-recovery","databaseId":"not-a-number","status":"READY"}`))
						}
						return
					}
					writeStoreResponse(t, w, "update-recovery", shortTTL, 31536000, 300)
				case "DELETE /memory-stores/store-1":
					storeDeleted = true
					_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
				case "GET /tasks/delete-task":
					_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			testProviderConfig(t, server.URL)
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{
					ConfigFile: config.StaticFile("testdata/store_basic.tf"),
					ConfigVariables: config.Variables{
						"name":      config.StringVariable("update-recovery"),
						"short_ttl": config.IntegerVariable(86400),
						"long_ttl":  config.IntegerVariable(31536000),
						"cadence":   config.IntegerVariable(300),
					},
				},
				{
					ConfigFile: config.StaticFile("testdata/store_basic.tf"),
					ConfigVariables: config.Variables{
						"name":      config.StringVariable("update-recovery"),
						"short_ttl": config.IntegerVariable(172800),
						"long_ttl":  config.IntegerVariable(31536000),
						"cadence":   config.IntegerVariable(300),
					},
					ExpectError: regexp.MustCompile("Agent Memory service failed to become ready after update|Unexpected Agent Memory database ID"),
				},
				// A failed refresh must preserve identity so retry can read the accepted update.
				{
					ConfigFile: config.StaticFile("testdata/store_basic.tf"),
					ConfigVariables: config.Variables{
						"name":      config.StringVariable("update-recovery"),
						"short_ttl": config.IntegerVariable(172800),
						"long_ttl":  config.IntegerVariable(31536000),
						"cadence":   config.IntegerVariable(300),
					},
					Check: func(state *terraform.State) error {
						mu.Lock()
						defer mu.Unlock()
						if creates != 1 || updates != 1 {
							return fmt.Errorf("unexpected repeat mutation: creates=%d updates=%d", creates, updates)
						}
						return resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "short_term_ttl_seconds", "172800"))(state)
					},
				},
			}})
		})
	}
}
