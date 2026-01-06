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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// Checks CRUDI (CREATE, READ, UPDATE, IMPORT) operations on the database resource.
func TestAccResourceRedisCloudActiveActiveDatabase_CRUDI(t *testing.T) {
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	databaseName := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)

	const databaseResourceName = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	const datasourceRegionName = "data.rediscloud_active_active_subscription_regions.example"
	const subscriptionResourceName = "rediscloud_active_active_subscription.example"

	var subId, dbId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database creation
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/database_crudi_create.tf", map[string]string{
					"__SUBSCRIPTION_NAME__": subscriptionName,
					"__DATABASE_NAME__":     databaseName,
					"__PASSWORD__":          password,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test resource attributes
					resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
					resource.TestCheckResourceAttr(databaseResourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_data_persistence", "none"),
					resource.TestCheckResourceAttr(databaseResourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_password", password),
					resource.TestCheckResourceAttr(databaseResourceName, "enable_tls", "false"),
					resource.TestCheckResourceAttr(databaseResourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.0.value", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_modules.#", "0"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_source_ips.#", "2"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Check override_region for us-east-1 (with overrides)
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_data_persistence", "aof-every-write"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_password", "region-specific-password"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.0.value", "42"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_source_ips.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_source_ips.0", "192.175.0.0/16"),

					// Check override_region for us-east-2 (no overrides - fields should be null)
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.1.name", "us-east-2"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.override_global_data_persistence"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.override_global_password"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.1.override_global_alert.#", "0"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.override_source_ips"),

					// Check tags
					resource.TestCheckResourceAttr(databaseResourceName, "tags.deployment_family", "blue"),
					resource.TestCheckResourceAttr(databaseResourceName, "tags.priority", "code-2"),

					// API check: verify database was created correctly
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionResourceName]
						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse subscription ID: %s", r.Primary.ID)
						}

						dbResource := s.RootModule().Resources[databaseResourceName]
						dbId, err = strconv.Atoi(dbResource.Primary.Attributes["db_id"])
						if err != nil {
							return fmt.Errorf("couldn't parse database ID: %s", dbResource.Primary.Attributes["db_id"])
						}

						apiClient := sharedTestClient(t)

						// Verify subscription
						sub, err := apiClient.Client.Subscription.Get(context.TODO(), subId)
						if err != nil {
							return fmt.Errorf("failed to get subscription: %w", err)
						}
						if redis.StringValue(sub.Name) != subscriptionName {
							return fmt.Errorf("expected subscription name %q, got %q", subscriptionName, redis.StringValue(sub.Name))
						}

						// Verify database via API
						db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
						if err != nil {
							return fmt.Errorf("failed to get database: %w", err)
						}
						if redis.StringValue(db.Name) != databaseName {
							return fmt.Errorf("expected database name %q, got %q", databaseName, redis.StringValue(db.Name))
						}
						if redis.StringValue(db.GlobalDataPersistence) != "none" {
							return fmt.Errorf("expected global_data_persistence %q, got %q", "none", redis.StringValue(db.GlobalDataPersistence))
						}

						return nil
					},

					// Test data sources
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", databaseName),
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "enable_tls", "false"),
					resource.TestCheckResourceAttrSet(datasourceName, "tls_certificate"),
					resource.TestCheckResourceAttr(datasourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(datasourceName, "tags.deployment_family", "blue"),
					resource.TestCheckResourceAttr(datasourceName, "tags.priority", "code-2"),

					// Test regions data source
					resource.TestCheckResourceAttr(datasourceRegionName, "subscription_name", subscriptionName),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.vpc_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.networking_deployment_cidr", "192.168.0.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.vpc_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.networking_deployment_cidr", "10.0.1.0/24"),
				),
			},
			// Test database update: change global and local alerts, enable OSS cluster API
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/database_crudi_update.tf", map[string]string{
					"__SUBSCRIPTION_NAME__": subscriptionName,
					"__DATABASE_NAME__":     databaseName,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_password", "updated-password"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.0.value", "60"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_modules.#", "0"),

					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_data_persistence", "none"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_password", "password-updated"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.0.value", "41"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_source_ips.#", "0"),

					// API check: verify updates applied
					func(s *terraform.State) error {
						apiClient := sharedTestClient(t)
						db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
						if err != nil {
							return fmt.Errorf("failed to get database: %w", err)
						}
						if redis.StringValue(db.GlobalDataPersistence) != "aof-every-1-second" {
							return fmt.Errorf("expected global_data_persistence %q, got %q", "aof-every-1-second", redis.StringValue(db.GlobalDataPersistence))
						}
						if redis.BoolValue(db.SupportOSSClusterAPI) != true {
							return fmt.Errorf("expected support_oss_cluster_api true, got false")
						}
						return nil
					},

					// Test data source reflects updates
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(datasourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "true"),
				),
			},
			// Test database update: remove alerts, restore default user
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/database_crudi_update_no_alerts.tf", map[string]string{
					"__SUBSCRIPTION_NAME__": subscriptionName,
					"__DATABASE_NAME__":     databaseName,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "external_endpoint_for_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_password", "updated-password"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_alert.#", "0"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_modules.#", "0"),
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.name", "us-east-1"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_data_persistence", "none"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_password", "password-updated"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_alert.#", "0"),
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.0.override_global_source_ips.#", "0"),
				),
			},
			// Test import
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/database_crudi_import.tf", map[string]string{
					"__SUBSCRIPTION_NAME__": subscriptionName,
					"__DATABASE_NAME__":     databaseName,
				}),
				ResourceName:      databaseResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"global_password",
					"global_source_ips.#",
					"global_source_ips.0",
					"override_region.#",
					"override_region.0.%",
					"override_region.0.name",
					"override_region.0.override_global_alert.#",
					"override_region.0.override_global_alert.0.%",
					"override_region.0.override_global_alert.0.name",
					"override_region.0.override_global_alert.0.value",
					"override_region.0.override_global_data_persistence",
					"override_region.0.override_global_password",
					"override_region.0.enable_default_user",
				},
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveDatabase_optionalAttributes(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	// Test that attributes can be optional, either by setting them or not having them set when compared to CRUDI test
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	name := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)
	const resourceName = "rediscloud_active_active_subscription_database.example"
	portNumber := 10101

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseOptionalAttributes, subscriptionName, name, password, portNumber),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "port", strconv.Itoa(portNumber)),
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveDatabase_timeUtcRequiresValidInterval(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabaseInvalidTimeUtc, testCloudAccountName, name, password),
				ExpectError: regexp.MustCompile("unexpected value at override_region\\.\\d*\\.remote_backup\\.0\\.time_utc - time_utc can only be set when interval is either every-24-hours or every-12-hours"),
			},
		},
	})
}

