package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAccResourceRedisCloudAclRole_CRUDI(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	tagVariables := config.Variables{
		"team":  config.StringVariable(resourceTags.Team),
		"owner": config.StringVariable(resourceTags.Owner),
	}

	prefix := testRandomWithPrefix()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabasePassword := prefix + "aA.1"
	exampleRuleName := prefix + "-rule"

	testRoleName := prefix + "-test-role"
	testRoleNameUpdated := testRoleName + "-updated"

	subAndDb := fmt.Sprintf(testAccResourceRedisCloudProDatabaseAcl,
		cloudAccountName, exampleSubscriptionName, exampleDatabasePassword)

	testCreateTerraform := subAndDb +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleName)

	testUpdateTerraform := subAndDb +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleNameUpdated)

	const testAclRole = "rediscloud_acl_role.test"
	const testAclRoleData = "data.rediscloud_acl_role.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		// Sometimes after deletion, the entity 'flickers'
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			// Test role creation including association with database
			{
				ConfigVariables: tagVariables,
				Config:          testCreateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(testAclRole, "name", testRoleName),
					resource.TestCheckResourceAttr(testAclRole, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.0.regions.#", "0"),

					// Test role exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources[testAclRole]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the role ID: %s", redis.StringValue(&r.Primary.ID))
						}

						apiClient := client.MustTestClient(t)
						role, err := apiClient.Client.Roles.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(role.Name) != testRoleName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(role.Name))
						}

						return nil
					},

					// Test the datasource
					resource.TestMatchResourceAttr(
						testAclRoleData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "name", testRoleName),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test role update
			{
				ConfigVariables: tagVariables,
				Config:          testUpdateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(testAclRole, "name", testRoleNameUpdated),
					resource.TestCheckResourceAttr(testAclRole, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.0.regions.#", "0"),

					// Test the datasource
					resource.TestMatchResourceAttr(
						testAclRoleData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "name", testRoleNameUpdated),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test that the role is imported successfully
			{
				Config:            fmt.Sprintf(testRole, testRoleNameUpdated),
				ResourceName:      testAclRole,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const referencableRule = `
resource "rediscloud_acl_rule" "example" {
    name = "%s"
    rule = "+@all"
}
`

const testRole = `
resource "rediscloud_acl_role" "test" {
	name = "%s"
	rule {
		name = rediscloud_acl_rule.example.name
		database {
			subscription = rediscloud_subscription.example.id
			database = rediscloud_subscription_database.example.db_id
		}
	}
}

data "rediscloud_acl_role" "test" {
	name = rediscloud_acl_role.test.name
}
`

const testAccResourceRedisCloudProDatabaseAcl = `
variable "team" {
  type = string
}

variable "owner" {
  type = string
}

data "rediscloud_payment_method" "card" {
  card_type = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider {
    resource_tags = {
      team  = var.team
      owner = var.owner
    }

    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
  }
}

resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
    name = "example"
    protocol = "redis"
    dataset_size_in_gb = 3
    data_persistence = "none"
    data_eviction = "allkeys-random"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    password = "%s"
    support_oss_cluster_api = false
    external_endpoint_for_oss_cluster_api = false
    replication = false
    average_item_size_in_bytes = 0
    client_ssl_certificate = ""
    periodic_backup_path = ""
	enable_default_user = true
    redis_version = "7.4"

    alert {
        name = "dataset-size"
        value = 1
    }

    modules = [
        {
          name = "RedisBloom"
        }
    ]

	tags = {
		"market" = "emea"
		"material" = "cardboard"
	}
}
`
