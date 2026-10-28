package unit_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryInactiveSummarizationDefinitions(t *testing.T) {
	var storeDeleted atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			storeDeleted.Store(r.Method == http.MethodDelete)
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture":
			_, _ = w.Write([]byte(`{"taskId":"fixture","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			if storeDeleted.Load() {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`{"storeId":"store-1","name":"fixture","databaseId":"123","status":"READY","summarization":{"enabled":false,"triggerStrategy":"event_count","eventCount":{"threshold":20,"retainCount":10}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	testProviderConfig(t, server.URL)

	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{
			ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_inactive.tf"),
			Check: resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.trigger_strategy", "event_count"),
				resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "summarization.0.event_count.0.threshold", "20"),
			),
		},
		{ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateVerify: true},
		{
			ConfigFile: config.StaticFile("testdata/lifecycle/summarization/summarization_inactive.tf"),
			PlanOnly:   true,
		},
	}})
}
