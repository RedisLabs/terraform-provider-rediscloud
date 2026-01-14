package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// NOTE: We do not test transitioning from public_endpoint_access=false to public_endpoint_access=true
// with default source_ips because of an API limitation: the API sets default source_ips at database
// creation time and does NOT automatically update them when the subscription's public_endpoint_access
// changes. Users must explicitly set source_ips when changing public_endpoint_access.

func TestAccRedisCloudProDatabase_DefaultSourceIPs_PrivateAccess(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	password := acctest.RandString(20)
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentDisabled := utils.GetTestConfig(t, "./pro/testdata/pro_subscription_public_endpoint_disabled.tf")
	configDisabled := fmt.Sprintf(contentDisabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
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
		},
	})
}

func TestAccRedisCloudProDatabase_DefaultSourceIPs_PublicAccess(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	password := acctest.RandString(20)
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentEnabled := utils.GetTestConfig(t, "./pro/testdata/pro_subscription_public_endpoint_enabled.tf")
	configEnabled := fmt.Sprintf(contentEnabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify subscription has public_endpoint_access enabled
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "public_endpoint_access", "true"),

					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Source IPs should default to public access when public_endpoint_access=true
					resource.TestCheckResourceAttr(databaseResource, "source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "source_ips.*", "0.0.0.0/0"),

					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "source_ips.*", "0.0.0.0/0"),
				),
			},
		},
	})
}
