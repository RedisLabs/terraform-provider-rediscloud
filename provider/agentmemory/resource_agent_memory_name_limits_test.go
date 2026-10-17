package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryNameLimitsAreValidatedByAPI(t *testing.T) {
	for _, tc := range []struct {
		name         string
		storeID      string
		resourceType string
		resourceName string
		errorTitle   string
	}{
		{name: "store name", resourceType: "rediscloud_agent_memory", resourceName: strings.Repeat("a", 65), errorTitle: "Failed to create Agent Memory service"},
		{name: "API key name", resourceType: "rediscloud_agent_memory_api_key", storeID: "store-1", resourceName: strings.Repeat("a", 101), errorTitle: "Failed to create Agent Memory API key"},
		{name: "API key store ID", resourceType: "rediscloud_agent_memory_api_key", storeID: strings.Repeat("a", 65), resourceName: "runtime", errorTitle: "Failed to create Agent Memory API key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tc.storeID != "" && r.Method == http.MethodGet {
					assert.Equal(t, "/memory-stores/"+tc.storeID+"/api-keys", r.URL.Path)
					_, _ = w.Write([]byte(`{"apiKeys":[]}`))
					return
				}
				requests.Add(1)
				assert.Equal(t, http.MethodPost, r.Method)
				expectedPath := "/memory-stores"
				if tc.storeID != "" {
					expectedPath += "/" + tc.storeID + "/api-keys"
				}
				assert.Equal(t, expectedPath, r.URL.Path)
				var body map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				nameField := "name"
				if tc.storeID != "" {
					nameField = "keyName"
				}
				assert.Equal(t, tc.resourceName, body[nameField])
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"Name or identifier exceeds the API limit"}`))
			}))
			defer server.Close()

			attributes := "database_id = 123"
			if tc.storeID != "" {
				attributes = fmt.Sprintf("store_id = %q", tc.storeID)
			}
			config := testProviderConfig(t, server.URL) + fmt.Sprintf(`
resource %q "example" {
 name = %q
 %s
}
`, tc.resourceType, tc.resourceName, attributes)
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config, ExpectError: regexp.MustCompile(tc.errorTitle)},
				},
			})
			assert.Equal(t, int32(1), requests.Load())
		})
	}
}
