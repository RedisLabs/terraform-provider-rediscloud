package essentialsdatabase_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	crudiConfig        = "testdata/resource_crudi/main.tf"
	defaultUserConfig  = "testdata/resource_default_user/main.tf"
	redisVersionConfig = "testdata/resource_redis_version/main.tf"
)

// timeouts is config-only, password is sensitive and enable_payg_features is never returned by the API.
var databaseImportVerifyIgnore = []string{"password", "enable_payg_features", "timeouts"}

func crudiVariables(subscriptionName, databaseName, password string, tags map[string]string) config.Variables {
	tagVariables := map[string]config.Variable{}
	for k, v := range tags {
		tagVariables[k] = config.StringVariable(v)
	}
	return config.Variables{
		"subscription_name": config.StringVariable(subscriptionName),
		"database_name":     config.StringVariable(databaseName),
		"password":          config.StringVariable(password),
		"tags":              config.MapVariable(tagVariables),
	}
}

func checkBasicDatabase(databaseName, password string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestMatchResourceAttr(databaseResourceName, "id", regexp.MustCompile(databaseIDPattern)),
		resource.TestCheckResourceAttrSet(databaseResourceName, "subscription_id"),
		resource.TestCheckResourceAttrSet(databaseResourceName, "db_id"),
		resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
		resource.TestCheckResourceAttr(databaseResourceName, "protocol", "stack"),
		resource.TestCheckResourceAttr(databaseResourceName, "cloud_provider", "AWS"),
		resource.TestCheckResourceAttr(databaseResourceName, "region", "us-east-1"),
		resource.TestCheckResourceAttrSet(databaseResourceName, "redis_version_compliance"),
		resource.TestCheckResourceAttr(databaseResourceName, "resp_version", "resp3"),
		resource.TestCheckResourceAttr(databaseResourceName, "data_persistence", "none"),
		resource.TestCheckResourceAttr(databaseResourceName, "data_eviction", "volatile-lru"),
		resource.TestCheckResourceAttr(databaseResourceName, "replication", "false"),
		resource.TestCheckResourceAttrSet(databaseResourceName, "public_endpoint"),
		resource.TestCheckResourceAttr(databaseResourceName, "private_endpoint", ""),
		resource.TestCheckResourceAttr(databaseResourceName, "source_ips.#", "0"),
		resource.TestCheckResourceAttr(databaseResourceName, "enable_default_user", "true"),
		resource.TestCheckResourceAttr(databaseResourceName, "password", password),
		resource.TestCheckResourceAttr(databaseResourceName, "enable_payg_features", "false"),
		resource.TestCheckResourceAttr(databaseResourceName, "memory_limit_in_gb", "0"),
		resource.TestCheckResourceAttr(databaseResourceName, "support_oss_cluster_api", "false"),
		resource.TestCheckResourceAttr(databaseResourceName, "external_endpoint_for_oss_cluster_api", "false"),
		resource.TestCheckResourceAttr(databaseResourceName, "enable_database_clustering", "false"),
		resource.TestCheckResourceAttr(databaseResourceName, "regex_rules.#", "0"),
		resource.TestCheckResourceAttr(databaseResourceName, "enable_tls", "false"),
	)
}

// resource.Test rather than ParallelTest: Redis Cloud allows only one essentials subscription per account.
func TestAccResourceRedisCloudEssentialsDatabase_CRUDI(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	databaseNameUpdated := databaseName + "-updated"
	password := acctest.RandString(20)
	tags := map[string]string{"environment": "production", "cost_center": "0700", "department": "finance"}
	checkTags := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(databaseResourceName, "tags.%", "3"),
		resource.TestCheckResourceAttr(databaseResourceName, "tags.environment", "production"),
		resource.TestCheckResourceAttr(databaseResourceName, "tags.cost_center", "0700"),
		resource.TestCheckResourceAttr(databaseResourceName, "tags.department", "finance"),
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile(crudiConfig),
				ConfigVariables: crudiVariables(subscriptionName, databaseName, password, tags),
				Check:           resource.ComposeAggregateTestCheckFunc(checkBasicDatabase(databaseName, password), checkTags),
			},
			{
				ConfigFile:      config.StaticFile(crudiConfig),
				ConfigVariables: crudiVariables(subscriptionName, databaseName, password, map[string]string{"UpperCaseKey": "invalid", "environment": "production"}),
				ExpectError:     regexp.MustCompile("tag keys and values must be lower case, invalid entries: UpperCaseKey"),
			},
			{
				ConfigFile:      config.StaticFile(crudiConfig),
				ConfigVariables: crudiVariables(subscriptionName, databaseNameUpdated, password, tags),
				Check:           resource.ComposeAggregateTestCheckFunc(checkBasicDatabase(databaseNameUpdated, password), checkTags),
			},
			{
				ConfigFile:              config.StaticFile(crudiConfig),
				ConfigVariables:         crudiVariables(subscriptionName, databaseNameUpdated, password, tags),
				ResourceName:            databaseResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: databaseImportVerifyIgnore,
			},
		},
	})
}

