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
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var contractFlag = flag.Bool("contract", false,
	"Add this flag '-contract' to run tests for contract associated accounts")

var marketplaceFlag = flag.Bool("marketplace", false,
	"Add this flag '-marketplace' to run tests for marketplace associated accounts")

// Checks CRUDI (CREATE,READ,UPDATE,IMPORT) operations on the subscription resource.
func TestAccResourceRedisCloudSubscription_CRUDI(t *testing.T) {

	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscription, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.1", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_value", "10000"),
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
						return nil
					},
				),
			},
			{
				// Checks if the changes in the creation plan are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionNoCreationPlan, testCloudAccountName, name, "ram"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_value", "10000"),
				),
			},
			{
				// Checks if the creation_plan block is ignored after the IMPORT operation.
				ResourceName: resourceName,
				ImportState:  true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					creationPlan, ok := states[0].Attributes["creation_plan.#"]
					if ok && creationPlan != "0" {
						return fmt.Errorf("Unexpected creation_plan block. Should be 0, instead of  %s", creationPlan)
					}
					return nil
				},
			},
			{
				// Checks if an error is raised when a ForceNew attribute is changed and the creation_plan block is not defined.
				Config:       fmt.Sprintf(testAccResourceRedisCloudSubscriptionNoCreationPlan, testCloudAccountName, name, "ram-and-flash"),
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

func TestAccResourceRedisCloudSubscription_preferredAZsModulesOptional(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionPreferredAZsModulesOptional, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
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
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionContractPayment, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionContractPayment, testCloudAccountName, updatedName),
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
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionMarketplacePayment, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionMarketplacePayment, testCloudAccountName, updatedName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

// Checks that modules are allocated correctly into each creation-plan db if there are multiple modules, including "RedisGraph" and the number of databases is one.
func TestModulesAllocationWhenGraphAndQuantityIsOne(t *testing.T) {
	numDatabases := 1
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   1000,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisJSON", "RedisGraph", "RedisBloom"},
		"quantity":                     numDatabases,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 10000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRamAndFlash, planMap)
	assert.Empty(t, diags)
	otherDatabases := 0
	graphDatabases := 0
	for _, createDb := range createDbs {
		var modules []string
		for _, module := range createDb.Modules {
			modules = append(modules, *module.Name)
		}
		if len(modules) == 1 && modules[0] == "RedisGraph" {
			graphDatabases++
		}
		if len(modules) == 2 {
			assert.ElementsMatch(t, modules, []string{"RedisJSON", "RedisBloom"})
			otherDatabases++
		}
	}
	assert.Len(t, createDbs, 2)
	assert.True(t, graphDatabases == 1)
	assert.True(t, otherDatabases == 1)
}

// Checks that modules are allocated correctly into each creation-plan db if there are multiple modules, including "RedisGraph" and the number of databases is greater than one.
func TestModulesAllocationWhenGraphAndQuantityMoreThanOne(t *testing.T) {
	numDatabases := 5
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisJSON", "RedisGraph", "RedisBloom"},
		"quantity":                     numDatabases,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 10000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	graphDatabases := 0
	otherDatabases := 0
	for _, createDb := range createDbs {
		var modules []string
		for _, module := range createDb.Modules {
			modules = append(modules, *module.Name)
		}
		if len(modules) == 1 && modules[0] == "RedisGraph" {
			graphDatabases++
		}
		if len(modules) == 2 {
			assert.ElementsMatch(t, modules, []interface{}{"RedisJSON", "RedisBloom"})
			otherDatabases++
		}
	}
	assert.True(t, graphDatabases == 1)
	assert.True(t, otherDatabases == numDatabases-1)
}

// Checks that modules are allocated correctly into each creation-plan db if the only module is "RedisGraph".
func TestModulesAllocationWhenOnlyGraphModule(t *testing.T) {
	numDatabases := 5
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisGraph"},
		"quantity":                     numDatabases,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 10000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, numDatabases)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		modules := createDb.Modules
		assert.True(t, len(modules) == 1 && *modules[0].Name == "RedisGraph")
	}
}

// Checks that modules are allocated correctly into the creation-plan dbs if "RedisGraph" is not included
func TestModulesAllocationWhenNoGraph(t *testing.T) {
	numDatabases := 5
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisJSON", "RediSearch", "RedisBloom"},
		"quantity":                     numDatabases,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 10000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, numDatabases)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		var modules []string
		for _, module := range createDb.Modules {
			modules = append(modules, *module.Name)
		}
		assert.Len(t, modules, 3)
		assert.ElementsMatch(t, modules, []interface{}{"RedisJSON", "RedisBloom", "RediSearch"})
	}
}

func TestNoModulesInCreatePlanDatabases(t *testing.T) {
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{},
		"quantity":                     2,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 10000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, 2)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		modules := createDb.Modules
		assert.Len(t, modules, 0)
	}
}

func TestNoAverageItemSizeInBytes(t *testing.T) {
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0, // 0 is the value that is returned when the field is not present
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{},
		"quantity":                     2,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 10000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, 2)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		assert.Nil(t, createDb.AverageItemSizeInBytes)
	}
}

func TestRediSearchThroughputMeasurementWhenReplicationIsFalse(t *testing.T) {
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RediSearch"},
		"quantity":                     2,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 12000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "number-of-shards", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 12000/1000, *createDb.ThroughputMeasurement.Value)
}

func TestRediSearchThroughputMeasurementWhenReplicationIsTrue(t *testing.T) {
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RediSearch"},
		"quantity":                     2,
		"replication":                  true,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "operations-per-second",
		"throughput_measurement_value": 12000,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "number-of-shards", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 12000/500, *createDb.ThroughputMeasurement.Value)
}

func TestRedisGraphThroughputMeasurementWhenReplicationIsFalse(t *testing.T) {
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisGraph"},
		"quantity":                     2,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "number-of-shards",
		"throughput_measurement_value": 2,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "operations-per-second", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2*250, *createDb.ThroughputMeasurement.Value)
}

func TestRedisGraphThroughputMeasurementWhenReplicationIsTrue(t *testing.T) {
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   1000,
		"memory_limit_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisGraph"},
		"quantity":                     2,
		"replication":                  true,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "number-of-shards",
		"throughput_measurement_value": 2,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, diags, 1, "Warning should be reported when storage was ram and using `average_item_size_in_bytes`")
	assert.Equal(t, diag.Warning, diags[0].Severity)
	createDb := createDbs[0]
	assert.Equal(t, "operations-per-second", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2*500, *createDb.ThroughputMeasurement.Value)
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

// TF config for provisioning a new subscription.
const testAccResourceRedisCloudSubscription = `
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
    security_group_ids = []
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

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom"]
  }
}
`

const testAccResourceRedisCloudSubscriptionPreferredAZsModulesOptional = `
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
    security_group_ids = []
  }

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}
`

// TF config for provisioning a subscription without the creation_plan block.
const testAccResourceRedisCloudSubscriptionNoCreationPlan = `
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
  memory_storage = "%s"

  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
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
    security_group_ids = []
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

  creation_plan {
    memory_limit_in_gb = 2
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = []
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
    security_group_ids = []
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

  creation_plan {
    memory_limit_in_gb = 2
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = []
  }
}
`
