package agentmemory_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAgentMemoryTimeoutValidationBeforeAPI(t *testing.T) {
	for _, kind := range []string{"store", "key"} {
		for _, operation := range []string{"create", "read", "update", "delete"} {
			t.Run(kind+"/"+operation, func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					w.WriteHeader(http.StatusInternalServerError)
				}))
				defer server.Close()
				resourceType, arguments := "rediscloud_agent_memory", "name=\"validation\"\ndatabase_id=123"
				expected := "Invalid Attribute Value|invalid duration"
				if kind == "key" {
					resourceType, arguments = "rediscloud_agent_memory_api_key", "name=\"runtime\"\nstore_id=\"store-1\""
					if operation == "update" {
						expected = "Unsupported argument"
					}
				}
				config := testProviderConfig(t, server.URL) + fmt.Sprintf(`
resource %q "example" {
  %s
  timeouts {
    %s = "not-a-duration"
  }
}
`, resourceType, arguments, operation)
				resource.UnitTest(t, resource.TestCase{
					ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
					Steps:                    []resource.TestStep{{Config: config, PlanOnly: true, ExpectError: regexp.MustCompile(expected)}},
				})
				if requests.Load() != 0 {
					t.Error("invalid timeout configuration reached the API")
				}
			})
		}
	}
}
