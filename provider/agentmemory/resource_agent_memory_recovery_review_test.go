package agentmemory_test

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

// An accepted create followed by invalid read data must retain the returned ID.
// A retry may replace the tainted resource, but must delete it before recreating.
func TestReviewCreateMalformedReadKeepsResourceID(t *testing.T) {
	var mu sync.Mutex
	creates := 0
	deletes := 0
	malformed := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			creates++
			_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "create-task"})
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/create-task":
			_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "create-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": fmt.Sprintf("store-%d", creates)}}})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/memory-stores/"):
			database := "123"
			if malformed {
				database = "not-a-number"
				malformed = false
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"storeId": strings.TrimPrefix(r.URL.Path, "/memory-stores/"), "name": "recovery-store", "databaseId": database, "status": "READY", "shortMemory": map[string]any{"ttlSeconds": 86400}, "longTermMemory": map[string]any{"ttlSeconds": 31536000}, "extractionCadence": map[string]any{"activeIntervalSeconds": 300}, "extractionStrategy": "INSTRUCT", "endpoint": "https://memory.example.test", "createdAt": "2026-09-22T12:00:00Z"})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/memory-stores/"):
			deletes++
			_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "delete-task"})
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/delete-task":
			_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "delete-task", "status": "processing-completed"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := testProviderConfig(t, server.URL) + `resource "rediscloud_agent_memory" "example" {
 name="recovery-store"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 }`
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config, ExpectError: regexp.MustCompile("Unexpected Agent Memory database ID")},
			{Config: config, Check: func(s *terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if creates-deletes != 1 {
					return fmt.Errorf("REPRODUCED: %d creates and %d deletes after malformed read and retry; expected one live store, not an orphan", creates, deletes)
				}
				t.Logf("Tracked recovery: creates=%d deletes=%d live=%d", creates, deletes, creates-deletes)
				return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", fmt.Sprintf("store-%d", creates))(s)
			}},
		},
	})
}
