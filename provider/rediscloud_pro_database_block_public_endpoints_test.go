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
					// Verify subscription has public_endpoint_access disabled
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "public_endpoint_access", "false"),
					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Source IPs should default to RFC1918 private ranges when public_endpoint_access=false
					resource.TestCheckResourceAttr(databaseResource, "source_ips.#", "4"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "source_ips.*", "10.0.0.0/8"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "source_ips.*", "172.16.0.0/12"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "source_ips.*", "192.168.0.0/16"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "source_ips.*", "100.64.0.0/10"),
					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(datasourceName, "source_ips.#", "4"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "source_ips.*", "10.0.0.0/8"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "source_ips.*", "172.16.0.0/12"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "source_ips.*", "192.168.0.0/16"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "source_ips.*", "100.64.0.0/10"),
				),
			},
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify subscription has public_endpoint_access enabled
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "public_endpoint_access", "true"),
					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", "example"),
					resource.TestCheckResourceAttr(databaseResource, "data_persistence", "none"),
					// Source IPs should default to public access when public_endpoint_access=true
					resource.TestCheckResourceAttr(databaseResource, "source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "source_ips.*", "0.0.0.0/0"),
					// Data source checks after update
					resource.TestCheckResourceAttr(datasourceName, "name", "example"),
					resource.TestCheckResourceAttr(datasourceName, "source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "source_ips.*", "0.0.0.0/0"),
				),
			},
		},
	})

}
