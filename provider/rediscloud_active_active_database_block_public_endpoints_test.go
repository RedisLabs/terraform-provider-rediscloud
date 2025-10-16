package provider

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccActiveActiveSubscriptionDatabaseBlockPublicEndpoints(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	content := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_disabled.tf")
	configDisabled := fmt.Sprintf(content, subscriptionName)

	contentEnabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_enabled.tf")
	configEnabled := fmt.Sprintf(contentEnabled, subscriptionName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configDisabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),
					resource.TestCheckResourceAttr(databaseResource, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttr(databaseResource, "global_data_persistence", "aof-every-1-second"),
					resource.TestCheckResourceAttr(databaseResource, "global_modules.#", "1"),
					resource.TestCheckResourceAttr(databaseResource, "global_modules.0", "RedisJSON"),
					resource.TestCheckResourceAttr(databaseResource, "global_alert.#", "1"),
					resource.TestCheckResourceAttr(databaseResource, "global_alert.0.name", "dataset-size"),
					resource.TestCheckResourceAttr(databaseResource, "global_alert.0.value", "40"),
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "1"),
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.0", "192.168.0.0/16"),
					resource.TestCheckResourceAttrSet(databaseResource, "override_region.0.source_ips"),
					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttrSet(datasourceName, "global_source_ips.#"),
				),
			},
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),
					resource.TestCheckResourceAttr(databaseResource, "dataset_size_in_gb", "1"),
					resource.TestCheckResourceAttrSet(databaseResource, "global_source_ips.#"),
					// Data source checks after update
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttrSet(datasourceName, "global_source_ips.#"),
				),
			},
		},
	})

}
