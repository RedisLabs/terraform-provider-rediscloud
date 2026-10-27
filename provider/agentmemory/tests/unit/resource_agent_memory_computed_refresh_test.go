package unit_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryMutableComputedMetadata(t *testing.T) {
	var mu sync.Mutex
	updated := false
	creates, patches, deletes := 0, 0, 0
	updatedReads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			creates++
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			patches++
			updated = true
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			deletes++
			_, _ = w.Write([]byte(`{"taskId":"fixture"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture":
			_, _ = w.Write([]byte(`{"taskId":"fixture","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
			name, status, endpoint := "before", "READY", "https://old.example.test"
			if updated {
				name, endpoint = "after", "https://new.example.test"
				updatedReads++
				// Update completes at READY; a later refresh observes the outage.
				if updatedReads > 1 {
					status = "UNAVAILABLE"
				}
			}
			_, _ = fmt.Fprintf(w, `{"storeId":"store-1","name":%q,"databaseId":"123","status":%q,"endpoint":%q,"endpoints":[{"url":%q,"provider":"AWS","region":"us-east-1","isAccessible":%t}],"createdAt":"2026-10-17T12:00:00Z"}`, name, status, endpoint, endpoint, !updated)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := func(name string) string {
		return testProviderConfig(t, server.URL) + fmt.Sprintf(`resource "rediscloud_agent_memory" "example" {
 name=%q
 database_id=123
 }`, name)
	}
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config("before"), Check: resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "status", "READY")},
		{Config: config("after"), Check: resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"),
			resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "status", "READY"),
			resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoint", "https://new.example.test"),
			resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "endpoints.0.is_accessible", "false"),
		)},
		{RefreshState: true, Check: resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "status", "UNAVAILABLE")},
		{Config: config("after"), PlanOnly: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, creates)
	require.Equal(t, 1, patches)
	require.Equal(t, 1, deletes)
}
