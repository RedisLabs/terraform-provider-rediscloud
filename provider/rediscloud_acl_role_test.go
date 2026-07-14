package provider_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudAclRole_CRUDI(t *testing.T) {

	prefix := testRandomWithPrefix()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabasePassword := prefix + "aA.1"
	exampleRuleName := prefix + "-rule"

	testRoleName := prefix + "-test-role"
	testRoleNameUpdated := testRoleName + "-updated"

	proSubBoilerPlate := utils.GetTestConfig(t, "./pro/testdata/pro_subscription_boilerplate.tf")
	proSubBoilerPlateFormatted := fmt.Sprintf(proSubBoilerPlate, cloudAccountName, exampleSubscriptionName, exampleDatabasePassword)

	testCreateTerraform := proSubBoilerPlateFormatted + testAccResourceRedisCloudProDatabaseAcl +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleName)

	testUpdateTerraform := proSubBoilerPlateFormatted + testAccResourceRedisCloudProDatabaseAcl +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleNameUpdated)

	const testAclRole = "rediscloud_acl_role.test"
	const testAclRoleData = "data.rediscloud_acl_role.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		// Sometimes after deletion, the entity 'flickers'
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			// Test role creation including association with database
			{
				Config: testCreateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(testAclRole, "name", testRoleName),
					resource.TestCheckResourceAttr(testAclRole, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.0.regions.#", "0"),

					// Test role exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources[testAclRole]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the role ID: %s", redis.StringValue(&r.Primary.ID))
						}

						client := sharedTestClient(t)
						role, err := client.Client.Roles.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(role.Name) != testRoleName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(role.Name))
						}

						return nil
					},

					// Test the datasource
					resource.TestMatchResourceAttr(
						testAclRoleData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "name", testRoleName),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test role update
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(testAclRole, "name", testRoleNameUpdated),
					resource.TestCheckResourceAttr(testAclRole, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRole, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRole, "rule.0.database.0.regions.#", "0"),

					// Test the datasource
					resource.TestMatchResourceAttr(
						testAclRoleData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "name", testRoleNameUpdated),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.#", "1"),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr(testAclRoleData, "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testAclRoleData, "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test that the role is imported successfully
			{
				Config:            fmt.Sprintf(testRole, testRoleNameUpdated),
				ResourceName:      testAclRole,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const referencableRule = `
resource "rediscloud_acl_rule" "example" {
    name = "%s"
    rule = "+@all"
}
`

const testRole = `
resource "rediscloud_acl_role" "test" {
	name = "%s"
	rule {
		name = rediscloud_acl_rule.example.name
		database {
			subscription = rediscloud_subscription.example.id
			database = rediscloud_subscription_database.example.db_id
		}
	}
}

data "rediscloud_acl_role" "test" {
	name = rediscloud_acl_role.test.name
}
`
