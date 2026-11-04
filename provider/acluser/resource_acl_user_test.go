package acluser_test

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

const aclUserResourceName = "rediscloud_acl_user.test"

func TestAccResourceRedisCloudAclUser_CRUDI(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	prefix := utils.RandomWithPrefix()
	subscriptionName := prefix + "-subscription"
	databasePassword := prefix + "aA.1"
	roleName := prefix + "-role"
	roleNameUpdated := roleName + "-updated"
	userName := prefix + "-test-user"
	userNameUpdated := userName + "-updated"
	userPassword := prefix + "aA.1"
	userPasswordUpdated := userPassword + "-updated"

	identifier := ""

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"role_name":          config.StringVariable(roleName),
					"user_name":          config.StringVariable(userName),
					"user_password":      config.StringVariable(userPassword),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(aclUserResourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(aclUserResourceName, "name", userName),
					resource.TestCheckResourceAttr(aclUserResourceName, "role", roleName),
					resource.TestCheckResourceAttr(aclUserResourceName, "password", userPassword),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[aclUserResourceName]
						identifier = r.Primary.ID

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the user ID: %s", r.Primary.ID)
						}

						apiClient := client.MustTestClient(t)
						user, err := apiClient.Client.Users.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(user.Name) != userName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(user.Name))
						}
						if redis.StringValue(user.Role) != roleName {
							return fmt.Errorf("unexpected role value: %s", redis.StringValue(user.Role))
						}
						return nil
					},
				),
			},
			// Renaming the role updates the user in place.
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"role_name":          config.StringVariable(roleNameUpdated),
					"user_name":          config.StringVariable(userName),
					"user_password":      config.StringVariable(userPassword),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aclUserResourceName, "name", userName),
					resource.TestCheckResourceAttr(aclUserResourceName, "role", roleNameUpdated),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[aclUserResourceName]
						if r.Primary.ID != identifier {
							return fmt.Errorf("entity should have the same identifier, but has changed from %s to %s", identifier, r.Primary.ID)
						}
						return nil
					},
				),
			},
			// A name change replaces the user.
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"role_name":          config.StringVariable(roleNameUpdated),
					"user_name":          config.StringVariable(userNameUpdated),
					"user_password":      config.StringVariable(userPassword),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aclUserResourceName, "name", userNameUpdated),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[aclUserResourceName]
						if r.Primary.ID == identifier {
							return fmt.Errorf("entity should have a new identifier, but is still: %s", identifier)
						}
						identifier = r.Primary.ID
						return nil
					},
				),
			},
			// A password change replaces the user.
			{
				ConfigFile: config.StaticFile("testdata/resource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"role_name":          config.StringVariable(roleNameUpdated),
					"user_name":          config.StringVariable(userNameUpdated),
					"user_password":      config.StringVariable(userPasswordUpdated),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aclUserResourceName, "password", userPasswordUpdated),
					func(s *terraform.State) error {
						r := s.RootModule().Resources[aclUserResourceName]
						if r.Primary.ID == identifier {
							return fmt.Errorf("entity should have a new identifier, but is still: %s", identifier)
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
					"role_name":          config.StringVariable(roleNameUpdated),
					"user_name":          config.StringVariable(userNameUpdated),
					"user_password":      config.StringVariable(userPasswordUpdated),
				},
				ResourceName:            aclUserResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// State written by the last SDKv2 release must plan cleanly with the Framework implementation.
// Both steps address the provider by its registry source so that the SDKv2 state and the
// Framework provider share the same provider address.
func TestAccResourceRedisCloudAclUser_sdkStateCompatibility(t *testing.T) {
	t.Setenv("TF_ACC_PROVIDER_NAMESPACE", "redislabs")

	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	prefix := utils.RandomWithPrefix()
	subscriptionName := prefix + "-subscription"
	databasePassword := prefix + "aA.1"
	roleName := prefix + "-role"
	userName := prefix + "-test-user"
	userPassword := prefix + "aA.1"

	resource.Test(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		CheckDestroy: testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/resource_sdk_v2_20_0.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"database_password":  config.StringVariable(databasePassword),
					"role_name":          config.StringVariable(roleName),
					"user_name":          config.StringVariable(userName),
					"user_password":      config.StringVariable(userPassword),
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
					"role_name":          config.StringVariable(roleName),
					"user_name":          config.StringVariable(userName),
					"user_password":      config.StringVariable(userPassword),
				},
				PlanOnly: true,
			},
		},
	})
}

func testAccCheckAclUserDestroy(s *terraform.State) error {
	apiClient, err := client.GetTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_acl_user" {
			continue
		}

		id, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		users, err := apiClient.Client.Users.List(context.TODO())
		if err != nil {
			return err
		}

		for _, user := range users {
			if redis.IntValue(user.ID) == id {
				return fmt.Errorf("user %d still exists", id)
			}
		}
	}

	return nil
}
