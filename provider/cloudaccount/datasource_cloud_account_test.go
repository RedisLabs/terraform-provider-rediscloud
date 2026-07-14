package cloudaccount_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAccDataSourceRedisCloudCloudAccount_basic(t *testing.T) {

	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()

	const testCloudAccount = "data.rediscloud_cloud_account.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // test doesn't create a resource at the moment, so don't need to check anything
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("./testdata/datasource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(cloudAccountName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(testCloudAccount, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(testCloudAccount, "provider_type", "AWS"),
					resource.TestCheckResourceAttr(testCloudAccount, "name", cloudAccountName),
					resource.TestCheckResourceAttrSet(testCloudAccount, "access_key_id"),
				),
			},
		},
	})
}
