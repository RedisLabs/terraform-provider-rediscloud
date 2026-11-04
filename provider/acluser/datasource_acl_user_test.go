package acluser_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudAclUser_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	prefix := utils.RandomWithPrefix()
	subscriptionName := prefix + "-subscription"
	databasePassword := prefix + "aA.1"
	roleName := prefix + "-role"
	userName := prefix + "-test-user"
	userPassword := prefix + "aA.1"

	const dataSourceName = "data.rediscloud_acl_user.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_basic.tf"),
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
					resource.TestMatchResourceAttr(dataSourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(dataSourceName, "name", userName),
					resource.TestCheckResourceAttr(dataSourceName, "role", roleName),
				),
			},
		},
	})
}
