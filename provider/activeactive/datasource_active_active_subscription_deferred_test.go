package activeactive_test

import (
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// The subscription name is unknown at plan, so Terraform defers the read to apply.
// Computed object collections must then plan as unknown, not as empty lists.
const activeActiveSubscriptionDeferredReadConfig = `
resource "terraform_data" "name" {
  input = "deferred"
}

data "rediscloud_active_active_subscription" "example" {
  name = terraform_data.name.output
}

output "maintenance_mode" {
  value = data.rediscloud_active_active_subscription.example.maintenance_windows[0].mode
}

output "pricing_type" {
  value = data.rediscloud_active_active_subscription.example.pricing[0].type
}
`

func TestDataSourceRedisCloudActiveActiveSubscription_deferredRead(t *testing.T) {
	// Configure needs credentials, but a deferred read never calls the API.
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             activeActiveSubscriptionDeferredReadConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectUnknownOutputValue("maintenance_mode"),
						plancheck.ExpectUnknownOutputValue("pricing_type"),
					},
				},
			},
		},
	})
}
