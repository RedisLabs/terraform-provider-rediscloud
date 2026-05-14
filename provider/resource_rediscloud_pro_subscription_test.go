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
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var contractFlag = flag.Bool("contract", false,
	"Add this flag '-contract' to run tests for contract associated accounts")

var marketplaceFlag = flag.Bool("marketplace", false,
	"Add this flag '-marketplace' to run tests for marketplace associated accounts")

// Checks CRUDI (CREATE,READ,UPDATE,IMPORT) operations on the subscription resource with Redis 7.
func TestAccResourceRedisCloudProSubscription_CRUDI_Redis7(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_redis7.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(testCloudAccountName),
					"subscription_name":  config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.query_performance_factor", "4x"),

					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "3"),
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

						apiClient := sharedTestClient(t)
						sub, err := apiClient.Client.Subscription.Get(context.TODO(), subId)
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_no_creation_plan.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
					"memory_storage":               config.StringVariable("ram"),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_changed_payment_method.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_no_creation_plan.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
					"memory_storage":               config.StringVariable("ram-and-flash"),
				},
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

// Checks CRUDI (CREATE,READ,UPDATE,IMPORT) operations on the subscription resource with Redis 8.
func TestAccResourceRedisCloudProSubscription_CRUDI_Redis8(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_redis8.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(testCloudAccountName),
					"subscription_name":  config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.average_item_size_in_bytes", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.query_performance_factor", "4x"),

					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "0"),
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

						apiClient := sharedTestClient(t)
						sub, err := apiClient.Client.Subscription.Get(context.TODO(), subId)
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_no_creation_plan.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
					"memory_storage":               config.StringVariable("ram"),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_changed_payment_method.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_no_creation_plan.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
					"memory_storage":               config.StringVariable("ram-and-flash"),
				},
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_preferredAZsModulesOptional(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_preferred_azs_modules_optional.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_createUpdateContractPayment(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*contractFlag {
		t.Skip("The '-contract' parameter wasn't provided in the test command.")
	}

	name := testRandomWithPrefix()
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_contract_payment.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
				),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_contract_payment.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(updatedName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_createUpdateMarketplacePayment(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*marketplaceFlag {
		t.Skip("The '-marketplace' parameter wasn't provided in the test command.")
	}

	name := testRandomWithPrefix()
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_marketplace_payment.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.0.networks.0.networking_subnet_id"),
				),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_marketplace_payment.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(updatedName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudProSubscription_RedisVersion(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	identifier := ""

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_no_redis_version.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_redis_version_latest.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix() + "-mw"
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
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_default.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_auto.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceName, "maintenance_windows.0.window.#", "0"),
				),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_manual.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_error_manual.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				ExpectError: regexp.MustCompile("Must provide at least one maintenance window with manual maintenance mode"),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_error_auto.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				ExpectError: regexp.MustCompile("Automatic mode cannot be set with a manual maintenance window"),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_manual_multiple.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_maintenance_windows_auto.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
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

func TestAccResourceRedisCloudProSubscription_PublicEndpointAccess(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()
	resourceName := "rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_access_disabled.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "false"),
				),
			},
			{
				ConfigFile: config.StaticFile("./pro/testdata/pro_subscription_access_enabled.tf"),
				ConfigVariables: config.Variables{
					"rediscloud_cloud_account":     config.StringVariable(testCloudAccountName),
					"rediscloud_subscription_name": config.StringVariable(name),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "true"),
				),
			},
		},
	})
}

// Checks that modules are allocated correctly into each creation-plan db if there are multiple modules, including "RedisGraph" and the number of databases is one.
func TestFlexSubModulesAllocationWhenGraphAndQuantityIsOne(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRamAndFlash, planMap)
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

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
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

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, numDatabases)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		modules := createDb.Modules
		assert.True(t, len(modules) == 1 && *modules[0].Name == "RedisGraph")
	}
}

// Checks that modules are allocated correctly into the creation-plan dbs if "RedisGraph" is not included
func TestFlexSubModulesAllocationWhenNoGraph(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
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

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, 2)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		modules := createDb.Modules
		assert.Len(t, modules, 0)
	}
}

func TestFlexSubNoAverageItemSizeInBytes(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, createDbs, 2)
	assert.Empty(t, diags)
	for _, createDb := range createDbs {
		assert.Nil(t, createDb.AverageItemSizeInBytes)
	}
}

func TestFlexSubRediSearchThroughputMeasurementWhenReplicationIsFalse(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "number-of-shards", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2, *createDb.ThroughputMeasurement.Value)
}

func TestFlexSubRediSearchThroughputMeasurementWhenReplicationIsTrue(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "number-of-shards", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2, *createDb.ThroughputMeasurement.Value)
}

func TestFlexSubRedisGraphThroughputMeasurementWhenReplicationIsFalse(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Empty(t, diags)
	createDb := createDbs[0]
	assert.Equal(t, "operations-per-second", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2*250, *createDb.ThroughputMeasurement.Value)
}

func TestFlexSubRedisGraphThroughputMeasurementWhenReplicationIsTrue(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUBSCRIPTION")

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
	createDbs, diags := pro.BuildSubscriptionCreatePlanDatabases(databases.MemoryStorageRam, planMap)
	assert.Len(t, diags, 1, "Warning should be reported when storage was ram and using `average_item_size_in_bytes`")
	assert.False(t, diags.HasError(), "diagnostic should be a warning, not an error")
	createDb := createDbs[0]
	assert.Equal(t, "operations-per-second", *createDb.ThroughputMeasurement.By)
	assert.Equal(t, 2*500, *createDb.ThroughputMeasurement.Value)
}

func testAccCheckProSubscriptionDestroy(s *terraform.State) error {
	apiClient, err := getTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := apiClient.Client.Subscription.List(context.TODO())
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
