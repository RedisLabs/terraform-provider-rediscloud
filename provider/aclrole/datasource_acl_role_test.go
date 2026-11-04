package aclrole_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudAclRole_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	prefix := utils.RandomWithPrefix()
	ruleName := prefix + "-rule"
	roleName := prefix + "-role"

	const dataSourceName = "data.rediscloud_acl_role.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/datasource_basic.tf"),
				ConfigVariables: config.Variables{
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(prefix + "-subscription"),
					"database_password":  config.StringVariable(prefix + "aA.1"),
					"rule_name":          config.StringVariable(ruleName),
					"role_name":          config.StringVariable(roleName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(dataSourceName, "name", roleName),
					resource.TestCheckResourceAttr(dataSourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "rule.0.name", ruleName),
					resource.TestCheckResourceAttr(dataSourceName, "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr(dataSourceName, "rule.0.database.0.subscription", regexp.MustCompile(`^\d+$`)),
					resource.TestMatchResourceAttr(dataSourceName, "rule.0.database.0.database", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttr(dataSourceName, "rule.0.database.0.regions.#", "0"),
				),
			},
		},
	})
}
