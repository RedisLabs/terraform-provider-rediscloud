package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryOptionalSummarizationThresholdsLifecycle(t *testing.T) {
	var mu sync.Mutex
	policy := map[string]any{"enabled": true, "triggerStrategy": "event_count", "eventCount": map[string]any{"threshold": 20, "retainCount": 10}}
	creates, deletes := 0, 0
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
			}
			requested, ok := payload["summarization"].(map[string]any)
			if !ok {
				t.Error("request omitted summarization")
				w.WriteHeader(500)
				return
			}
			// Omitted eventCount retains service defaults or the previously stored thresholds.
			for key, value := range requested {
				policy[key] = value
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case "GET /tasks/fixture-task":
			_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case "GET /memory-stores/store-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"storeId": "store-1", "name": "summary-defaults", "databaseId": "123", "status": "READY", "summarization": policy})
		case "DELETE /memory-stores/store-1":
			deletes++
			policy = nil
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + `
resource "rediscloud_agent_memory" "example" {
 name="summary-defaults"
 database_id=123
 summarization { enabled=true }
}
`

	configured := testProviderConfig(t, server.URL) + `resource "rediscloud_agent_memory" "example" {
 name="summary-defaults"
 database_id=123
 summarization {
  enabled=true
  event_count {
   threshold=12
   retain_count=6
  }
 }
}`
	check := func(count string) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if creates != 1 || deletes != 0 || policy["eventCount"] == nil {
				return fmt.Errorf("store or retained thresholds changed unexpectedly")
			}
			thresholdCheck := resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.threshold")
			if count == "1" {
				thresholdCheck = resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.threshold", "12")
			}
			return resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), thresholdCheck)(state)
		}
	}
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config, Check: check("0")},
		{Config: config, PlanOnly: true},
		{Config: configured, Check: check("1")},
		{Config: configured, PlanOnly: true},
		{Config: config, Check: check("0")},
		{Config: config, PlanOnly: true},
		{Config: config, ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateId: "store-1", ImportStateCheck: func(states []*terraform.InstanceState) error {
			if len(states) != 1 || states[0].Attributes["summarization.event_count.threshold"] != "12" {
				return fmt.Errorf("import omitted stored summarization thresholds")
			}
			return nil
		}},
		{Config: config, PlanOnly: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || deletes != 1 || policy != nil {
		t.Fatal("fixture cleanup failed")
	}
}
