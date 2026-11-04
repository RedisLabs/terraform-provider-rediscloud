package aclrole_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const aclRoleResourceName = "rediscloud_acl_role.test"

func TestAccResourceRedisCloudAclRole_CRUDI(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	prefix := utils.RandomWithPrefix()
	subscriptionName := prefix + "-subscription"
	databasePassword := prefix + "aA.1"
	ruleName := prefix + "-rule"
	roleName := prefix + "-test-role"
	roleNameUpdated := roleName + "-updated"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		// Sometimes after deletion, the entity 'flickers'
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(aclRoleResourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(aclRoleResourceName, "name", roleName),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.0.name", ruleName),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(aclRoleResourceName, "rule.0.database.0.subscription", regexp.MustCompile(`^\d+$`)),
					resource.TestMatchResourceAttr(aclRoleResourceName, "rule.0.database.0.database", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.0.database.0.regions.#", "0"),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[aclRoleResourceName]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the role ID: %s", r.Primary.ID)
						}

						apiClient := client.MustTestClient(t)
						role, err := apiClient.Client.Roles.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(role.Name) != roleName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(role.Name))
						}
						return nil
					},
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleNameUpdated),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(aclRoleResourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(aclRoleResourceName, "name", roleNameUpdated),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.0.name", ruleName),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(aclRoleResourceName, "rule.0.database.0.subscription", regexp.MustCompile(`^\d+$`)),
					resource.TestMatchResourceAttr(aclRoleResourceName, "rule.0.database.0.database", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(aclRoleResourceName, "rule.0.database.0.regions.#", "0"),
				),
			},
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleNameUpdated),
				},
				ResourceName:      aclRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// State written by the last SDKv2 release must plan cleanly with the Framework implementation.
// Both steps address the provider by its registry source so that the SDKv2 state and the
// Framework provider share the same provider address.
func TestAccResourceRedisCloudAclRole_sdkStateCompatibility(t *testing.T) {
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "redislabs")

	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	prefix := utils.RandomWithPrefix()
	subscriptionName := prefix + "-subscription"
	databasePassword := prefix + "aA.1"
	ruleName := prefix + "-rule"
	roleName := prefix + "-test-role"

	resource.Test(t, resource.TestCase{
		PreCheck: envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/resource_sdk_v2_20_0.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleName),
				},
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/resource_registry_source.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleName),
				},
				PlanOnly: true,
			},
			// A plan-only step leaves rediscloud_subscription at schema version 0, which the
			// post-test destroy cannot read. Applying stores the upgraded state.
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/resource_registry_source.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleName),
				},
			},
		},
	})
}
