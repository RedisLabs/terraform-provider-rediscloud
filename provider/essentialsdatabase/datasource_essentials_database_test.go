package essentialsdatabase_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	databaseResourceName     = "rediscloud_essentials_database.example"
	byNameDatasourceName     = "data.rediscloud_essentials_database.by_name"
	byIDDatasourceName       = "data.rediscloud_essentials_database.by_id"
	databaseIDPattern        = `^\d+/\d+$`
	essentialsDatasourceTest = "testdata/datasource_basic"
)

// resource.Test rather than ParallelTest: Redis Cloud allows only one essentials subscription per account.
// The first step creates the database alone so the data sources are not read before it exists.
func TestAccDataSourceRedisCloudEssentialsDatabase_basic(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	password := acctest.RandString(20)

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(essentialsDatasourceTest + "/database.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
				},
				Check: resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
			},
			{
				ConfigDirectory: config.StaticDirectory(essentialsDatasourceTest),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkDataSource(byNameDatasourceName, databaseName, password),
					checkDataSource(byIDDatasourceName, databaseName, password),
				),
			},
			{
				ConfigDirectory: config.StaticDirectory(essentialsDatasourceTest),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"database_name":     config.StringVariable(databaseName),
					"password":          config.StringVariable(password),
					"lookup_name":       config.StringVariable(databaseName + "-missing"),
				},
				ExpectError: regexp.MustCompile("Your query returned no results"),
			},
		},
	})
}

func checkDataSource(datasourceName, databaseName, password string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestMatchResourceAttr(datasourceName, "id", regexp.MustCompile(databaseIDPattern)),
		resource.TestCheckResourceAttrPair(datasourceName, "id", databaseResourceName, "id"),
		resource.TestCheckResourceAttrPair(datasourceName, "db_id", databaseResourceName, "db_id"),
		resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
		resource.TestCheckResourceAttr(datasourceName, "name", databaseName),
		resource.TestCheckResourceAttr(datasourceName, "protocol", "stack"),
		resource.TestCheckResourceAttr(datasourceName, "cloud_provider", "AWS"),
		resource.TestCheckResourceAttr(datasourceName, "region", "us-east-1"),
		resource.TestCheckResourceAttrSet(datasourceName, "redis_version"),
		resource.TestCheckResourceAttrSet(datasourceName, "redis_version_compliance"),
		resource.TestCheckResourceAttr(datasourceName, "resp_version", "resp3"),
		resource.TestCheckResourceAttr(datasourceName, "data_persistence", "none"),
		resource.TestCheckResourceAttr(datasourceName, "data_eviction", "volatile-lru"),
		resource.TestCheckResourceAttr(datasourceName, "replication", "false"),
		resource.TestCheckResourceAttrSet(datasourceName, "activated_on"),
		resource.TestCheckResourceAttrSet(datasourceName, "public_endpoint"),
		resource.TestCheckResourceAttr(datasourceName, "private_endpoint", ""),
		resource.TestCheckResourceAttr(datasourceName, "source_ips.#", "0"),
		resource.TestCheckResourceAttr(datasourceName, "replica.#", "0"),
		resource.TestCheckResourceAttr(datasourceName, "enable_default_user", "true"),
		resource.TestCheckResourceAttr(datasourceName, "password", password),
		resource.TestCheckResourceAttrSet(datasourceName, "modules.#"),
		resource.TestCheckResourceAttrSet(datasourceName, "alert.#"),
		resource.TestCheckResourceAttr(datasourceName, "memory_limit_in_gb", "0"),
		resource.TestCheckResourceAttr(datasourceName, "support_oss_cluster_api", "false"),
		resource.TestCheckResourceAttr(datasourceName, "external_endpoint_for_oss_cluster_api", "false"),
		resource.TestCheckResourceAttr(datasourceName, "enable_database_clustering", "false"),
		resource.TestCheckResourceAttr(datasourceName, "regex_rules.#", "0"),
		resource.TestCheckResourceAttr(datasourceName, "enable_tls", "false"),
		resource.TestCheckResourceAttr(datasourceName, "tags.%", "2"),
		resource.TestCheckResourceAttr(datasourceName, "tags.environment", "production"),
		resource.TestCheckResourceAttr(datasourceName, "tags.cost_center", "0700"),
	)
}