const activeActiveSubscriptionBoilerplate = `
	data "rediscloud_payment_method" "card" {
		card_type = "Visa"
		last_four_numbers = "5556"
	}

	resource "rediscloud_active_active_subscription" "example" {
		name = "%s"
		payment_method_id = data.rediscloud_payment_method.card.id
		cloud_provider = "AWS"

		creation_plan {
			dataset_size_in_gb = 1
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

const testAccResourceRedisCloudActiveActiveDatabaseOptionalAttributes = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"

	creation_plan {
		dataset_size_in_gb = 1
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

resource "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = "%s"
	dataset_size_in_gb = 1
	support_oss_cluster_api = false 
	external_endpoint_for_oss_cluster_api = false
	enable_tls = false
	global_resp_version = "resp3"

	global_data_persistence = "none"
	global_password = "%s" 
	global_source_ips = ["192.168.0.0/16", "192.170.0.0/16"]
	global_alert {
		name = "dataset-size"
		value = 1
	}
	override_region {
		name = "us-east-1"
		override_global_data_persistence = "aof-every-write"
		override_global_source_ips = ["192.175.0.0/16"]
		override_global_password = "region-specific-password"
		override_global_alert {
			name = "dataset-size"
			value = 42
		}
	}
	override_region {
		name = "us-east-2"
	}
	port = %d
} 
`

const testAccResourceRedisCloudActiveActiveDatabaseInvalidTimeUtc = activeActiveSubscriptionBoilerplate + `
resource "rediscloud_active_active_subscription_database" "example" {
	subscription_id = rediscloud_active_active_subscription.example.id
	name = "%s"
	dataset_size_in_gb = 1
	support_oss_cluster_api = false 
	external_endpoint_for_oss_cluster_api = false
	enable_tls = false

	global_data_persistence = "none"
	global_password = "%s" 
	global_source_ips = ["192.168.0.0/16", "192.170.0.0/16"]
	global_alert {
		name = "dataset-size"
		value = 1
	}
	override_region {
		name = "us-east-1"
		override_global_data_persistence = "aof-every-write"
		override_global_source_ips = ["192.175.0.0/16"]
		override_global_password = "region-specific-password"
		override_global_alert {
			name = "dataset-size"
			value = 42
		}
		remote_backup {
			interval = "every-6-hours"
			time_utc = "16:00"
			storage_type = "aws-s3"
			storage_path = "uri://interval.not.12.or.24.hours.test"
		}
	}
	override_region {
		name = "us-east-2"
	}
} 
`

func TestAccResourceRedisCloudActiveActiveDatabase_autoMinorVersionUpgrade(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	t.Skip("auto_minor_version_upgrade feature temporarily removed")

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-subscription"
	const resourceName = "rediscloud_active_active_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database creation with auto_minor_version_upgrade set to false
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/auto_minor_version_upgrade.tf", map[string]string{
					"__CLOUD_ACCOUNT__":              testCloudAccountName,
					"__SUBSCRIPTION_NAME__":          subscriptionName,
					"__AUTO_MINOR_VERSION_UPGRADE__": "false",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "auto-minor-version-upgrade-test"),
					resource.TestCheckResourceAttr(resourceName, "auto_minor_version_upgrade", "false"),
				),
			},
			// Test database update with auto_minor_version_upgrade set to true
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/auto_minor_version_upgrade.tf", map[string]string{
					"__CLOUD_ACCOUNT__":              testCloudAccountName,
					"__SUBSCRIPTION_NAME__":          subscriptionName,
					"__AUTO_MINOR_VERSION_UPGRADE__": "true",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "auto_minor_version_upgrade", "true"),
				),
			},
		},
	})
}
