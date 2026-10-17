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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryStoreIdentityPreserved(t *testing.T) {
	for _, tc := range []struct{ name, database, strategy, dependency, errorPattern string }{
		{name: "database change", database: "456", strategy: `"INSTRUCT"`, errorPattern: "database change requires an explicit migration"},
		{name: "unknown database change", database: "terraform_data.value.output", strategy: `"INSTRUCT"`, dependency: `resource "terraform_data" "value" { input=456 }`, errorPattern: "database change requires an explicit migration"},
		{name: "strategy change", database: "123", strategy: `"OTHER"`, errorPattern: "strategy change requires an explicit migration"},
		{name: "unknown strategy change", database: "123", strategy: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="OTHER" }`, errorPattern: "strategy change requires an explicit migration"},
		{name: "unknown database unchanged", database: "terraform_data.value.output", strategy: `"INSTRUCT"`, dependency: `resource "terraform_data" "value" { input=123 }`},
		{name: "unknown strategy unchanged", database: "123", strategy: "terraform_data.value.output", dependency: `resource "terraform_data" "value" { input="INSTRUCT" }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			creates, patches, deletes := 0, 0, 0
			live := false
			contextValue := "existing-agent-context"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
					creates++
					live = true
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
					var payload map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					assert.NotContains(t, payload, "databaseId")
					assert.NotContains(t, payload, "extractionStrategy")
					patches++
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/tasks/fixture-task":
					_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
				case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1" && live:
					writeStoreResponse(t, w, "store", 86400, 31536000, 300)
				case r.Method == http.MethodDelete && r.URL.Path == "/memory-stores/store-1":
					deletes++
					live = false
					contextValue = ""
					_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := func(database, strategy, dependency string) string {
				return testProviderConfig(t, server.URL) + dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="store"
 database_id=%s
 extraction_strategy=%s
 short_term_ttl_seconds=86400
 long_term_ttl_seconds=31536000
 extraction_cadence_seconds=300
}
`, database, strategy)
			}
			initial := config("123", `"INSTRUCT"`, "")
			changed := config(tc.database, tc.strategy, tc.dependency)
			check := func(state *terraform.State) error {
				mu.Lock()
				defer mu.Unlock()
				if creates != 1 || deletes != 0 || !live || contextValue != "existing-agent-context" {
					return fmt.Errorf("store identity/context changed: creates=%d deletes=%d live=%t", creates, deletes, live)
				}
				if tc.errorPattern != "" && patches != 0 {
					return fmt.Errorf("rejected change reached update API")
				}
				return resource.TestCheckResourceAttr("rediscloud_agent_memory.example", "id", "store-1")(state)
			}
			attempt := resource.TestStep{Config: changed, Check: check}
			if tc.errorPattern != "" {
				attempt.ExpectError = regexp.MustCompile(tc.errorPattern)
			}
			resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{
				{Config: initial, Check: check}, attempt, {Config: initial, Check: check}, {Config: initial, PlanOnly: true},
			}})
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, 1, creates)
			assert.Equal(t, 1, deletes)
			assert.False(t, live)
		})
	}
}
