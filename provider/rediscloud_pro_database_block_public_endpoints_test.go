package provider

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRedisCloudProDatabaseBlockPublicEndpoints(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	password := acctest.RandString(20)

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentDisabled := utils.GetTestConfig(t, "./pro/testdata/pro_subscription_public_endpoint_disabled.tf")
	configDisabled := fmt.Sprintf(contentDisabled, subscriptionName, password)

	contentEnabled := utils.GetTestConfig(t, "./pro/testdata/pro_subscription_public_endpoint_enabled.tf")
	configEnabled := fmt.Sprintf(contentEnabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configDisabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "name", "example"),
					resource.TestCheckResourceAttr(databaseResource, "protocol", "redis"),
					resource.TestCheckResourceAttr(databaseResource, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(databaseResource, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(databaseResource, "data_persistence", "none"),
					resource.TestCheckResourceAttr(databaseResource, "data_eviction", "allkeys-random"),
					resource.TestCheckResourceAttr(databaseResource, "replication", "false"),
					resource.TestCheckResourceAttr(databaseResource, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(databaseResource, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(databaseResource, "password", password),
					resource.TestCheckResourceAttr(databaseResource, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(databaseResource, "redis_version", "8.2"),
					resource.TestCheckResourceAttrSet(databaseResource, "source_ips"),
					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", "example"),
					resource.TestCheckResourceAttr(datasourceName, "protocol", "redis"),
					resource.TestCheckResourceAttrSet(datasourceName, "source_ips"),
				),
			},
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "name", "example"),
					resource.TestCheckResourceAttr(databaseResource, "data_persistence", "none"),
					resource.TestCheckResourceAttrSet(databaseResource, "source_ips"),
					// Data source checks after update
					resource.TestCheckResourceAttr(datasourceName, "name", "example"),
					resource.TestCheckResourceAttrSet(datasourceName, "source_ips"),
				),
			},
		},
	})

}
