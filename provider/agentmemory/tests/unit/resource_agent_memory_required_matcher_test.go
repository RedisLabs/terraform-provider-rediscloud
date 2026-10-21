package unit_test

import (
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

func TestAgentMemoryRequiredMatcherBlocks(t *testing.T) {
	for _, tc := range []struct{ name, matcher, dependency, pattern string }{
		{name: "missing matcher", pattern: "Invalid Block"},
		{name: "missing regex", matcher: `matcher { kind="regex" }`, pattern: "Invalid Block"},
		{name: "unknown kind missing regex", matcher: `matcher { kind=terraform_data.value.output }`, dependency: `resource "terraform_data" "value" { input="regex" }`, pattern: "Invalid Block"},
		{name: "valid", matcher: "matcher {\nkind=\"regex\"\nregex { pattern=\"ticket-[0-9]+\" }\n}"},
		{name: "unknown pattern", matcher: "matcher {\nkind=\"regex\"\nregex { pattern=terraform_data.value.output }\n}", dependency: `resource "terraform_data" "value" { input="ticket-[0-9]+" }`},
	} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/enabled=%t", tc.name, enabled), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					http.Error(w, "validation must not call the API", 500)
				}))
				defer server.Close()
				config := testProviderConfig(t, server.URL) + tc.dependency + fmt.Sprintf(`
resource "rediscloud_agent_memory" "example" {
 name="required-matcher-store"
 database_id=123
 long_term_memory_exclusions {
  enabled=true
  custom_detectors {
   enabled=true
   detectors {
    name="ticket-detector"
    enabled=%t
    %s
   }
  }
 }
}
`, enabled, tc.matcher)
				step := resource.TestStep{Config: config, PlanOnly: true, ExpectNonEmptyPlan: true}
				if tc.pattern != "" {
					step.ExpectError = regexp.MustCompile(tc.pattern)
				}
				resource.UnitTest(t, resource.TestCase{ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(), Steps: []resource.TestStep{step}})
				assert.Zero(t, calls.Load(), "schema validation must not reach the API")
			})
		}
	}
}
