package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Checks CRUDI (CREATE, READ, UPDATE, IMPORT) operations on the database resource.
func TestAccResourceRedisCloudActiveActiveDatabase_CRUDI_temp(t *testing.T) {

	subscriptionName := "playground-matt"
	databaseName := "playground-matt"
	const datasourceName = "data.rediscloud_active_active_subscription_database.temp"
	const datasourceRegionName = "data.rediscloud_active_active_subscription_regions.temp"
	const subscriptionResourceName = "rediscloud_active_active_subscription.temp"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Test database creation
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActiveDatabase_TEMP),
				Check: resource.ComposeAggregateTestCheckFunc(

					// Test subscription datasource
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "db_id"),
					resource.TestCheckResourceAttr(datasourceName, "name", databaseName),
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "3"),
					resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(datasourceName, "enable_tls", "false"),
					resource.TestCheckResourceAttr(datasourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(datasourceName, "global_modules.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "global_modules.0", "RedisJSON"),

					resource.TestCheckResourceAttr(datasourceName, "tags.deployment_family", "blue"),
					resource.TestCheckResourceAttr(datasourceName, "tags.priority", "code-2"),

					// Test the region datasource
					resource.TestCheckResourceAttr(datasourceRegionName, "subscription_name", subscriptionName),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.vpc_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.networking_deployment_cidr", "192.168.0.0/24"),

					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.0.databases.0.database_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.databases.0.database_name", databaseName),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.databases.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.0.databases.0.write_operations_per_second", "1000"),

					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.vpc_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.networking_deployment_cidr", "10.0.1.0/24"),

					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.1.databases.0.database_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.databases.0.database_name", databaseName),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.databases.0.read_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.1.databases.0.write_operations_per_second", "1000"),
				),
			},
		},
	})
}

const testAccResourceRedisCloudActiveActiveDatabase_TEMP = `
data "rediscloud_active_active_subscription" "temp" {
 name = "playground-matt"
}

data "rediscloud_active_active_subscription_database" "temp" {
 name = "playground-matt"
 subscription_id = data.rediscloud_active_active_subscription.temp.id
}

data "rediscloud_active_active_subscription_regions" "temp" {
 subscription_name = "playground-matt"
}
`
