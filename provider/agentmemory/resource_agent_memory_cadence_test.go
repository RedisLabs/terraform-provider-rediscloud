package agentmemory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryCadenceValidationByAPI(t *testing.T) {
	for _, tc := range []struct {
		name       string
		expression string
		dependency string
		value      int
	}{
		{name: "zero", expression: "0", value: 0},
		{name: "negative", expression: "-1", value: -1},
		{name: "below previous minimum", expression: "30", value: 30},
		{name: "above previous maximum", expression: "601", value: 601},
		{name: "unknown", expression: "terraform_data.cadence.output", dependency: `resource "terraform_data" "cadence" { input=0 }`, value: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/memory-stores", r.URL.Path)
				var body map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, map[string]any{"activeIntervalSeconds": float64(tc.value)}, body["extractionCadence"])
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"Cadence rejected by the API"}`))
			}))
			defer server.Close()
			config := testProviderConfig(t, server.URL) + tc.dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="store"
 database_id=123
 extraction_cadence_seconds=%s
}
`, tc.expression)
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Steps: []resource.TestStep{
					{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true},
					{Config: config, ExpectError: regexp.MustCompile(`(?s)Failed to create Agent Memory service.*Cadence rejected by\s+the API`)},
				},
			})
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}
