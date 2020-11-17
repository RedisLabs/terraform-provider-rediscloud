package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceRedisCloudSubscription_addUpdateDeleteDatabase(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	password2 := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDb, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)
						sub, err := client.client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}

						listDb := client.client.Database.List(context.TODO(), subId)
						if listDb.Next() != true {
							return fmt.Errorf("no database found: %s", listDb.Err())
						}
						if listDb.Err() != nil {
							return listDb.Err()
						}

						return nil
					},
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionTwoDbs, testCloudAccountName, name, 2, password, password2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "database.#", "2"),
					resource.TestMatchTypeSetElemNestedAttrs(resourceName, "database.*", map[string]*regexp.Regexp{
						"db_id":              regexp.MustCompile("^[1-9][0-9]*$"),
						"name":               regexp.MustCompile("tf-database"),
						"protocol":           regexp.MustCompile("redis"),
						"memory_limit_in_gb": regexp.MustCompile("2"),
					}),
					resource.TestMatchTypeSetElemNestedAttrs(resourceName, "database.*", map[string]*regexp.Regexp{
						"db_id":              regexp.MustCompile("^[1-9][0-9]*$"),
						"name":               regexp.MustCompile("tf-database-2"),
						"protocol":           regexp.MustCompile("memcached"),
						"memory_limit_in_gb": regexp.MustCompile("2"),
					}),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						subId, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)

						nameId, err := getDatabaseNameIdMap(context.TODO(), subId, client)
						if err != nil {
							return err
						}

						if _, ok := nameId["tf-database"]; !ok {
							return fmt.Errorf("first database doesn't exist")
						}
						if _, ok := nameId["tf-database-2"]; !ok {
							return fmt.Errorf("second database doesn't exist")
						}

						return nil
					},
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDb, testCloudAccountName, name, 2, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchTypeSetElemNestedAttrs(resourceName, "database.*", map[string]*regexp.Regexp{
						"db_id":              regexp.MustCompile("^[1-9][0-9]*$"),
						"name":               regexp.MustCompile("tf-database"),
						"protocol":           regexp.MustCompile("redis"),
						"memory_limit_in_gb": regexp.MustCompile("1"),
					}),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						subId, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*apiClient)

						nameId, err := getDatabaseNameIdMap(context.TODO(), subId, client)
						if err != nil {
							return err
						}

						if _, ok := nameId["tf-database"]; !ok {
							return fmt.Errorf("first database doesn't exist")
						}
						if _, ok := nameId["tf-database-2"]; ok {
							return fmt.Errorf("second database still exist")
						}

						return nil
					},
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckSubscriptionDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*apiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := client.client.Subscription.List(context.TODO())
		if err != nil {
			return err
		}

		for _, sub := range subs {
			if redis.IntValue(sub.ID) == subId {
				return fmt.Errorf("subscription %d still exists", subId)
			}
		}
	}

	return nil
}

const testAccResourceRedisCloudSubscriptionOneDb = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  persistent_storage_encryption = false

  allowlist {
    cidrs = ["192.168.0.0/16"]
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
  }
}
`

const testAccResourceRedisCloudSubscriptionTwoDbs = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  persistent_storage_encryption = false

  allowlist {
    cidrs = ["192.168.0.0/16"]
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
  }

  database {
    name = "tf-database-2"
    protocol = "memcached"
    memory_limit_in_gb = 2
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "number-of-shards"
    throughput_measurement_value = 2
    password = "%s"
  }
}
`
