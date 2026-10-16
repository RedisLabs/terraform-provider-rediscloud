package agentmemory_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

type inspectDeadlineTransport struct {
	base     http.RoundTripper
	host     string
	requests *atomic.Int32
	t        *testing.T
}

func (tr inspectDeadlineTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == tr.host {
		tr.requests.Add(1)
		deadline, ok := req.Context().Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > time.Hour {
			tr.t.Error("Terraform CLI reached the API without a bounded resource operation deadline")
		}
	}
	return tr.base.RoundTrip(req)
}

func TestAgentMemoryTerraformLifecycleBoundsRequests(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(fmt.Sprintf("configured-%t", configured), func(t *testing.T) {
			var requests, creates, deletes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "POST /memory-stores":
					creates.Add(1)
					_, _ = w.Write([]byte(`{"taskId":"create-task"}`))
				case "GET /tasks/create-task":
					_, _ = w.Write([]byte(`{"status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case "GET /memory-stores/store-1":
					if deletes.Load() != 0 {
						http.NotFound(w, r)
						return
					}
					_, _ = w.Write([]byte(`{"storeId":"store-1","name":"deadline-cli","databaseId":"123","status":"READY","shortMemory":{"ttlSeconds":86400},"longTermMemory":{"ttlSeconds":31536000},"extractionCadence":{"activeIntervalSeconds":300},"extractionStrategy":"INSTRUCT"}`))
				case "DELETE /memory-stores/store-1":
					deletes.Add(1)
					_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
				case "GET /tasks/delete-task":
					_, _ = w.Write([]byte(`{"status":"processing-completed"}`))
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			// Inspect the client's context; HTTP does not transmit its deadline to the server.
			previous := http.DefaultTransport
			http.DefaultTransport = inspectDeadlineTransport{base: previous, host: strings.TrimPrefix(server.URL, "http://"), requests: &requests, t: t}
			t.Cleanup(func() { http.DefaultTransport = previous })
			config := testProviderConfig(t, server.URL) + `
resource "rediscloud_agent_memory" "example" {
  name = "deadline-cli"
  database_id = 123
}
`
			check := resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")
			if configured {
				config = strings.Replace(config, "database_id = 123", "database_id = 123\ntimeouts {\ncreate=\"7m\"\nread=\"7m\"\nupdate=\"7m\"\ndelete=\"7m\"\n}", 1)
				check = resource.ComposeTestCheckFunc(check, resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "timeouts.create", "7m"))
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: config, Check: check},
					{Config: config, PlanOnly: true},
				},
			})
			if requests.Load() == 0 || creates.Load() != 1 || deletes.Load() != 1 {
				t.Error("Terraform lifecycle did not exercise and clean up the local fixture exactly once")
			}

		})
	}
}
