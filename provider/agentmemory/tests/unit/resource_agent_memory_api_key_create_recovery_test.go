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

func TestAgentMemoryAPIKeyCreateFailurePreservesParent(t *testing.T) {
	var mu sync.Mutex
	creates, keyPosts := 0, 0
	keyCreated := false
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
		case "GET /memory-stores/store-1":
			writeStoreResponse(t, w, "key-failure-parent", 86400, 31536000, 300)
		case "POST /memory-stores/store-1/api-keys":
			keyPosts++
			if keyPosts == 1 {
				w.WriteHeader(500)
				_, _ = w.Write([]byte(`{"message":"simulated rejection before key creation"}`))
				return
			}
			keyCreated = true
			_, _ = w.Write([]byte(`{"apiKey":"fake-review-key"}`))
		case "GET /memory-stores/store-1/api-keys":
			if keyCreated {
				_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"runtime","obfuscatedToken":"fake...key","createdAt":1790078409}]}`))
			} else {
				_, _ = w.Write([]byte(`{"apiKeys":[]}`))
			}
		case "DELETE /memory-stores/store-1/api-keys/key-1":
			keyCreated = false
			w.WriteHeader(204)
		case "DELETE /memory-stores/store-1":
			_, _ = w.Write([]byte(`{"taskId":"delete-task"}`))
		case "GET /tasks/delete-task":
			_, _ = w.Write([]byte(`{"taskId":"delete-task","status":"processing-completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + `
resource "rediscloud_agent_memory" "example" {
 name="key-failure-parent"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
}
resource "rediscloud_agent_memory_api_key" "runtime" {
 store_id=rediscloud_agent_memory.example.id
 name="runtime"
}`
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: config, ExpectError: regexp.MustCompile("Failed to create Agent Memory API key")},
		// Retry must reuse the stored parent identity after the key creation failure.
		{Config: config, Check: func(state *terraform.State) error {
			mu.Lock()
			defer mu.Unlock()
			if creates != 1 || keyPosts != 2 {
				return fmt.Errorf("unexpected parent/key retries: store creates=%d key attempts=%d", creates, keyPosts)
			}
			return resource.ComposeAggregateTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.runtime", "id", "key-1"), resource.TestCheckResourceAttr("rediscloud_agent_memory_api_key.runtime", "api_key", "fake-review-key"))(state)
		}},
	}})
}
