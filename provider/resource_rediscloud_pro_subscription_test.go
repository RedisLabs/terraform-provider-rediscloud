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
func TestAccResourceRedisCloudProSubscription_CRUDI(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscription, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.query_performance_factor", "4x"),

					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.1", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.2", "RediSearch"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_value", "10000"),

					resource.TestCheckResourceAttr(resourceName, "pricing.#", "0"),

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
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionNoCreationPlan, testCloudAccountName, name, "ram"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.replication", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.throughput_measurement_value", "10000"),
				),
			},
			{
				// Checks if the changes to the payment_method are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionChangedPaymentMethod, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
				),
			},
			{
				// Checks if the payment_method and creation_plan block are ignored after the IMPORT operation.
				ResourceName: resourceName,
				ImportState:  true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					paymentMethod, ok := states[0].Attributes["payment_method"]
					if ok && paymentMethod != "credit-card" {
						return fmt.Errorf("Unexpected payment_method block. Should be 'credit-card', instead of  %s", paymentMethod)
					}
					creationPlan, ok := states[0].Attributes["creation_plan.#"]
					if ok && creationPlan != "0" {
						return fmt.Errorf("Unexpected creation_plan block. Should be 0, instead of  %s", creationPlan)
					}
					return nil
				},
			},
			{
				// Checks if an error is raised when a ForceNew attribute is changed and the creation_plan block is not defined.
				Config:       fmt.Sprintf(testAccResourceRedisCloudProSubscriptionNoCreationPlan, testCloudAccountName, name, "ram-and-flash"),
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_preferredAZsModulesOptional(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionPreferredAZsModulesOptional, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_createUpdateContractPayment(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*contractFlag {
		t.Skip("The '-contract' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionContractPayment, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionContractPayment, testCloudAccountName, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_createUpdateMarketplacePayment(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*marketplaceFlag {
		t.Skip("The '-marketplace' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionMarketplacePayment, testCloudAccountName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudSubscriptionMarketplacePayment, testCloudAccountName, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_RedisVersion(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	identifier := ""

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionWithRedisVersion, testCloudAccountName, name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Take a snapshot of the ID
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_subscription.test"]
						identifier = r.Primary.ID
						return nil
					},
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionWithRedisVersion, testCloudAccountName, name, "redis_version = \"latest\""),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Take a snapshot of the ID
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_subscription.test"]
						if r.Primary.ID == identifier {
							return fmt.Errorf("entity should have a different identifier, but was still %s", identifier)
						}
						return nil
					},
				),
			},
			{
				ResourceName:            "rediscloud_subscription.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"creation_plan", "redis_version"},
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_MaintenanceWindows(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix) + "-mw"
	resourceName := "rediscloud_subscription.example"
	datasourceName := "data.rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	const defaultMW = ""
	const autoMw = `maintenance_windows {
		mode = "automatic"
	}`
	const manualMw = `maintenance_windows {
		mode = "manual"
		window {
				start_hour = 22
				duration_in_hours = 8
				days = ["Monday", "Thursday"]
		}
	}`
	const errorManualMw = `maintenance_windows {
		mode = "manual"
		# Should have windows
	}`
	const errorAutoMw = `maintenance_windows {
		mode = "automatic"
		# Should not have windows
		window {
				start_hour = 22
				duration_in_hours = 8
				days = ["Monday", "Thursday"]
		}
	}`
	const multipleManualMw = `maintenance_windows {
		mode = "manual"
		window {
				start_hour = 22
				duration_in_hours = 8
				days = ["Monday", "Thursday"]
		}
		window {
				start_hour = 12
				duration_in_hours = 6
				days = ["Friday", "Saturday", "Sunday"]
		}
	}`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, defaultMW),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, autoMw),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, manualMw),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.days.1", "Thursday"),
				),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, errorManualMw),
				ExpectError: regexp.MustCompile("Must provide at least one maintenance window with manual maintenance mode"),
			},
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, errorAutoMw),
				ExpectError: regexp.MustCompile("Automatic mode cannot be set with a manual maintenance window"),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, multipleManualMw),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "2"),

					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.1.days.2", "Sunday"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.#", "2"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.1.days.2", "Sunday"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionMaintenanceWindows, testCloudAccountName, name, autoMw),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.#", "0"),
				),
			},
		},
	})
}

