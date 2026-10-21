package acceptance

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	storeResource = "rediscloud_agent_memory.test"
	keyResource   = "rediscloud_agent_memory_api_key.test"
)

type fixture struct {
	name   string
	config string
	vars   config.Variables
	check  func()
}

func newFixture(t *testing.T, storeConfig string) fixture {
	t.Helper()
	name := utils.RandomWithPrefix() + "-am"
	checks := []func(*testing.T) bool{envchecks.RedisCloudCheck}
	cloudAccount := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	if cloudAccount != "" {
		checks = append(checks, func(t *testing.T) bool {
			return envchecks.RequireEnvVars(t, "TEAM_RESOURCE_TAG", "OWNER_RESOURCE_TAG")
		})
	}
	return fixture{
		name:   name,
		config: utils.GetTestConfig(t, "testdata/infrastructure.tf") + "\n" + utils.GetTestConfig(t, storeConfig),
		vars: config.Variables{
			"name":               config.StringVariable(name),
			"region":             config.StringVariable(envDefault("AGENT_MEMORY_TEST_REGION", "us-east-1")),
			"cloud_account_name": config.StringVariable(cloudAccount),
			"team":               config.StringVariable(os.Getenv("TEAM_RESOURCE_TAG")),
			"owner":              config.StringVariable(os.Getenv("OWNER_RESOURCE_TAG")),
			"card_type":          config.StringVariable(envDefault("AGENT_MEMORY_TEST_CARD_TYPE", "Visa")),
			"card_last_four":     config.StringVariable(envDefault("AGENT_MEMORY_TEST_CARD_LAST_FOUR", "5556")),
		},
		check: envchecks.ComposePreChecks(t, checks...),
	}
}

func (f fixture) step(vars config.Variables, checks ...resource.TestCheckFunc) resource.TestStep {
	values := make(config.Variables, len(f.vars)+len(vars))
	for name, value := range f.vars {
		values[name] = value
	}
	for name, value := range vars {
		values[name] = value
	}
	return resource.TestStep{Config: f.config, ConfigVariables: values, Check: resource.ComposeAggregateTestCheckFunc(checks...)}
}

func (f fixture) importStep(vars config.Variables, resourceName string) resource.TestStep {
	// Import steps do not reuse the previous step's variable file.
	step := f.step(vars)
	step.Check = nil
	step.ResourceName = resourceName
	step.ImportState, step.ImportStateVerify = true, true
	return step
}

func (f fixture) testCase(owned *trackedResources, steps []resource.TestStep) resource.TestCase {
	return resource.TestCase{
		PreCheck:                 f.check,
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy: resource.ComposeTestCheckFunc(
			owned.checkDestroyed,
			testhelpers.CheckProSubscriptionDestroy,
		),
		Steps: steps,
	}
}

func envDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
