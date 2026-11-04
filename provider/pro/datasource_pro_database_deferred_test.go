package pro_test

import (
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// The subscription id is unknown at plan, so Terraform defers the read to apply.
// Computed object collections must then plan as unknown, not as empty sets.
const proDatabaseDeferredReadConfig = `
resource "terraform_data" "subscription_id" {
  input = "1"
}

data "rediscloud_database" "example" {
  subscription_id = terraform_data.subscription_id.output
  name            = "deferred"
}

output "alert_count" {
  value = length(data.rediscloud_database.example.alert)
}

output "module_names" {
  value = data.rediscloud_database.example.module[*].name
}

output "backup_status" {
  value = one(data.rediscloud_database.example.latest_backup_status)
}

output "tags" {
  value = data.rediscloud_database.example.tags
}
`

func TestDataSourceRedisCloudProDatabase_deferredRead(t *testing.T) {
	// Configure needs credentials, but a deferred read never calls the API.
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             proDatabaseDeferredReadConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectUnknownOutputValue("alert_count"),
						plancheck.ExpectUnknownOutputValue("module_names"),
						plancheck.ExpectUnknownOutputValue("backup_status"),
						plancheck.ExpectUnknownOutputValue("tags"),
					},
				},
			},
		},
	})
}
