package provider

import (
	"context"
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	client2 "github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var activeActiveContractFlag = flag.Bool("activeActiveContract", false,
	"Add this flag '-activeActiveContract' to run tests for contract associated accounts")

var activeActiveMarketplaceFlag = flag.Bool("activeActiveMarketplace", false,
	"Add this flag '-activeActiveMarketplace' to run tests for marketplace associated accounts")

// Checks CRUDI (CREATE, READ, UPDATE, IMPORT) operations on the subscription resource with Redis 7.
// Also checks active-active subscription regions.
func TestAccResourceRedisCloudActiveActiveSubscription_CRUDI_Redis7(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"
	const datasourceSubscriptionName = "data.rediscloud_active_active_subscription.example"
	const datasourceRegionName = "data.rediscloud_active_active_subscription_regions.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudActiveActiveSubscriptionRedis7(t, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the subscription resource
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),

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

					resource.TestCheckResourceAttr(resourceName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(resourceName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(resourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(resourceName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(resourceName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.price_period", "hour"),

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName].Primary
						region1 := r.Attributes["pricing.0.region"]
						region2 := r.Attributes["pricing.1.region"]

						match := (region1 == "us-east-1" && region2 == "us-east-2") ||
							(region2 == "us-east-1" && region1 == "us-east-2")

						if !match {
							return fmt.Errorf("regions within pricing response are incorrect. expected us-east-1 and us-east-2")
						}
						return nil
					},

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*client2.ApiClient)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}
						return nil
					},

					// Test the subscription datasource
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "name", name),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "payment_method_id"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "aws_account_id"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "number_of_databases", "0"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "status", "active"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.#", "2"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.2", "Sunday"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.price_period", "hour"),

					// Test the region datasource

					resource.TestCheckResourceAttr(datasourceRegionName, "subscription_name", name),
					//resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.regionId"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.networking_deployment_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.vpc_id"),
					//resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.regionId"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.networking_deployment_cidr", "10.0.1.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.vpc_id"),

					// Test the database resource - check override_region blocks reference valid subscription regions
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "override_region.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-2",
					}),

					// Test database is using Redis 7.4
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "redis_version", "7.4"),

					// checks enabling default user is true
					//resource.TestCheckResourceAttr(resourceName, "regions.1.enable_default_user", "true"),
				),
			},
			{
				// Checks if the changes in the creation plan are ignored.
				Config: testAccResourceRedisCloudActiveActiveSubscriptionUpdateRedis7(t, name, "AWS"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),

					// Check database settings
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "global_enable_default_user", "false"),

					// Check database override_region blocks reference valid subscription regions
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "override_region.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-2",
					}),

					// also checks user has removed default user
					//resource.TestCheckResourceAttr(resourceName, "regions.1.enable_default_user", "false"),

					// maintenance windows spec is omitted so should default back to automatic
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.#", "0"),
				),
			},
			{
				// Checks if the changes to the payment_method are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
				),
			},
			{
				// Checks if the payment_method and creation_plan block are ignored after the IMPORT operation.
				Config:       testAccResourceRedisCloudActiveActiveSubscriptionImportRedis7(t, name),
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
				Config:       fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan, name, "GCP"),
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

