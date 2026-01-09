package provider

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const testPrivateLinkConfigFile = "./privatelink/testdata/pro_private_link.tf"

func TestAccResourceRedisCloudPrivateLink_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "AWS_TEST_CLOUD_ACCOUNT_NAME")

	const resourceName = "rediscloud_private_link.pro_private_link"
	const datasourceName = "data.rediscloud_private_link.pro_private_link"
	const datasourceScriptName = "data.rediscloud_private_link_endpoint_script.endpoint_script"

	shareName := acctest.RandomWithPrefix(testResourcePrefix) + "-privatelink"

	terraformConfig := getRedisPrivateLinkConfig(t, shareName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: terraformConfig,
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

					//resource.TestCheckResourceAttrSet(datasourceScriptName, "id"),
					//resource.TestCheckResourceAttrSet(datasourceScriptName, "subscription_id"),
					//resource.TestCheckResourceAttrSet(datasourceScriptName, "endpoint_script"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func getRedisPrivateLinkConfig(t *testing.T, shareName string) string {
	subName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-private-link"
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	password := acctest.RandString(20)
	content := utils.GetTestConfig(t, testPrivateLinkConfigFile)
	return fmt.Sprintf(content, subName, exampleCloudAccountName, shareName, password)
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

	shareName := acctest.RandomWithPrefix(testResourcePrefix) + "-port-test"
	terraformConfig := getRedisPrivateLinkConfig(t, shareName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: terraformConfig,
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
		if len(matches) != 2 {
			return fmt.Errorf("could not extract port from private_endpoint: %s", privateEndpoint)
		}
		expectedPort := matches[1]

		// Get the private link resource
		plResource, ok := s.RootModule().Resources[privateLinkResourceName]
		if !ok {
			return fmt.Errorf("private link resource not found: %s", privateLinkResourceName)
		}

		// Get the database ID from the database resource
		dbId := dbResource.Primary.Attributes["db_id"]

		// Check the databases in the private link output
		// The databases are stored as databases.# (count) and databases.<hash>.port, databases.<hash>.database_id
		databasesCount := plResource.Primary.Attributes["databases.#"]
		if databasesCount == "" || databasesCount == "0" {
			return fmt.Errorf("private link has no databases in output")
		}

		count, err := strconv.Atoi(databasesCount)
		if err != nil {
			return fmt.Errorf("could not parse databases count: %v", err)
		}

		// Iterate through the databases to find the matching one and check its port
		// TypeSet uses hash-based keys, so we need to search through all attributes
		var foundDatabase bool
		var privateLinkPort string
		var privateLinkDbId string

		for key, value := range plResource.Primary.Attributes {
			// Look for database_id attributes
			if matched, _ := regexp.MatchString(`^databases\.\d+\.database_id$`, key); matched {
				// Extract the hash from the key to find the corresponding port
				hashRegex := regexp.MustCompile(`^databases\.(\d+)\.database_id$`)
				hashMatches := hashRegex.FindStringSubmatch(key)
				if len(hashMatches) == 2 {
					hash := hashMatches[1]
					portKey := fmt.Sprintf("databases.%s.port", hash)
					if port, exists := plResource.Primary.Attributes[portKey]; exists {
						privateLinkDbId = value
						privateLinkPort = port
						foundDatabase = true

						// If the database_id matches, check the port
						if value == dbId {
							if port != expectedPort {
								return fmt.Errorf(
									"PORT MISMATCH: private link port (%s) does not match database private_endpoint port (%s)\n"+
										"  Database ID: %s\n"+
										"  Database private_endpoint: %s\n"+
										"  Private link databases[].port: %s\n"+
										"  Expected port: %s",
									port, expectedPort, dbId, privateEndpoint, port, expectedPort,
								)
							}
							// Port matches - success!
							return nil
						}
					}
				}
			}
		}

		// If we found databases but none matched our database ID, that's also a problem
		// (the database_id might be 0 which is the bug symptom)
		if foundDatabase {
			// Check if the database_id is 0 (which indicates the API isn't properly associating databases)
			if privateLinkDbId == "0" {
				// Even if database_id is 0, we should still check the port
				if count == 1 {
					// Only one database, so we can still compare ports
					if privateLinkPort != expectedPort {
						return fmt.Errorf(
							"POTENTIAL BUG - database_id is 0 AND port mismatch detected:\n"+
								"  Database private_endpoint: %s (port: %s)\n"+
								"  Private link databases[0].port: %s\n"+
								"  Private link databases[0].database_id: %s (expected: %s)\n"+
								"  This indicates the private link API is not correctly returning database information",
							privateEndpoint, expectedPort, privateLinkPort, privateLinkDbId, dbId,
						)
					}
					// Port matches but database_id is wrong - partial bug
					return fmt.Errorf(
						"PARTIAL BUG - port matches but database_id is incorrect:\n"+
							"  Database ID: %s\n"+
							"  Private link databases[0].database_id: %s\n"+
							"  Port matches: %s",
						dbId, privateLinkDbId, expectedPort,
					)
				}
			}
			return fmt.Errorf(
				"could not find database %s in private link databases output\n"+
					"  Found database_id: %s with port: %s",
				dbId, privateLinkDbId, privateLinkPort,
			)
		}

		return fmt.Errorf("no databases found in private link output")
	}
}
