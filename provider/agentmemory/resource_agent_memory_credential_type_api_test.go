package agentmemory_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryCredentialTypeValidationByAPI(t *testing.T) {
	for _, selector := range credentialTypeCases {
		for _, deferred := range []bool{false, true} {
			name := selector.block + "/known"
			expression, dependency := `"unsupported"`, ""
			if deferred {
				name = selector.block + "/unknown"
				expression = `tostring(terraform_data.selector.output)`
				dependency = `resource "terraform_data" "selector" { input="unsupported" }`
			}
			t.Run(name, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/memory-stores", r.URL.Path)
					var payload map[string]any
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
					var model map[string]any
					if selector.block == "llm" {
						model, _ = payload["llm"].(map[string]any)
					} else {
						memory, _ := payload["longTermMemory"].(map[string]any)
						model, _ = memory["embedding"].(map[string]any)
					}
					credentials, _ := model["credentials"].(map[string]any)
					assert.Equal(t, "unsupported", credentials["type"])
					assert.Equal(t, "FAKE-STATIC", credentials["apiKey"])
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"message":"Credential type rejected"}`))
				}))
				defer server.Close()
				config := testProviderConfig(t, server.URL) + credentialTypeConfig(selector.block, selector.field, expression, dependency)
				resource.UnitTest(t, resource.TestCase{
					ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
					Steps: []resource.TestStep{
						{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
						{Config: config, ExpectError: regexp.MustCompile(`(?s)Failed to create Agent Memory service.*Credential\s+type\s+rejected`)},
					},
				})
				assert.Equal(t, int32(1), calls.Load())
			})
		}
	}
}
