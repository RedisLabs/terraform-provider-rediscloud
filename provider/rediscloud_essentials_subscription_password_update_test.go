package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"regexp"
	"testing"
)

func TestAccResourceRedisCloudEssentialsDatabase_DisableDefaultUser(t *testing.T) {
	//testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)
	databaseName := subscriptionName + "-db"
	databaseNameUpdated := databaseName + "-updated"

	const resourceName = "rediscloud_essentials_database.example"
	const datasourceName = "data.rediscloud_essentials_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseDisableDefaultUserCreate, subscriptionName, databaseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test creating resource
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "db_id"),
					resource.TestCheckResourceAttr(resourceName, "name", databaseName),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "password", "j43589rhe39f"),

					// Test the datasource
					resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", databaseName),
					resource.TestCheckResourceAttr(datasourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(datasourceName, "password", "j43589rhe39f"),
				),
			},
			{
				// test update
				Config: fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseDisableDefaultUserUpdate, subscriptionName, databaseNameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestMatchResourceAttr(resourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "db_id"),
					resource.TestCheckResourceAttr(resourceName, "name", databaseNameUpdated),
					resource.TestCheckResourceAttr(resourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "password", "j43589rhe39f"),

					// Test the datasource
					resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile("^\\d+/\\d+$")),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", databaseNameUpdated),
					resource.TestCheckResourceAttr(datasourceName, "enable_default_user", "false"),
					resource.TestCheckResourceAttr(datasourceName, "password", ""),
				),
			},
			{
				Config:                  fmt.Sprintf(testAccResourceRedisCloudEssentialsDatabaseDisableDefaultUserUpdate, subscriptionName, databaseName),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password", "enable_payg_features"},
			},
		},
	})
}

const testAccResourceRedisCloudEssentialsDatabaseDisableDefaultUserCreate = `

data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

data "rediscloud_essentials_plan" "example" {
  name = "Single-Zone_1GB"
  cloud_provider = "AWS"
  region = "eu-west-1"
}

data "rediscloud_essentials_database" "example" {
	subscription_id = rediscloud_essentials_subscription.example.id
	name = rediscloud_essentials_database.example.name
}

resource "rediscloud_essentials_subscription" "example" {
  name = "%s"
  plan_id = data.rediscloud_essentials_plan.example.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id     = rediscloud_essentials_subscription.example.id
  name                = "%s"
  enable_default_user = true
  password            = "j43589rhe39f"

  data_persistence = "none"
  replication      = false

  alert {
    name  = "throughput-higher-than"
    value = 80
  }
  tags = {
    "envaaaa" = "qaaaa"
  }
}
`

const testAccResourceRedisCloudEssentialsDatabaseDisableDefaultUserUpdate = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}

data "rediscloud_essentials_plan" "example" {
  name = "Single-Zone_1GB"
  cloud_provider = "AWS"
  region = "eu-west-1"
}

data "rediscloud_essentials_database" "example" {
	subscription_id = rediscloud_essentials_subscription.example.id
	name = rediscloud_essentials_database.example.name
}

resource "rediscloud_essentials_subscription" "example" {
  name = "%s"
  plan_id = data.rediscloud_essentials_plan.example.id
  payment_method_id = data.rediscloud_payment_method.card.id
}

resource "rediscloud_essentials_database" "example" {
  subscription_id     = rediscloud_essentials_subscription.example.id
  name                = "%s"
  enable_default_user = false
  data_persistence = "none"
  replication      = false

  alert {
    name  = "throughput-higher-than"
    value = 80
  }
  tags = {
    "envaaaa" = "qaaaa"
  }
}
`
