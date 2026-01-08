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
// The TestAccActiveActiveSubscriptionDatabase_BlockPublicEndpoints test below covers the case where
// source_ips are explicitly set and verifies they are preserved across public_endpoint_access changes.

func TestAccActiveActiveSubscriptionDatabase_DefaultSourceIPs_PrivateAccess(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	password := acctest.RandString(20)
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentDisabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_disabled_default_source_ips.tf")
	configDisabled := fmt.Sprintf(contentDisabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configDisabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the subscription has public_endpoint_access disabled
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "false"),

					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Global source IPs should default to RFC1918 private ranges when public_endpoint_access=false
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "4"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "10.0.0.0/8"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "172.16.0.0/12"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "192.168.0.0/16"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "100.64.0.0/10"),

					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "global_source_ips.#", "4"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "10.0.0.0/8"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "172.16.0.0/12"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "192.168.0.0/16"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "100.64.0.0/10"),
				),
			},
		},
	})
}

func TestAccActiveActiveSubscriptionDatabase_DefaultSourceIPs_PublicAccess(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	password := acctest.RandString(20)
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentEnabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_enabled_default_source_ips.tf")
	configEnabled := fmt.Sprintf(contentEnabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the subscription has public_endpoint_access enabled
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "true"),

					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Global source IPs should default to public access when public_endpoint_access=true
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "0.0.0.0/0"),

					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(datasourceName, "global_source_ips.*", "0.0.0.0/0"),
				),
			},
		},
	})
}

func TestAccActiveActiveSubscriptionDatabase_BlockPublicEndpoints(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	password := acctest.RandString(20)
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix)

	contentDisabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_disabled.tf")
	configDisabled := fmt.Sprintf(contentDisabled, subscriptionName, password)

	contentEnabled := utils.GetTestConfig(t, "./activeactive/testdata/public_endpoint_enabled.tf")
	configEnabled := fmt.Sprintf(contentEnabled, subscriptionName, password)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: configDisabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the subscription has public_endpoint_access disabled
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "false"),

					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Global source IPs should be explicitly set
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "192.168.0.0/16"),
					resource.TestCheckResourceAttr(databaseResource, "global_enable_default_user", "true"),

					resource.TestCheckResourceAttr(databaseResource, "override_region.#", "2"),

					// us-east-1 has explicit override of source_ips
					resource.TestCheckTypeSetElemNestedAttrs(databaseResource, "override_region.*", map[string]string{
						"name":                         "us-east-1",
						"override_global_source_ips.#": "1",
					}),
					resource.TestCheckTypeSetElemAttr(databaseResource, "override_region.*.override_global_source_ips.*", "172.16.0.0/16"),

					// us-east-2 has no source_ips override (field is absent from state when empty)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResource, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),

					// Data source checks
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					resource.TestCheckResourceAttr(datasourceName, "dataset_size_in_gb", "1"),
					// TODO: Data source global_source_ips assertion removed - the data source uses
					// the first region's source_ips which may be an override value, not the global value.
					// The API returns regions in arbitrary order, so this needs a proper fix
					// (potentially requiring API changes to return a distinct global value).
				),
			},
			{
				Config: configEnabled,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the subscription has public_endpoint_access enabled
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "true"),
					// Database resource checks
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),

					// Global source IPs should be the same (explicitly set in testdata)
					resource.TestCheckResourceAttr(databaseResource, "global_source_ips.#", "1"),
					resource.TestCheckTypeSetElemAttr(databaseResource, "global_source_ips.*", "192.168.0.0/16"),
					resource.TestCheckResourceAttr(databaseResource, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(databaseResource, "override_region.#", "2"),
					// us-east-1 should still have the source_ips override
					resource.TestCheckTypeSetElemNestedAttrs(databaseResource, "override_region.*", map[string]string{
						"name":                         "us-east-1",
						"override_global_source_ips.#": "1",
					}),
					resource.TestCheckTypeSetElemAttr(databaseResource, "override_region.*.override_global_source_ips.*", "172.16.0.0/16"),
					// Data source checks after update
					resource.TestCheckResourceAttr(datasourceName, "name", subscriptionName),
					// TODO: Data source global_source_ips assertion removed - see TODO above.
				),
			},
		},
	})

}
