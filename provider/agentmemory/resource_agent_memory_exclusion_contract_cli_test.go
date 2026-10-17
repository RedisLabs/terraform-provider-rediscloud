package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func exclusionListsConfig(enabled, entries bool) string {
	builtin, custom := "", ""
	if entries {
		builtin = "detectors {\nid=\"email\"\nenabled=true\naction=\"redact\"\n}"
		custom = "detectors {\nname=\"ticket\"\nenabled=true\naction=\"redact\"\nmatcher {\nkind=\"regex\"\nregex { pattern=\"ticket+\" }\n}\n}"
	}
	return fmt.Sprintf(`resource "rediscloud_agent_memory" "example" {
 name="exclusion-lists"
 database_id=123
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
 long_term_memory_exclusions {
  enabled=true
  built_in_detectors {
   enabled=%t
   %s
  }
  custom_detectors {
   enabled=%t
   %s
  }
 }
}`, enabled, builtin, enabled, custom)
}

func TestAgentMemoryExclusionListsLifecycle(t *testing.T) {
	var mu sync.Mutex
	var store map[string]any
	creates, patches, deletes := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores", r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if store == nil {
				creates++
				store = payload
			} else {
				patches++
				policy, _ := payload["longTermMemoryExclusions"].(map[string]any)
				if policy == nil {
					t.Error("missing policy")
					w.WriteHeader(400)
					return
				}
				for _, key := range []string{"builtInDetectors", "customDetectors"} {
					group, _ := policy[key].(map[string]any)
					list, present := group["detectors"]
					if !present || list == nil {
						t.Error("clear request omitted the explicit empty list")
						w.WriteHeader(400)
						return
					}
					entries, _ := list.([]any)
					if len(entries) != 0 {
						t.Error("clear request retained detector entries")
						w.WriteHeader(400)
						return
					}
				}
				store["longTermMemoryExclusions"] = policy
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
			_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && store != nil:
			store["storeId"] = "store-1"
			store["databaseId"] = "123"
			store["status"] = "READY"
			store["extractionStrategy"] = "INSTRUCT"
			_ = json.NewEncoder(w).Encode(store)
		case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
			deletes++
			store = nil
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	prefix := testProviderConfig(t, server.URL)
	resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
		{Config: prefix + exclusionListsConfig(false, true), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.built_in_detectors.detectors.0.id", "email"), resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.detectors.0.name", "ticket"))},
		{Config: prefix + exclusionListsConfig(true, false), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1"), resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.built_in_detectors.detectors.0.id"), resource.TestCheckNoResourceAttr("rediscloud_agent_memory.example", "long_term_memory_exclusions.custom_detectors.detectors.0.name"))},
		{Config: prefix + exclusionListsConfig(true, false), PlanOnly: true},
	}})
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, creates)
	require.Equal(t, 1, patches)
	require.Equal(t, 1, deletes)
	require.Nil(t, store)
}
