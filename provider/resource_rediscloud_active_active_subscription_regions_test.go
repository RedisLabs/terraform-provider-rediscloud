package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudActiveActiveSubscriptionRegions_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TEST_SUB_ACTIVE_ACTIVE")

	subName := testRandomWithPrefix() + "-regions-test"
	dbName := testRandomWithPrefix() + "-regions" + "-db"
	dbPass := acctest.RandString(20)
	const resourceName = "rediscloud_active_active_subscription_regions.example"
	const datasourceRegionName = "data.rediscloud_active_active_subscription_regions.example"

	var subId int

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./activeactive/testdata/aa_regions_create.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subName),
					"database_name":     config.StringVariable(dbName),
					"database_password": config.StringVariable(dbPass),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "region.2.region", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "region.2.networking_deployment_cidr", "10.2.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_write_operations_per_second", "1500"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_read_operations_per_second", "1500"),

					// Test the db regions datasource
					resource.TestCheckResourceAttr(datasourceRegionName, "subscription_name", subName),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.2.vpc_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.2.region", "us-west-2"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.2.networking_deployment_cidr", "10.2.0.0/24"),
					resource.TestCheckResourceAttrSet(datasourceRegionName, "regions.2.databases.0.database_id"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.2.databases.0.database_name", dbName),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.2.databases.0.read_operations_per_second", "1500"),
					resource.TestCheckResourceAttr(datasourceRegionName, "regions.2.databases.0.write_operations_per_second", "1500"),

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

						if redis.StringValue(sub.Name) != subName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(sub.Name))
						}
						return nil
					},
				),
			},
			{
				// Checks region re-created correctly
				ConfigFile: config.StaticFile("./activeactive/testdata/aa_regions_recreate.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subName),
					"database_name":     config.StringVariable(dbName),
					"database_password": config.StringVariable(dbPass),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "region.2.region", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "region.2.networking_deployment_cidr", "10.3.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_write_operations_per_second", "1500"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_read_operations_per_second", "1500"),
				),
			},
			{
				// Checks region DB updated correctly
				ConfigFile: config.StaticFile("./activeactive/testdata/aa_regions_update_db.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subName),
					"database_name":     config.StringVariable(dbName),
					"database_password": config.StringVariable(dbPass),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "region.2.region", "eu-west-2"),
					resource.TestCheckResourceAttr(resourceName, "region.2.networking_deployment_cidr", "10.3.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.database_name", dbName),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_write_operations_per_second", "1000"),
					resource.TestCheckResourceAttr(resourceName, "region.2.database.0.local_read_operations_per_second", "1000"),
				),
			},
			{
				// Checks regions deleted (eu-west-2 and us-east-2) and created (eu-west-1) correctly
				ConfigFile: config.StaticFile("./activeactive/testdata/aa_regions_remove_and_create.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subName),
					"database_name":     config.StringVariable(dbName),
					"database_password": config.StringVariable(dbPass),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "region.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "region.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "region.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(resourceName, "region.1.region", "eu-west-1"),
					resource.TestCheckResourceAttr(resourceName, "region.1.networking_deployment_cidr", "10.2.0.0/24"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"delete_regions"},
			},
		},
	})
}
