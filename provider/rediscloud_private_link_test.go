package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const testPrivateLinkConfigFile = "./privatelink/testdata/pro_private_link.tf"
const testPrivateLinkConfigWithoutPrivateLinkFile = "./privatelink/testdata/pro_private_link_without_privatelink.tf"

func TestAccResourceRedisCloudPrivateLink_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")

	const resourceName = "rediscloud_private_link.pro_private_link"
	const subscriptionResourceName = "rediscloud_subscription.pro_subscription"
	const datasourceName = "data.rediscloud_private_link.pro_private_link"

	// Generate names reused across configs
	subName := testRandomWithPrefix() + "-pro-private-link"
	shareName := testRandomWithPrefix() + "-privatelink"
	password := acctest.RandString(20)
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create everything including privatelink
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: config.Variables{
					"subscription_name":  config.StringVariable(subName),
					"cloud_account_name": config.StringVariable(exampleCloudAccountName),
					"share_name":         config.StringVariable(shareName),
					"database_password":  config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "share_name"),
					resource.TestCheckResourceAttr(resourceName, "principal.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(resourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(resourceName, "databases.#"),

					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttr(datasourceName, "principals.#", "2"),
					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(datasourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(datasourceName, "databases.#"),
				),
			},
			// Step 2: Import test
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: config.Variables{
					"subscription_name":  config.StringVariable(subName),
					"cloud_account_name": config.StringVariable(exampleCloudAccountName),
					"share_name":         config.StringVariable(shareName),
					"database_password":  config.StringVariable(password),
				},
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Remove privatelink, verify deletion via API
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigWithoutPrivateLinkFile),
				ConfigVariables: config.Variables{
					"subscription_name":  config.StringVariable(subName),
					"cloud_account_name": config.StringVariable(exampleCloudAccountName),
					"database_password":  config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(subscriptionResourceName, "id"),
					testAccCheckPrivateLinkDeleted(subscriptionResourceName),
				),
			},
		},
	})
}

func testAccCheckPrivateLinkDeleted(subscriptionResourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subResource, ok := s.RootModule().Resources[subscriptionResourceName]
		if !ok {
			return fmt.Errorf("subscription not found: %s", subscriptionResourceName)
		}

		subId, err := strconv.Atoi(subResource.Primary.ID)
		if err != nil {
			return err
		}

		apiClient, err := getTestClient()
		if err != nil {
			return err
		}

		_, err = apiClient.Client.PrivateLink.GetPrivateLink(context.TODO(), subId)
		if err == nil {
			return fmt.Errorf("privatelink for subscription %d still exists after deletion", subId)
		}

		var notFound *pl.NotFound
		if !errors.As(err, &notFound) {
			return fmt.Errorf("unexpected error checking privatelink: %w", err)
		}

		return nil
	}
}

// TestAccResourceRedisCloudPrivateLink_PortConsistency verifies that the port returned
// in the private link databases output matches the port in the database's private_endpoint.
// This test was added to catch a bug where the private link API returns a different port
// than what's shown in the database's private_endpoint for Pro subscriptions.
func TestAccResourceRedisCloudPrivateLink_PortConsistency(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")

	const databaseResourceName = "rediscloud_subscription_database.pro_database"
	const privateLinkResourceName = "rediscloud_private_link.pro_private_link"

	subName := testRandomWithPrefix() + "-pro-private-link"
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	shareName := testRandomWithPrefix() + "-port-test"
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: config.Variables{
					"subscription_name":  config.StringVariable(subName),
					"cloud_account_name": config.StringVariable(exampleCloudAccountName),
					"share_name":         config.StringVariable(shareName),
					"database_password":  config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the private link has at least one database entry
					resource.TestCheckResourceAttrSet(privateLinkResourceName, "databases.#"),
					// Custom check to verify port consistency
					testCheckPrivateLinkPortMatchesDatabaseEndpoint(databaseResourceName, privateLinkResourceName),
				),
			},
		},
	})
}

// testCheckPrivateLinkPortMatchesDatabaseEndpoint returns a TestCheckFunc that verifies
// the port in the private link's databases output matches the port from the database's
// private_endpoint string.
func testCheckPrivateLinkPortMatchesDatabaseEndpoint(databaseResourceName, privateLinkResourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Get the database resource
		dbResource, ok := s.RootModule().Resources[databaseResourceName]
		if !ok {
			return fmt.Errorf("database resource not found: %s", databaseResourceName)
		}

		// Get the private_endpoint from the database
		privateEndpoint := dbResource.Primary.Attributes["private_endpoint"]
		if privateEndpoint == "" {
			return fmt.Errorf("database private_endpoint is empty")
		}

		// Extract port from private_endpoint (format: "hostname:port")
		portRegex := regexp.MustCompile(`:(\d+)$`)
		matches := portRegex.FindStringSubmatch(privateEndpoint)
		if len(matches) < 2 {
			return fmt.Errorf("could not extract port from private_endpoint: %s", privateEndpoint)
		}
		dbPort := matches[1]

		// Get the private link resource
		plResource, ok := s.RootModule().Resources[privateLinkResourceName]
		if !ok {
			return fmt.Errorf("private link resource not found: %s", privateLinkResourceName)
		}

		// Check that at least one database entry has a matching port
		dbsCount := plResource.Primary.Attributes["databases.#"]
		if dbsCount == "0" {
			return fmt.Errorf("private link has no database entries")
		}

		count, err := strconv.Atoi(dbsCount)
		if err != nil {
			return fmt.Errorf("could not parse databases count: %w", err)
		}

		for i := 0; i < count; i++ {
			plPort := plResource.Primary.Attributes[fmt.Sprintf("databases.%d.port", i)]
			if plPort == dbPort {
				return nil
			}
		}

		return fmt.Errorf("no private link database entry has port %s matching database private_endpoint %s", dbPort, privateEndpoint)
	}
}
