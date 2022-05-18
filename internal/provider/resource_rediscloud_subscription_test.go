package provider

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var contractFlag = flag.Bool("contract", false,
	"Add this flag '-contract' to run tests for contract associated accounts")

var marketplaceFlag = flag.Bool("marketplace", false,
	"Add this flag '-marketplace' to run tests for marketplace associated accounts")

func TestAccResourceRedisCloudSubscription_createWithDatabase(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "database.0.data_eviction", "volatile-lru"),
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
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceRedisCloudSubscription_addUpdateDeleteDatabase(t *testing.T) {

	if testing.Short() {
		t.Skip("Requires manual execution over CI execution")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	password2 := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "database.0.data_eviction", "volatile-lru"),
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

func TestAccResourceRedisCloudSubscription_AddAdditionalDatabaseWithModule(t *testing.T) {

	if testing.Short() {
		t.Skip("Requires manual execution over CI execution")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	password2 := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionOneDb, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionTwoDbWithModule, testCloudAccountName, name, 2, password, password2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "database.#", "2"),
					resource.TestMatchResourceAttr(resourceName, "database.1.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttr(resourceName, "database.1.name", "tf-database-2"),
					resource.TestCheckResourceAttr(resourceName, "database.1.module.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "database.1.module.0.name", "RediSearch"),
				),
			},
		},
	})
}

// Steps:
// - Create a subscription with a multi-module db.
// - Add another multi-module db to the subscription.
func TestAccResourceRedisCloudSubscription_AddDatabasesWithMultiModules(t *testing.T) {
	if testing.Short() {
		t.Skip("Requires manual execution over CI execution")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionDbWithMultipleModules, testCloudAccountName, name, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "database.0.module.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "database.0.module.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "database.0.module.1.name", "RedisJSON"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionTwoDbsWithMultipleModules, testCloudAccountName, name, password, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "database.0.module.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "database.0.module.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "database.0.module.1.name", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "database.1.module.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "database.1.module.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "database.1.module.1.name", "RedisJSON"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudSubscription_AddManageDatabaseReplication(t *testing.T) {

	if testing.Short() {
		t.Skip("Requires manual execution over CI execution")
	}

	originResourceName := "rediscloud_subscription.origin"
	originSubName := acctest.RandomWithPrefix(testResourcePrefix)
	originDatabaseName := "tf-database-origin"
	originDatabasePassword := acctest.RandString(20)

	replicaResourceName := "rediscloud_subscription.replica"
	replicaSubName := acctest.RandomWithPrefix(testResourcePrefix)
	replicaDatabaseName := "tf-database-replica"
	replicaDatabasePassword := acctest.RandString(20)

	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionsWithReplicaDB, testCloudAccountName, originSubName, originDatabaseName, originDatabasePassword, replicaSubName, replicaDatabaseName, replicaDatabasePassword),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(originResourceName, "name", originSubName),
					resource.TestCheckResourceAttr(originResourceName, "database.#", "1"),
					resource.TestCheckResourceAttr(originResourceName, "database.0.name", originDatabaseName),
					resource.TestCheckResourceAttr(replicaResourceName, "name", replicaSubName),
					resource.TestCheckResourceAttr(replicaResourceName, "database.#", "1"),
					resource.TestCheckResourceAttr(replicaResourceName, "database.0.name", replicaDatabaseName),
					resource.TestCheckResourceAttr(replicaResourceName, "database.0.replica_of.#", "1"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionsWithoutReplicaDB, testCloudAccountName, originSubName, originDatabaseName, originDatabasePassword, replicaSubName, replicaDatabaseName, replicaDatabasePassword),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(replicaResourceName, "name", replicaSubName),
					resource.TestCheckResourceAttr(replicaResourceName, "database.#", "1"),
					resource.TestCheckResourceAttr(replicaResourceName, "database.0.name", replicaDatabaseName),
					resource.TestCheckResourceAttr(replicaResourceName, "database.0.replica_of.#", "0"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudSubscription_createUpdateContractPayment(t *testing.T) {

	if !*contractFlag {
		t.Skip("The '-contract' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	password := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionContractPayment, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "database.0.data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionContractPayment, testCloudAccountName, updatedName, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudSubscription_createUpdateMarketplacePayment(t *testing.T) {

	if !*marketplaceFlag {
		t.Skip("The '-marketplace' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	password := acctest.RandString(20)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionMarketplacePayment, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "database.#", "1"),
					resource.TestMatchResourceAttr(resourceName, "database.0.db_id", regexp.MustCompile("^[1-9][0-9]*$")),
					resource.TestCheckResourceAttrSet(resourceName, "database.0.password"),
					resource.TestCheckResourceAttr(resourceName, "database.0.name", "tf-database"),
					resource.TestCheckResourceAttr(resourceName, "database.0.memory_limit_in_gb", "1"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionMarketplacePayment, testCloudAccountName, updatedName, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
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

// A simple Set function to generate a hash based on two attributes.
func testSetFunc(v interface{}) int {
	m := v.(map[string]interface{})
	result := fmt.Sprintf("%s%d", m["name"].(string), m["memory_limit_in_gb"].(int))
	return schema.HashString(result)
}

// Tests the diff() function. Checks if the function detects a new or modified database.
func TestDiffFunction(t *testing.T) {
	var oldDbBlocks []interface{}
	var newDbBlocks []interface{}

	// The user created 3 dbs
	oldDbBlocks = append(oldDbBlocks, map[string]interface{}{
		"name":               "db-0",
		"memory_limit_in_gb": 1,
	}, map[string]interface{}{
		"name":               "db-1",
		"memory_limit_in_gb": 1,
	}, map[string]interface{}{
		"name":               "db-2",
		"memory_limit_in_gb": 1,
	})

	// The user deleted db-0, modified db-1, added db-3 (new).
	newDbBlocks = append(newDbBlocks, map[string]interface{}{
		"name":               "db-1",
		"memory_limit_in_gb": 2,
	}, map[string]interface{}{
		"name":               "db-2",
		"memory_limit_in_gb": 1,
	}, map[string]interface{}{
		"name":               "db-3",
		"memory_limit_in_gb": 1,
	})

	oldSet := schema.NewSet(testSetFunc, oldDbBlocks)
	newSet := schema.NewSet(testSetFunc, newDbBlocks)

	addition, existing, deletion := diff(oldSet, newSet, func(v interface{}) string {
		m := v.(map[string]interface{})
		return m["name"].(string)
	})

	assert.Len(t, addition, 1)
	assert.Len(t, existing, 1)
	assert.Len(t, deletion, 1)

	assert.Equal(t, addition[0]["name"], "db-3")
	assert.Equal(t, existing[0]["name"], "db-1")
	assert.Equal(t, deletion[0]["name"], "db-0")
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
	data_eviction = "volatile-lru"
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
	data_eviction = "volatile-lru"
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
	data_eviction = "volatile-lru"
    replication = false
    throughput_measurement_by = "number-of-shards"
    throughput_measurement_value = 2
    password = "%s"
  }
}
`

const testAccResourceRedisCloudSubscriptionTwoDbWithModule = `
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
	data_eviction = "volatile-lru"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
  }

  database {
    name = "tf-database-2"
     protocol = "redis"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true
    data_persistence = "none"
	data_eviction = "volatile-lru"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]

	module {
		name = "RediSearch"
	}
  }
}
`

const testAccResourceRedisCloudSubscriptionDbWithMultipleModules = `
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
    name = "tf-database-0"
	protocol = "redis"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]

    dynamic "module" {
      for_each = ["RedisJSON", "RedisBloom"]
      content {
        name = module.value
      }
    }
  }
}
`

const testAccResourceRedisCloudSubscriptionTwoDbsWithMultipleModules = `
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
    name = "tf-database-0"
	protocol = "redis"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]

    dynamic "module" {
      for_each = ["RedisJSON", "RedisBloom"]
      content {
        name = module.value
      }
    }
  }
  
  database {
    name = "tf-database-1"
	protocol = "redis"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]

    dynamic "module" {
      for_each = ["RedisJSON", "RedisBloom"]
      content {
        name = module.value
      }
    }
  }
  
}
`

const testAccResourceRedisCloudSubscriptionsWithReplicaDB = `
locals {
  test_cloud_account_name = "%s"
  origin_sub_name = "%s"
  origin_db_name = "%s"
  origin_db_password = "%s"
  
  replica_sub_name = "%s"
  replica_db_name = "%s"
  replica_db_password = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = local.test_cloud_account_name
}

