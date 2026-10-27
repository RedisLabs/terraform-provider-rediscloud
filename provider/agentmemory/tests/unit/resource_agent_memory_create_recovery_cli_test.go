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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryCreateReadRecoveryLifecycle(t *testing.T) {
	for _, defaults := range []bool{false, true} {
		for _, scenario := range []string{"malformed", "404", "500", "503"} {
			t.Run(fmt.Sprintf("%s-defaults-%t", scenario, defaults), func(t *testing.T) {
				var mu sync.Mutex
				creates, deletes := 0, 0
				failedRead := false
				live := map[string]bool{}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
						// A tainted store must be deleted before Terraform creates its replacement.
						if len(live) != 0 {
							t.Error("new create attempted while an earlier fixture remains live")
						}
						creates++
						live[fmt.Sprintf("store-%d", creates)] = true
						_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
					case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
						_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "create-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": fmt.Sprintf("store-%d", creates)}}})
					case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/memory-stores/"):
						id := strings.TrimPrefix(r.URL.Path, "/memory-stores/")
						if !live[id] {
							http.NotFound(w, r)
							return
						}
						database := "123"
						if !failedRead {
							failedRead = true
							switch scenario {
							case "malformed":
								database = "not-a-number"
							case "404":
								w.WriteHeader(404)
								return
							case "500":
								w.WriteHeader(500)
								return
							case "503":
								w.WriteHeader(503)
								return
							}
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"storeId": id, "name": "recovery-store", "databaseId": database, "status": "READY", "shortMemory": map[string]any{"ttlSeconds": 86400}, "longTermMemory": map[string]any{"ttlSeconds": 31536000}, "extractionCadence": map[string]any{"activeIntervalSeconds": 300}, "extractionStrategy": "INSTRUCT", "endpoint": "https://memory.example.test", "createdAt": "2026-10-16T12:00:00Z"})
					case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/memory-stores/"):
						id := strings.TrimPrefix(r.URL.Path, "/memory-stores/")
						if !live[id] {
							http.NotFound(w, r)
							return
						}
						delete(live, id)
						deletes++
						_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
					case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
						_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
					default:
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				defer func() {
					mu.Lock()
					defer mu.Unlock()
					if len(live) != 0 || creates != deletes {
						t.Errorf("local orphan: creates=%d deletes=%d live=%v", creates, deletes, live)
					}
				}()
				prefix := testProviderConfig(t, server.URL)
				ttl := ""
				if !defaults {
					ttl = "short_term_ttl_seconds=86400\nlong_term_ttl_seconds=31536000\nextraction_cadence_seconds=300\n"
				}
				config := prefix + fmt.Sprintf("resource \"rediscloud_agent_memory\" \"example\" {\nname=\"recovery-store\"\ndatabase_id=123\n%s}\n", ttl)
				check := func(state *terraform.State) error {
					mu.Lock()
					defer mu.Unlock()
					expectedCreates, expectedDeletes := 1, 0
					if scenario != "404" {
						expectedCreates, expectedDeletes = 2, 1
					}
					if creates != expectedCreates || deletes != expectedDeletes || len(live) != 1 {
						return fmt.Errorf("unexpected recovery: creates=%d deletes=%d live=%d", creates, deletes, len(live))
					}
					if state.RootModule().Resources["rediscloud_agent_memory.example"].Primary.Tainted {
						return fmt.Errorf("successful recovery left the store tainted")
					}
					return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", fmt.Sprintf("store-%d", creates))(state)
				}
				createStep := resource.TestStep{Config: config, Check: check}
				if scenario != "404" {
					message := "Agent Memory service failed to become ready"
					if scenario == "malformed" {
						message = "Unexpected Agent Memory database ID"
					}
					createStep.ExpectError = regexp.MustCompile(message)
					createStep.Check = nil
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
					createStep,
					{Config: config, Check: check},
					{Config: config, PlanOnly: true},
				}})
			})
		}
	}
}
