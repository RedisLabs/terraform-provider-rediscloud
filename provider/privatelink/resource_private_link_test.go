package privatelink_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	pl "github.com/RedisLabs/rediscloud-go-api/service/privatelink"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	testPrivateLinkConfigFile                   = "testdata/resource_private_link.tf"
	testPrivateLinkConfigWithoutPrivateLinkFile = "testdata/resource_private_link_without_privatelink.tf"
	privateLinkResourceName                     = "rediscloud_private_link.pro_private_link"
	privateLinkSubscriptionResourceName         = "rediscloud_subscription.pro_subscription"
)

func principalVariable(principal, alias string) config.Variable {
	attributes := map[string]config.Variable{
		"principal":      config.StringVariable(principal),
		"principal_type": config.StringVariable("aws_account"),
	}
	if alias != "" {
		attributes["principal_alias"] = config.StringVariable(alias)
	}
	return config.ObjectVariable(attributes)
}

func TestAccResourceRedisCloudPrivateLink_CRUDI(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	subscriptionName := utils.RandomWithPrefix() + "-pro-private-link"
	shareName := utils.RandomWithPrefix() + "-privatelink"
	password := acctest.RandString(20)

	baseVariables := func() config.Variables {
		return config.Variables{
			"subscription_name":  config.StringVariable(subscriptionName),
			"cloud_account_name": config.StringVariable(cloudAccountName),
			"database_password":  config.StringVariable(password),
			"team":               config.StringVariable(resourceTags.Team),
			"owner":              config.StringVariable(resourceTags.Owner),
		}
	}
	withPrivateLink := func(principals ...config.Variable) config.Variables {
		variables := baseVariables()
		variables["share_name"] = config.StringVariable(shareName)
		variables["principals"] = config.ListVariable(principals...)
		return variables
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: withPrivateLink(
					principalVariable("123456789012", "principal 1"),
					principalVariable("234567890123", "principal 2"),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(privateLinkResourceName, "id", privateLinkSubscriptionResourceName, "id"),
					resource.TestCheckResourceAttrPair(privateLinkResourceName, "subscription_id", privateLinkSubscriptionResourceName, "id"),
					resource.TestCheckResourceAttr(privateLinkResourceName, "share_name", shareName),
					resource.TestCheckResourceAttr(privateLinkResourceName, "principal.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(privateLinkResourceName, "principal.*", map[string]string{
						"principal":       "123456789012",
						"principal_type":  "aws_account",
						"principal_alias": "principal 1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(privateLinkResourceName, "principal.*", map[string]string{
						"principal":       "234567890123",
						"principal_type":  "aws_account",
						"principal_alias": "principal 2",
					}),
					resource.TestCheckResourceAttrSet(privateLinkResourceName, "resource_configuration_id"),
					resource.TestCheckResourceAttrSet(privateLinkResourceName, "resource_configuration_arn"),
					resource.TestCheckResourceAttrSet(privateLinkResourceName, "share_arn"),
					resource.TestCheckResourceAttrSet(privateLinkResourceName, "connections.#"),
					resource.TestCheckResourceAttrSet(privateLinkResourceName, "databases.#"),
				),
			},
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: withPrivateLink(
					principalVariable("123456789012", "principal 1"),
					principalVariable("345678901234", ""),
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkResourceName, "principal.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(privateLinkResourceName, "principal.*", map[string]string{
						"principal":       "123456789012",
						"principal_type":  "aws_account",
						"principal_alias": "principal 1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(privateLinkResourceName, "principal.*", map[string]string{
						"principal":      "345678901234",
						"principal_type": "aws_account",
					}),
				),
			},
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: withPrivateLink(
					principalVariable("123456789012", "principal 1"),
					principalVariable("345678901234", ""),
				),
				ResourceName:      privateLinkResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ConfigFile:      config.StaticFile(testPrivateLinkConfigWithoutPrivateLinkFile),
				ConfigVariables: baseVariables(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(privateLinkSubscriptionResourceName, "id"),
					testAccCheckPrivateLinkDeleted(privateLinkSubscriptionResourceName),
				),
			},
		},
	})
}

// TestAccResourceRedisCloudPrivateLink_MigrateFromSDK checks that state written by
// the last SDKv2 release plans cleanly with the Plugin Framework implementation.
func TestAccResourceRedisCloudPrivateLink_MigrateFromSDK(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	variables := config.Variables{
		"subscription_name":  config.StringVariable(utils.RandomWithPrefix() + "-pro-private-link"),
		"cloud_account_name": config.StringVariable(cloudAccountName),
		"database_password":  config.StringVariable(acctest.RandString(20)),
		"share_name":         config.StringVariable(utils.RandomWithPrefix() + "-privatelink"),
		"team":               config.StringVariable(resourceTags.Team),
		"owner":              config.StringVariable(resourceTags.Owner),
		"principals": config.ListVariable(
			principalVariable("123456789012", "principal 1"),
			principalVariable("234567890123", ""),
		),
	}

	// ExternalProviders cannot be combined with ConfigFile, so both steps use the file content.
	terraformConfig := utils.GetTestConfig(t, testPrivateLinkConfigFile)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		CheckDestroy: testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"rediscloud": {
						Source:            "RedisLabs/rediscloud",
						VersionConstraint: "2.20.0",
					},
				},
				Config:          terraformConfig,
				ConfigVariables: variables,
				Check:           resource.TestCheckResourceAttr(privateLinkResourceName, "principal.#", "2"),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				Config:                   terraformConfig,
				ConfigVariables:          variables,
				PlanOnly:                 true,
			},
		},
	})
}

// TestAccResourceRedisCloudPrivateLink_PortConsistency verifies that the port returned
// in the private link databases output matches the port in the database's private_endpoint.
// This test was added to catch a bug where the private link API returns a different port
// than what's shown in the database's private_endpoint for Pro subscriptions.
func TestAccResourceRedisCloudPrivateLink_PortConsistency(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	const databaseResourceName = "rediscloud_subscription_database.pro_database"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(testPrivateLinkConfigFile),
				ConfigVariables: config.Variables{
					"subscription_name":  config.StringVariable(utils.RandomWithPrefix() + "-pro-private-link"),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"database_password":  config.StringVariable(acctest.RandString(20)),
					"share_name":         config.StringVariable(utils.RandomWithPrefix() + "-port-test"),
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"principals": config.ListVariable(
						principalVariable("123456789012", "principal 1"),
						principalVariable("234567890123", "principal 2"),
					),
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

		apiClient, err := client.GetTestClient()
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
			return fmt.Errorf("could not parse databases count: %w", err)
		}

		// Iterate through the databases to find the matching one and check its port
		// TypeSet uses hash-based keys, so we need to search through all attributes
		var foundDatabase bool
		var privateLinkPort string
		var privateLinkDbId string

		databaseIdKeyRegex := regexp.MustCompile(`^databases\.(\d+)\.database_id$`)

		for key, value := range plResource.Primary.Attributes {
			hashMatches := databaseIdKeyRegex.FindStringSubmatch(key)
			if hashMatches != nil {
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
