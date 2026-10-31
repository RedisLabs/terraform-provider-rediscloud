package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
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
			if store == nil {
				http.NotFound(w, r)
				return
			}
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
	testProviderConfig(t, server.URL)

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
			{
				ConfigFile: config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_all.tf"),
				Check:      check,
			},
			{
				ConfigFile: config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_semantic.tf"),
				Check:      check,
			},
			{
				ConfigFile: config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_semantic.tf"),
				PlanOnly:   true,
			},
			{
				ConfigFile: config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_omitted.tf"),
				Check:      check,
			},
			{
				ConfigFile: config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_omitted.tf"),
				PlanOnly:   true,
			},
			{
				ConfigFile:    config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_omitted.tf"),
				ResourceName:  "rediscloud_agent_memory.example",
				ImportState:   true,
				ImportStateId: "store-1",
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("unexpected imported resource count")
					}
					for _, group := range []string{"semantic.0.prompt", "built_in_detectors.0.detectors.0.id", "custom_detectors.0.detectors.0.name"} {
						if states[0].Attributes["long_term_memory_exclusions.0."+group] == "" {
							return fmt.Errorf("import omitted stored group %s", group)
						}
					}
					return nil
				},
			},
			{
				ConfigFile: config.StaticFile("testdata/lifecycle/exclusions/exclusion_groups_omitted.tf"),
				PlanOnly:   true,
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || deletes != 1 || store != nil {
		t.Fatal("fixture cleanup failed")
	}
}