resource "rediscloud_subscription" "origin" {

  name                          = local.origin_sub_name
  payment_method_id             = data.rediscloud_payment_method.card.id
  memory_storage                = "ram"

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-2"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = []
    }
  }

  database {
    name                         = local.origin_db_name
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    password                     = local.origin_db_password
  }

}

resource "rediscloud_subscription" "replica" {

  name                          = local.replica_sub_name
  payment_method_id             = data.rediscloud_payment_method.card.id
  memory_storage                = "ram"

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-2"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = []
    }
  }

  database {
    name                         = local.replica_db_name
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    password                     = local.replica_db_password
    replica_of                   = [ {for d in rediscloud_subscription.origin.database : d.name => "redis://${d.public_endpoint}"}[local.origin_db_name] ]
  }

}
`

const testAccResourceRedisCloudSubscriptionsWithoutReplicaDB = `
locals {
  test_cloud_account_name = "%s"
  origin_sub_name = "%s"
  origin_db_name = "%s"
  origin_db_password = "%s"
  
  replica_sub_name = "%s"
  replica_db_name = "%s"
  replica_db_password = "%s"
}

data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = local.test_cloud_account_name
}

resource "rediscloud_subscription" "origin" {

  name                          = local.origin_sub_name
  payment_method_id             = data.rediscloud_payment_method.card.id
  memory_storage                = "ram"

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-2"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = []
    }
  }

  database {
    name                         = local.origin_db_name
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    password                     = local.origin_db_password
  }

}

resource "rediscloud_subscription" "replica" {

  name                          = local.replica_sub_name
  payment_method_id             = data.rediscloud_payment_method.card.id
  memory_storage                = "ram"

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-2"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = []
    }
  }

  database {
    name                         = local.replica_db_name
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    password                     = local.replica_db_password
  }

}
`

const testAccResourceRedisCloudSubscriptionContractPayment = `

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  memory_storage = "ram"

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
	data_eviction = "volatile-lru"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
    source_ips = ["10.0.0.0/8"]
  }
}
`

const testAccResourceRedisCloudSubscriptionMarketplacePayment = `

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  memory_storage = "ram"
  payment_method = "marketplace"

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
