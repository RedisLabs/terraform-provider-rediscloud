package agentmemory_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryInactiveSummarizationDefinitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture":
			_, _ = w.Write([]byte(`{"taskId":"fixture","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			_, _ = w.Write([]byte(`{"storeId":"store-1","name":"fixture","databaseId":"123","status":"READY","summarization":{"enabled":false,"triggerStrategy":"event_count","eventCount":{"threshold":20,"retainCount":10}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + `resource "rediscloud_agent_memory" "example" {
 name="fixture"
 database_id=123
 summarization {
  enabled=false
  trigger_strategy="event_count"
  event_count {
   threshold=20
   retain_count=10
  }
 }
}`
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config, Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.trigger_strategy", "event_count"),
			resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.event_count.threshold", "20"),
		)},
		{ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateVerify: true},
		{Config: config, PlanOnly: true},
	}})
}
