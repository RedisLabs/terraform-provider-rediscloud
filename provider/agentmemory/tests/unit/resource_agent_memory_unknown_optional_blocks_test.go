package unit_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryUnknownOptionalBlockPresence(t *testing.T) {
	for _, scenario := range []string{"models", "llm-credentials", "embedding-credentials", "summarization", "event-count", "exclusions", "semantic", "built-in", "custom", "matcher", "regex", "extraction", "timeouts"} {
		for _, present := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/present=%t", scenario, present), func(t *testing.T) {
				server := newOptionalBlockTestServer(t)
				defer server.Close()
				testProviderConfig(t, server.URL)
				variables := func(present bool, revision int) config.Variables {
					return config.Variables{
						"scenario": config.StringVariable(scenario), "present": config.BoolVariable(present), "revision": config.IntegerVariable(revision),
					}
				}
				action, id := plancheck.ResourceActionUpdate, "store-1"
				// Unknown model presence or strategy removal can require replacement. It must
				// already be visible in the saved plan, before the dependency is resolved.
				if scenario == "models" || scenario == "extraction" {
					action, id = plancheck.ResourceActionDestroyBeforeCreate, "store-2"
				}
				const address = "rediscloud_agent_memory.example"
				change := resource.TestStep{
					ConfigFile:       config.StaticFile("testdata/unknown_optional_blocks.tf"),
					ConfigVariables:  variables(present, 1),
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, action)}},
					Check:            resource.TestCheckResourceAttr(address, "id", id),
				}
				finalPresent := present
				// A configured detector requires a matcher; regex matchers require regex.
				// Reject a dynamically absent required child without changing the store.
				if !present && (scenario == "matcher" || scenario == "regex") {
					change.ConfigPlanChecks = resource.ConfigPlanChecks{}
					change.Check = nil
					change.ExpectError = regexp.MustCompile("Invalid Block|Invalid Attribute Value|Missing custom detector regex")
					finalPresent = true
				}
				steps := []resource.TestStep{
					{ConfigFile: config.StaticFile("testdata/unknown_optional_blocks.tf"), ConfigVariables: variables(true, 0), Check: resource.TestCheckResourceAttr(address, "id", "store-1")},
					change,
				}
				if change.ExpectError != nil {
					steps = append(steps, resource.TestStep{ConfigFile: config.StaticFile("testdata/unknown_optional_blocks.tf"), ConfigVariables: variables(true, 1), Check: resource.TestCheckResourceAttr(address, "id", "store-1")})
				}
				steps = append(steps, resource.TestStep{ConfigFile: config.StaticFile("testdata/unknown_optional_blocks.tf"), ConfigVariables: variables(finalPresent, 1), PlanOnly: true})
				resource.UnitTest(t, resource.TestCase{
					ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: steps,
				})
			})
		}
	}
}

// This fixture implements only control-plane CRUD; Terraform still saves and
// applies real plans so unknown block presence crosses the provider boundary.
func newOptionalBlockTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	var store map[string]any
	var lastID string
	creates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			if store != nil {
				t.Error("create must not orphan an existing fixture")
				http.Error(w, "fixture already exists", http.StatusConflict)
				return
			}
			if err := json.NewDecoder(r.Body).Decode(&store); err != nil {
				t.Errorf("decode create: %v", err)
				http.Error(w, "invalid fixture payload", http.StatusBadRequest)
				return
			}
			creates++
			lastID = fmt.Sprintf("store-%d", creates)
			store["storeId"], store["databaseId"], store["status"], store["extractionStrategy"] = lastID, "123", "READY", "INSTRUCT"
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
			_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": lastID}}})
		case r.Method == http.MethodGet && store != nil && r.URL.Path == "/memory-stores/"+lastID:
			_ = json.NewEncoder(w).Encode(store)
		case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+lastID:
			var patch map[string]any
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				t.Errorf("decode update: %v", err)
				http.Error(w, "invalid fixture payload", http.StatusBadRequest)
				return
			}
			for name, value := range patch {
				store[name] = value
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodDelete && store != nil && r.URL.Path == "/memory-stores/"+lastID:
			store = nil
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(func() { mu.Lock(); defer mu.Unlock(); assert.Nil(t, store, "fixture must be destroyed") })
	return server
}