// Checks CRUDI (CREATE, READ, UPDATE, IMPORT) operations on the subscription resource with Redis 8.
// Also checks active-active subscription regions.
func TestAccResourceRedisCloudActiveActiveSubscription_CRUDI_Redis8(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"
	const datasourceSubscriptionName = "data.rediscloud_active_active_subscription.example"
	const datasourceRegionName = "data.rediscloud_active_active_subscription_regions.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudActiveActiveSubscriptionRedis8(t, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the subscription resource
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "true"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "aws_account_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),

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

					resource.TestCheckResourceAttr(resourceName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(resourceName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(resourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(resourceName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(resourceName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(resourceName, "pricing.1.price_period", "hour"),

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName].Primary
						region1 := r.Attributes["pricing.0.region"]
						region2 := r.Attributes["pricing.1.region"]

						match := (region1 == "us-east-1" && region2 == "us-east-2") ||
							(region2 == "us-east-1" && region1 == "us-east-2")

						if !match {
							return fmt.Errorf("regions within pricing response are incorrect. expected us-east-1 and us-east-2")
						}
						return nil
					},

					func(s *terraform.State) error {
						r := s.RootModule().Resources[resourceName]

						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return err
						}

						client := testProvider.Meta().(*client2.ApiClient)
						sub, err := client.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return err
						}

						if redis.StringValue(sub.Name) != name {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}
						return nil
					},

					// Test the subscription datasource
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "name", name),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "payment_method_id"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "aws_account_id"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "number_of_databases", "0"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "status", "active"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.#", "2"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.start_hour", "22"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.duration_in_hours", "8"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.#", "2"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.0", "Monday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.0.days.1", "Thursday"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.start_hour", "12"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.duration_in_hours", "6"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.#", "3"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.0", "Friday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.1", "Saturday"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.1.days.2", "Sunday"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.#", "2"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.0.price_period", "hour"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.type", "MinimumPrice"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.quantity", "1"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.quantity_measurement", "subscription"),
					resource.TestCheckResourceAttrSet(datasourceSubscriptionName, "pricing.1.price_per_unit"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.price_currency", "USD"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "pricing.1.price_period", "hour"),

					// Test the region datasource

					resource.TestCheckResourceAttr(datasourceRegionName, "subscription_name", name),
					//resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.regionId"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.networking_deployment_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.vpc_id"),
					//resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.regionId"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.networking_deployment_cidr", "10.0.1.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.vpc_id"),

					// Test the database resource - check override_region blocks reference valid subscription regions
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "override_region.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-2",
					}),

					// Test database is using Redis 8.2
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "redis_version", "8.2"),

					// checks enabling default user is true
					//resource.TestCheckResourceAttr(resourceName, "regions.1.enable_default_user", "true"),
				),
			},
			{
				// Checks if the changes in the creation plan are ignored.
				Config: testAccResourceRedisCloudActiveActiveSubscriptionUpdateRedis8(t, name, "AWS"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.quantity", "1"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.modules.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),

					// Check database settings
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "global_enable_default_user", "false"),

					// Check database override_region blocks reference valid subscription regions
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "override_region.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("rediscloud_active_active_subscription_database.example", "override_region.*", map[string]string{
						"name": "us-east-2",
					}),

					// also checks user has removed default user
					//resource.TestCheckResourceAttr(resourceName, "regions.1.enable_default_user", "false"),

					// maintenance windows spec is omitted so should default back to automatic
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "0"),

					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.mode", "automatic"),
					resource.TestCheckResourceAttr(datasourceSubscriptionName, "maintenance_windows.0.window.#", "0"),
				),
			},
			{
				// Checks if the changes to the payment_method are ignored.
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
				),
			},
			{
				// Checks if the payment_method and creation_plan block are ignored after the IMPORT operation.
				Config:       testAccResourceRedisCloudActiveActiveSubscriptionImportRedis8(t, name),
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
				Config:       fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan, name, "GCP"),
				ResourceName: resourceName,
				ExpectError:  regexp.MustCompile(`Error: the "creation_plan" block is required`),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscription_createUpdateContractPayment(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*activeActiveContractFlag {
		t.Skip("The '-activeActiveContract' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionContractPayment, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionContractPayment, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscription_createUpdateMarketplacePayment(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	if !*activeActiveMarketplaceFlag {
		t.Skip("The '-activeActiveMarketplace' parameter wasn't provided in the test command.")
	}

	name := acctest.RandomWithPrefix(testResourcePrefix)
	updatedName := fmt.Sprintf("%v-updatedName", name)
	const resourceName = "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "creation_plan.0.region.1.read_operations_per_second", "1000"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment, updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", updatedName),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveSubscription_PublicEndpointAccess(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRedisCloudActiveActiveSubscriptionPublicEndpointDisabled(t, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "false"),
				),
			},
			{
				Config: testAccResourceRedisCloudActiveActiveSubscriptionPublicEndpointEnabled(t, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "public_endpoint_access", "true"),
				),
			},
		},
	})
}

func testAccCheckActiveActiveSubscriptionDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*client2.ApiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_active_active_subscription" {
			continue
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		subs, err := client.Client.Subscription.List(context.TODO())
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

const testAccResourceRedisCloudActiveActiveSubscriptionNoCreationPlan = `
  
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "%s"

	maintenance_windows {
		mode = "automatic"
	}
}

data "rediscloud_active_active_subscription" "example" {
	name = rediscloud_active_active_subscription.example.name
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionChangedPaymentMethod = `
resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
    payment_method = "marketplace"
	cloud_provider = "AWS"

	creation_plan {
		memory_limit_in_gb = 1
		quantity = 1
		region {
			region = "us-east-1"
			networking_deployment_cidr = "192.168.0.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
		region {
			region = "us-east-2"
			networking_deployment_cidr = "10.0.1.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
	}
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionContractPayment = `

resource "rediscloud_active_active_subscription" "example" {
  	name = "%s"
	cloud_provider = "AWS"
  
	creation_plan {
	  	memory_limit_in_gb = 2
	  	quantity = 1
	  	region {
			region = "us-east-1"
			networking_deployment_cidr = "192.168.0.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
		region {
			region = "us-east-2"
			networking_deployment_cidr = "10.0.1.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
	}
}
`

const testAccResourceRedisCloudActiveActiveSubscriptionMarketplacePayment = `

resource "rediscloud_active_active_subscription" "example" {
  name = "%s"
  payment_method = "marketplace"

  cloud_provider = "AWS"
  creation_plan {
    memory_limit_in_gb = 2
    quantity = 1
    region {
		region = "us-east-1"
		networking_deployment_cidr = "192.168.0.0/24"
		write_operations_per_second = 1000
		read_operations_per_second = 1000
	}
	region {
		region = "us-east-2"
		networking_deployment_cidr = "10.0.1.0/24"
		write_operations_per_second = 1000
		read_operations_per_second = 1000
	}
	}
  }
`

func testAccResourceRedisCloudActiveActiveSubscription(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/active_active_sub.tf")
	return fmt.Sprintf(content, subscriptionName)
}

func testAccResourceRedisCloudActiveActiveSubscriptionRedis7(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/active_active_sub_redis7.tf")
	return fmt.Sprintf(content, subscriptionName)
}

func testAccResourceRedisCloudActiveActiveSubscriptionRedis8(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/active_active_sub_redis8.tf")
	return fmt.Sprintf(content, subscriptionName)
}

func testAccResourceRedisCloudActiveActiveSubscriptionUpdate(t *testing.T, subscriptionName string, cloudProvider string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/subscription_update.tf")
	return fmt.Sprintf(content, subscriptionName, cloudProvider)
}

func testAccResourceRedisCloudActiveActiveSubscriptionUpdateRedis7(t *testing.T, subscriptionName string, cloudProvider string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/subscription_update_redis7.tf")
	return fmt.Sprintf(content, subscriptionName, cloudProvider)
}

func testAccResourceRedisCloudActiveActiveSubscriptionUpdateRedis8(t *testing.T, subscriptionName string, cloudProvider string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/subscription_update_redis8.tf")
	return fmt.Sprintf(content, subscriptionName, cloudProvider)
}

func testAccResourceRedisCloudActiveActiveSubscriptionPublicEndpointDisabled(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_disabled.tf")
	return fmt.Sprintf(content, subscriptionName)
}

func testAccResourceRedisCloudActiveActiveSubscriptionPublicEndpointEnabled(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_enabled.tf")
	return fmt.Sprintf(content, subscriptionName)
}

func testAccResourceRedisCloudActiveActiveSubscriptionImportRedis7(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/subscription_import_redis7.tf")
	return fmt.Sprintf(content, subscriptionName)
}

func testAccResourceRedisCloudActiveActiveSubscriptionImportRedis8(t *testing.T, subscriptionName string) string {
	content := utils.GetTestConfig(t, "./activeactive/testdata/subscription_import_redis8.tf")
	return fmt.Sprintf(content, subscriptionName)
}