// Disabling the default user used to fail because the password was sent in the same request.
func TestAccResourceRedisCloudEssentialsDatabase_DisableDefaultUser(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	databaseNameUpdated := databaseName + "-updated"
	password := acctest.RandString(20)

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(defaultUserConfig),
				ConfigVariables: config.Variables{
					"subscription_name":   config.StringVariable(subscriptionName),
					"database_name":       config.StringVariable(databaseName),
					"enable_default_user": config.BoolVariable(true),
					"password":            config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(databaseResourceName, "id", regexp.MustCompile(databaseIDPattern)),
					resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
					resource.TestCheckResourceAttr(databaseResourceName, "enable_default_user", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "password", password),
					resource.TestCheckResourceAttr(databaseResourceName, "tags.envaaaa", "qaaaa"),
				),
			},
			{
				ConfigFile: config.StaticFile(defaultUserConfig),
				ConfigVariables: config.Variables{
					"subscription_name":   config.StringVariable(subscriptionName),
					"database_name":       config.StringVariable(databaseNameUpdated),
					"enable_default_user": config.BoolVariable(false),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "name", databaseNameUpdated),
					resource.TestCheckResourceAttr(databaseResourceName, "enable_default_user", "false"),
					resource.TestCheckResourceAttr(databaseResourceName, "password", ""),
				),
			},
			{
				ConfigFile: config.StaticFile(defaultUserConfig),
				ConfigVariables: config.Variables{
					"subscription_name":   config.StringVariable(subscriptionName),
					"database_name":       config.StringVariable(databaseNameUpdated),
					"enable_default_user": config.BoolVariable(false),
				},
				ResourceName:            databaseResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: databaseImportVerifyIgnore,
			},
		},
	})
}

func redisVersionVariables(subscriptionName, databaseName, password string, extra config.Variables) config.Variables {
	variables := config.Variables{
		"subscription_name": config.StringVariable(subscriptionName),
		"database_name":     config.StringVariable(databaseName),
		"password":          config.StringVariable(password),
	}
	for k, v := range extra {
		variables[k] = v
	}
	return variables
}

// Without redis_version in the configuration, the version chosen by the API is stored.
func TestAccResourceRedisCloudEssentialsDatabase_RedisVersion(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	password := acctest.RandString(20)

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile(redisVersionConfig),
				ConfigVariables: redisVersionVariables(subscriptionName, databaseName, password, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(databaseResourceName, "id", regexp.MustCompile(databaseIDPattern)),
					resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
					resource.TestCheckResourceAttrSet(databaseResourceName, "redis_version"),
					resource.TestCheckResourceAttrSet(databaseResourceName, "redis_version_actual"),
					resource.TestCheckResourceAttrSet(databaseResourceName, "redis_version_compliance"),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsDatabase_RedisVersionUpgrade(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	password := acctest.RandString(20)

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(redisVersionConfig),
				ConfigVariables: redisVersionVariables(subscriptionName, databaseName, password, config.Variables{
					"redis_version": config.StringVariable("8.2"),
					"protocol":      config.StringVariable("stack"),
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(databaseResourceName, "data_persistence", "none"),
				),
			},
			{
				ConfigFile: config.StaticFile(redisVersionConfig),
				ConfigVariables: redisVersionVariables(subscriptionName, databaseName, password, config.Variables{
					"redis_version": config.StringVariable("8.4"),
					"protocol":      config.StringVariable("stack"),
					"alert_value":   config.IntegerVariable(85),
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(databaseResourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttrSet(databaseResourceName, "public_endpoint"),
				),
			},
		},
	})
}

// redis_version is a minimum: lowering it below the running version must not trigger a downgrade or a lasting diff.
// SDKv2 hid this change with a DiffSuppressFunc; the framework applies it once without calling the upgrade endpoint.
func TestAccResourceRedisCloudEssentialsDatabase_RedisVersionAutoMinorUpgrade(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	password := acctest.RandString(20)

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(redisVersionConfig),
				ConfigVariables: redisVersionVariables(subscriptionName, databaseName, password, config.Variables{
					"redis_version": config.StringVariable("8.6"),
					"protocol":      config.StringVariable("stack"),
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version", "8.6"),
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version_actual", "8.6"),
				),
			},
			{
				ConfigFile: config.StaticFile(redisVersionConfig),
				ConfigVariables: redisVersionVariables(subscriptionName, databaseName, password, config.Variables{
					"redis_version": config.StringVariable("8.4"),
					"protocol":      config.StringVariable("stack"),
				}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(databaseResourceName, "redis_version_actual", "8.6"),
					resource.TestCheckResourceAttrSet(databaseResourceName, "public_endpoint"),
				),
			},
		},
	})
}

// State written by the last SDKv2 release must plan cleanly with the framework implementation.
// Step 1 pins that release in its terraform block, since ExternalProviders cannot be combined with ConfigFile.
func TestAccResourceRedisCloudEssentialsDatabase_UpgradeFromSDK(t *testing.T) {
	subscriptionName := utils.RandomWithPrefix()
	databaseName := subscriptionName + "-db"
	password := acctest.RandString(20)
	variables := crudiVariables(subscriptionName, databaseName, password, map[string]string{"environment": "production"})

	resource.Test(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		CheckDestroy: testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/upgrade_from_sdk/sdk_v2_20_0.tf"),
				ConfigVariables: variables,
				Check:           resource.TestCheckResourceAttr(databaseResourceName, "name", databaseName),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/upgrade_from_sdk/registry_source.tf"),
				ConfigVariables:          variables,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkBasicDatabase(databaseName, password),
			},
		},
	})
}
