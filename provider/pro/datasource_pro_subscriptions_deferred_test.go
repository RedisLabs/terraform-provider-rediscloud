package pro_test

import (
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// The unknown filter defers the data source read, so computed subscription values
// must remain unknown for downstream references.
const proSubscriptionsDeferredReadConfig = `
resource "terraform_data" "name" {
  input = "deferred"
}

data "rediscloud_subscriptions" "example" {
  filters {
    name = terraform_data.name.output
  }
}

output "subscription_name" {
  value = data.rediscloud_subscriptions.example.subscriptions[0].name
}

output "aws_account_id" {
  value = data.rediscloud_subscriptions.example.subscriptions[0].cloud_provider[0].aws_account_id
}

output "subscription_count" {
  value = length(data.rediscloud_subscriptions.example.subscriptions)
}
`

func TestDataSourceRedisCloudProSubscriptions_deferredRead(t *testing.T) {
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             proSubscriptionsDeferredReadConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectUnknownOutputValue("subscription_name"),
						plancheck.ExpectUnknownOutputValue("aws_account_id"),
						plancheck.ExpectUnknownOutputValue("subscription_count"),
					},
				},
			},
		},
	})
}
