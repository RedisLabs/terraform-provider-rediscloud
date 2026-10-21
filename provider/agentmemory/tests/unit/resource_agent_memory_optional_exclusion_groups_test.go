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

func TestAgentMemoryOptionalExclusionGroupsLifecycle(t *testing.T) {
	var mu sync.Mutex
	var store map[string]any
	creates, deletes := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /memory-stores":
			creates++
			if err := json.NewDecoder(r.Body).Decode(&store); err != nil {
				t.Error(err)
				return
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case "PATCH /memory-stores/store-1":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
				return
			}
			if requested, ok := payload["longTermMemoryExclusions"].(map[string]any); ok {
				old := store["longTermMemoryExclusions"].(map[string]any)
				// CAPI documents that omitted nested policy groups are preserved.
				for key, value := range requested {
					old[key] = value
				}
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case "GET /tasks/fixture-task":
			_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
		case "GET /memory-stores/store-1":
			store["storeId"], store["databaseId"], store["status"] = "store-1", "123", "READY"
			_ = json.NewEncoder(w).Encode(store)
		case "DELETE /memory-stores/store-1":
			deletes++
			store = nil
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	prefix := testProviderConfig(t, server.URL)
	initial := prefix + `
resource "rediscloud_agent_memory" "example" {
 name="remove-groups"
 database_id=123
 long_term_memory_exclusions {
  enabled=true
  semantic {
   enabled=true
   prompt="Exclude secrets"
  }
  built_in_detectors {
   enabled=true
   detectors {
    id="email"
    enabled=true
   }
  }
  custom_detectors {
   enabled=true
   detectors {
    name="ticket"
    enabled=true
    matcher {
     kind="regex"
     regex { pattern="ticket-[0-9]+" }
    }
   }
  }
 }
}
`
	removed := prefix + `
resource "rediscloud_agent_memory" "example" {
 name="remove-groups"
 database_id=123
 long_term_memory_exclusions { enabled=true }
}
`

	partial := prefix + `resource "rediscloud_agent_memory" "example" {
 name="remove-groups"
 database_id=123
 long_term_memory_exclusions {
  enabled=true
  semantic {
   enabled=true
   prompt="Exclude secrets"
  }
 }
}`
	check := func(state *terraform.State) error {
		mu.Lock()
		defer mu.Unlock()
		if creates != 1 || deletes != 0 {
			return fmt.Errorf("store was replaced")
		}
		policy := store["longTermMemoryExclusions"].(map[string]any)
		for _, group := range []string{"semantic", "builtInDetectors", "customDetectors"} {
			if policy[group] == nil {
				return fmt.Errorf("stored group %s was removed", group)
			}
		}
		return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")(state)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: initial, Check: check},
			{Config: partial, Check: check},
			{Config: partial, PlanOnly: true},
			{Config: removed, Check: check},
			{Config: removed, PlanOnly: true},
			{Config: removed, ResourceName: "rediscloud_agent_memory.example", ImportState: true, ImportStateId: "store-1", ImportStateCheck: func(states []*terraform.InstanceState) error {
				if len(states) != 1 {
					return fmt.Errorf("unexpected imported resource count")
				}
				for _, group := range []string{"semantic.prompt", "built_in_detectors.detectors.0.id", "custom_detectors.detectors.0.name"} {
					if states[0].Attributes["long_term_memory_exclusions."+group] == "" {
						return fmt.Errorf("import omitted stored group %s", group)
					}
				}
				return nil
			}},
			{Config: removed, PlanOnly: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || deletes != 1 || store != nil {
		t.Fatal("fixture cleanup failed")
	}
}
