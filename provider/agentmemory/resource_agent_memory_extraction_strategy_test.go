package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryExtractionStrategyLifecycle(t *testing.T) {
	var mu sync.Mutex
	var store map[string]any
	creates, patches, deletes := 0, 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			creates++
			strategy, present := payload["extractionStrategy"]
			if !present {
				strategy = "INSTRUCT"
			}
			store = map[string]any{
				"storeId": fmt.Sprintf("store-%d", creates), "name": payload["name"], "databaseId": "123", "status": "READY",
				"extractionStrategy": strategy, "shortMemory": map[string]any{"ttlSeconds": 86400},
				"longTermMemory": map[string]any{"ttlSeconds": 31536000}, "extractionCadence": map[string]any{"activeIntervalSeconds": 300},
			}
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodPatch && store != nil && r.URL.Path == "/memory-stores/"+store["storeId"].(string):
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			assert.NotContains(t, payload, "extractionStrategy")
			patches++
			store["name"] = payload["name"]
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
			id := "store-1"
			if store != nil {
				id = store["storeId"].(string)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "fixture-task", "status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": id}}})
		case r.Method == http.MethodGet && store != nil && r.URL.Path == "/memory-stores/"+store["storeId"].(string):
			_ = json.NewEncoder(w).Encode(store)
		case r.Method == http.MethodDelete && store != nil && r.URL.Path == "/memory-stores/"+store["storeId"].(string):
			deletes++
			store = nil
			_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := func(name, strategy string) string {
		attribute := ""
		if strategy != "" {
			attribute = fmt.Sprintf("extraction_strategy = %q", strategy)
		}
		return testProviderConfig(t, server.URL) + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name = %q
 database_id = 123
 %s
}
`, name, attribute)
	}
	const address = "rediscloud_agent_memory.example"
	initial, renamed := config("store", "FAKE-FUTURE-STRATEGY"), config("renamed", "FAKE-FUTURE-STRATEGY")
	changed := config("renamed", "INSTRUCT")
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{Config: initial, Check: resource.TestCheckResourceAttr(address, "extraction_strategy", "FAKE-FUTURE-STRATEGY")},
			{Config: initial, PlanOnly: true},
			{Config: renamed, Check: resource.TestCheckResourceAttr(address, "id", "store-1")},
			{Config: config("renamed", ""), PlanOnly: true},
			{Config: changed, ExpectError: regexp.MustCompile("strategy change requires an explicit migration")},
			{Config: renamed, PlanOnly: true},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, creates)
	assert.Equal(t, 1, patches)
	assert.Equal(t, 1, deletes)
	assert.Nil(t, store)
}

func TestAgentMemoryExtractionStrategyRejectsEmptyString(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{{
			Config: testProviderConfig(t, "http://127.0.0.1:1") + `
resource "rediscloud_agent_memory" "example" {
 name = "store"
 database_id = 123
 extraction_strategy = ""
}
`, PlanOnly: true, ExpectError: regexp.MustCompile("at least 1"),
		}},
	})
}
