package unit_test

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

func TestAgentMemoryLimitsAreValidatedByAPI(t *testing.T) {
	for _, tc := range []struct {
		name            string
		databaseID      int
		storeID         string
		resourceType    string
		resourceName    string
		errorTitle      string
		ttlAttribute    string
		ttlValue        int
		memoryField     string
		strategy        string
		unknownStrategy bool
	}{
		{name: "store name", databaseID: 123, resourceType: "rediscloud_agent_memory", resourceName: strings.Repeat("a", 65), errorTitle: "Failed to create Agent Memory service"},
		{name: "zero database ID", resourceType: "rediscloud_agent_memory", databaseID: 0, resourceName: "store", errorTitle: "Failed to create Agent Memory service"},
		{name: "negative database ID", resourceType: "rediscloud_agent_memory", databaseID: -1, resourceName: "store", errorTitle: "Failed to create Agent Memory service"},
		{name: "short_term_ttl_seconds zero", resourceType: "rediscloud_agent_memory", databaseID: 123, resourceName: "store", errorTitle: "Failed to create Agent Memory service", ttlAttribute: "short_term_ttl_seconds", ttlValue: 0, memoryField: "shortMemory"},
		{name: "short_term_ttl_seconds negative", resourceType: "rediscloud_agent_memory", databaseID: 123, resourceName: "store", errorTitle: "Failed to create Agent Memory service", ttlAttribute: "short_term_ttl_seconds", ttlValue: -1, memoryField: "shortMemory"},
		{name: "short_term_ttl_seconds above previous maximum", resourceType: "rediscloud_agent_memory", databaseID: 123, resourceName: "store", errorTitle: "Failed to create Agent Memory service", ttlAttribute: "short_term_ttl_seconds", ttlValue: 31536001, memoryField: "shortMemory"},
		{name: "long_term_ttl_seconds zero", resourceType: "rediscloud_agent_memory", databaseID: 123, resourceName: "store", errorTitle: "Failed to create Agent Memory service", ttlAttribute: "long_term_ttl_seconds", ttlValue: 0, memoryField: "longTermMemory"},
		{name: "long_term_ttl_seconds negative", resourceType: "rediscloud_agent_memory", databaseID: 123, resourceName: "store", errorTitle: "Failed to create Agent Memory service", ttlAttribute: "long_term_ttl_seconds", ttlValue: -1, memoryField: "longTermMemory"},
		{name: "long_term_ttl_seconds above previous maximum", resourceType: "rediscloud_agent_memory", databaseID: 123, resourceName: "store", errorTitle: "Failed to create Agent Memory service", ttlAttribute: "long_term_ttl_seconds", ttlValue: 31536001, memoryField: "longTermMemory"},
		{name: "unrecognised extraction strategy", databaseID: 123, resourceType: "rediscloud_agent_memory", resourceName: "store", strategy: "FAKE-FUTURE-STRATEGY", errorTitle: "Failed to create Agent Memory service"},
		{name: "unknown extraction strategy", databaseID: 123, resourceType: "rediscloud_agent_memory", resourceName: "store", strategy: "FAKE-FUTURE-STRATEGY", unknownStrategy: true, errorTitle: "Failed to create Agent Memory service"},
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
				if tc.storeID == "" {
					assert.Equal(t, float64(tc.databaseID), body["databaseId"])
				}
				if tc.ttlAttribute != "" {
					assert.Equal(t, map[string]any{"ttlSeconds": float64(tc.ttlValue)}, body[tc.memoryField])
				}
				if tc.strategy != "" {
					assert.Equal(t, tc.strategy, body["extractionStrategy"])
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"Value rejected by the API"}`))
			}))
			defer server.Close()

			attributes := fmt.Sprintf("database_id = %d", tc.databaseID)
			if tc.ttlAttribute != "" {
				attributes += fmt.Sprintf("\n%s = %d", tc.ttlAttribute, tc.ttlValue)
			}
			if tc.storeID != "" {
				attributes = fmt.Sprintf("store_id = %q", tc.storeID)
			}
			dependency := ""
			if tc.strategy != "" {
				expression := fmt.Sprintf("%q", tc.strategy)
				if tc.unknownStrategy {
					dependency = fmt.Sprintf(`resource "terraform_data" "strategy" { input = %q }`, tc.strategy)
					expression = "terraform_data.strategy.output"
				}
				attributes += "\nextraction_strategy = " + expression
			}
			config := testProviderConfig(t, server.URL) + dependency + fmt.Sprintf(`
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