// Checks that modules are allocated correctly into each creation-plan db if there are multiple modules, including "RedisGraph" and the number of databases is one.
func TestFlexSubModulesAllocationWhenGraphAndQuantityIsOne(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	numDatabases := 1
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   1000,
		"dataset_size_in_gb":           float64(1),
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
func TestFlexSubModulesAllocationWhenGraphAndQuantityMoreThanOne(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	numDatabases := 5
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
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
func TestFlexSubModulesAllocationWhenOnlyGraphModule(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	numDatabases := 5
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
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
func TestFlexSubModulesAllocationWhenNoGraph(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	numDatabases := 5
	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
		"modules":                      []interface{}{"RedisJSON", "RediSearch", "RedisBloom"},
		"quantity":                     numDatabases,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "number-of-shards",
		"throughput_measurement_value": 2,
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

func TestFlexSubNoModulesInCreatePlanDatabases(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
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

func TestFlexSubNoAverageItemSizeInBytes(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0, // 0 is the value that is returned when the field is not present
		"dataset_size_in_gb":           float64(1),
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

func TestFlexSubRediSearchThroughputMeasurementWhenReplicationIsFalse(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
		"modules":                      []interface{}{"RediSearch"},
		"quantity":                     2,
		"replication":                  false,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "number-of-shards",
		"throughput_measurement_value": 2,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "number-of-shards", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2, *createDb.ThroughputMeasurement.Value)
}

func TestFlexSubRediSearchThroughputMeasurementWhenReplicationIsTrue(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
		"modules":                      []interface{}{"RediSearch"},
		"quantity":                     2,
		"replication":                  true,
		"support_oss_cluster_api":      false,
		"throughput_measurement_by":    "number-of-shards",
		"throughput_measurement_value": 2,
	}
	createDbs, diags := buildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "number-of-shards", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2, *createDb.ThroughputMeasurement.Value)
}

func TestFlexSubRedisGraphThroughputMeasurementWhenReplicationIsFalse(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   0,
		"dataset_size_in_gb":           float64(1),
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

func TestFlexSubRedisGraphThroughputMeasurementWhenReplicationIsTrue(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

	planMap := map[string]interface{}{
		"average_item_size_in_bytes":   1000,
		"dataset_size_in_gb":           float64(1),
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

func testAccCheckProSubscriptionDestroy(s *terraform.State) error {
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
const testAccResourceRedisCloudProSubscription = `
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
  payment_method = "credit-card"
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
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    support_oss_cluster_api = false
	query_performance_factor = "4x"

    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom", "RediSearch"]
  }
}
`

const testAccResourceRedisCloudProSubscriptionWithRedisVersion = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "test" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  # redis_version here
  %s

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
    dataset_size_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = []
  }
}
`

const testAccResourceRedisCloudProSubscriptionPreferredAZsModulesOptional = `
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
    dataset_size_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}
`

// TF config for provisioning a subscription without the creation_plan block.
const testAccResourceRedisCloudProSubscriptionNoCreationPlan = `
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

const testAccResourceRedisCloudSubscriptionChangedPaymentMethod = `
data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method = "marketplace"
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
    dataset_size_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom"]
  }
}
`

const testAccResourceRedisCloudProSubscriptionContractPayment = `

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
    dataset_size_in_gb = 2
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
    dataset_size_in_gb = 2
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = []
  }
}
`

const testAccResourceRedisCloudProSubscriptionMaintenanceWindows = `
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
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.24.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom"]
  }

  %s
}

data "rediscloud_subscription" "example" {
	name = rediscloud_subscription.example.name
}
`
