package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudProDatabase_basic(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	const dataSourceById = "data.rediscloud_database.example-by-id"
	const dataSourceByName = "data.rediscloud_database.example-by-name"
	password := acctest.RandString(20)

	config := getRedisProDbDatasourceConfig(t, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceById, "name", "tf-database"),
					resource.TestCheckResourceAttr(dataSourceById, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceById, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceById, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(dataSourceById, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(dataSourceById, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(dataSourceById, "data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceById, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(dataSourceById, "replication", "false"),
					resource.TestCheckResourceAttr(dataSourceById, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(dataSourceById, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(dataSourceById, "password", password),
					resource.TestCheckResourceAttrSet(dataSourceById, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceById, "private_endpoint"),
					resource.TestCheckResourceAttr(dataSourceById, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(dataSourceById, "query_performance_factor", "2x"),

					resource.TestCheckResourceAttr(dataSourceByName, "name", "tf-database"),
					resource.TestCheckResourceAttr(dataSourceByName, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceByName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceByName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(dataSourceByName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(dataSourceByName, "resp_version", "resp3"),
					resource.TestCheckResourceAttr(dataSourceByName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceByName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(dataSourceByName, "replication", "false"),
					resource.TestCheckResourceAttr(dataSourceByName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(dataSourceByName, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(dataSourceByName, "password", password),
					resource.TestCheckResourceAttrSet(dataSourceByName, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceByName, "private_endpoint"),
					resource.TestCheckResourceAttr(dataSourceByName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(dataSourceByName, "query_performance_factor", "2x"),
					resource.TestCheckResourceAttr(dataSourceByName, "redis_version", "7.2"),
				),
			},
		},
	})

}

func getRedisProDbDatasourceConfig(t *testing.T, password string) string {
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	content, err := os.ReadFile("./testdata/testAccDatasourceRedisCloudProDatabase.tf")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return fmt.Sprintf(string(content), testCloudAccountName, subscriptionName, password)
}
