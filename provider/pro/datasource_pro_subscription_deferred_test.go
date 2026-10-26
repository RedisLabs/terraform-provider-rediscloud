package pro_test

import (
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// The subscription name is unknown at plan, so Terraform defers the read to apply.
// Computed object collections must then plan as unknown, not as empty lists.
const proSubscriptionDeferredReadConfig = `
resource "terraform_data" "name" {
  input = "deferred"
}

data "rediscloud_subscription" "example" {
  name = terraform_data.name.output
}

output "aws_account_id" {
  value = data.rediscloud_subscription.example.cloud_provider[0].aws_account_id
}

output "region" {
  value = one(data.rediscloud_subscription.example.cloud_provider[0].region).networks[0].networking_subnet_id
}

output "cloud_provider_count" {
  value = length(data.rediscloud_subscription.example.cloud_provider)
}

output "maintenance_mode" {
  value = data.rediscloud_subscription.example.maintenance_windows[0].mode
}

output "pricing_type" {
  value = data.rediscloud_subscription.example.pricing[0].type
}
`

func TestDataSourceRedisCloudProSubscription_deferredRead(t *testing.T) {
	// Configure needs credentials, but a deferred read never calls the API.
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             proSubscriptionDeferredReadConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectUnknownOutputValue("aws_account_id"),
						plancheck.ExpectUnknownOutputValue("region"),
						plancheck.ExpectUnknownOutputValue("cloud_provider_count"),
						plancheck.ExpectUnknownOutputValue("maintenance_mode"),
						plancheck.ExpectUnknownOutputValue("pricing_type"),
					},
				},
			},
		},
	})
}
