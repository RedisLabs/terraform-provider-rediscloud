package pro_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudProDatabase_basic(t *testing.T) {
	resourceTags, resourceTagsCheck := envchecks.BYOCResourceTagsValueAndCheck()
	cloudAccountName, cloudAccountCheck := envchecks.AWSBYOCValueAndCheck()
	subscriptionName := utils.RandomWithPrefix()
	password := acctest.RandString(20)

	const dataSourceById = "data.rediscloud_database.example-by-id"
	const dataSourceByName = "data.rediscloud_database.example-by-name"

	checkDataSource := func(name string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet(name, "db_id"),
			resource.TestCheckResourceAttrSet(name, "id"),
			resource.TestCheckResourceAttr(name, "name", "tf-database"),
			resource.TestCheckResourceAttr(name, "protocol", "redis"),
			resource.TestCheckResourceAttr(name, "region", "eu-west-1"),
			resource.TestCheckResourceAttr(name, "memory_limit_in_gb", "1"),
			resource.TestCheckResourceAttr(name, "support_oss_cluster_api", "true"),
			resource.TestCheckResourceAttr(name, "resp_version", "resp3"),
			resource.TestCheckResourceAttr(name, "data_persistence", "none"),
			resource.TestCheckResourceAttr(name, "data_eviction", "volatile-lru"),
			resource.TestCheckResourceAttr(name, "replication", "false"),
			resource.TestCheckResourceAttr(name, "throughput_measurement_by", "operations-per-second"),
			resource.TestCheckResourceAttr(name, "throughput_measurement_value", "1000"),
			resource.TestCheckResourceAttr(name, "password", password),
			resource.TestCheckResourceAttr(name, "enable_passwordless", "false"),
			resource.TestCheckResourceAttrSet(name, "public_endpoint"),
			resource.TestCheckResourceAttrSet(name, "private_endpoint"),
			resource.TestCheckResourceAttr(name, "enable_default_user", "true"),
			resource.TestCheckResourceAttr(name, "query_performance_factor", "2x"),
			resource.TestCheckResourceAttr(name, "auto_minor_version_upgrade", "true"),
			resource.TestCheckResourceAttr(name, "latest_backup_status.#", "1"),
			resource.TestCheckResourceAttr(name, "latest_import_status.#", "1"),
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, resourceTagsCheck, envchecks.RedisCloudCheck, cloudAccountCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/pro_database_data_source.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(cloudAccountName),
					"subscription_name":  config.StringVariable(subscriptionName),
					"password":           config.StringVariable(password),
					"team":               config.StringVariable(resourceTags.Team),
					"owner":              config.StringVariable(resourceTags.Owner),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDataSource(dataSourceById),
					checkDataSource(dataSourceByName),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceById, tfjsonpath.New("alert"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(dataSourceById, tfjsonpath.New("module"), knownvalue.SetSizeExact(0)),
					statecheck.ExpectKnownValue(dataSourceById, tfjsonpath.New("tags"), knownvalue.MapSizeExact(0)),
					statecheck.ExpectKnownValue(dataSourceById, tfjsonpath.New("hashing_policy"), knownvalue.ListSizeExact(0)),
				},
			},
		},
	})
}
