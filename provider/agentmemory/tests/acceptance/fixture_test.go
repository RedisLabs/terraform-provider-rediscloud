package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	storeResource = "rediscloud_agent_memory.test"
	keyResource   = "rediscloud_agent_memory_api_key.test"
)

type fixture struct {
	name         string
	directory    string
	region       string
	cloudAccount string
	team         string
	owner        string
	cardType     string
	cardLastFour string
	redisVersion config.Variable
	check        func()
}

func newFixture(t *testing.T, storeConfig string) fixture {
	t.Helper()
	checks := []func(*testing.T) bool{envchecks.RedisCloudCheck}
	cloudAccount := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	if cloudAccount != "" {
		checks = append(checks, func(t *testing.T) bool {
			return envchecks.RequireEnvVars(t, "TEAM_RESOURCE_TAG", "OWNER_RESOURCE_TAG")
		})
	}
	// Keep the shared infrastructure and the selected store shape as separate HCL files.
	directory := t.TempDir()
	for _, path := range []string{"testdata/infrastructure.tf", storeConfig} {
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(directory, filepath.Base(path)), content, 0600))
	}
	var redisVersion config.Variable = json.RawMessage("null")
	// Available versions differ between the integration and customer accounts.
	if version := os.Getenv("AGENT_MEMORY_TEST_REDIS_VERSION"); version != "" {
		redisVersion = config.StringVariable(version)
	}
	return fixture{
		name:         utils.RandomWithPrefix() + "-am",
		directory:    directory,
		region:       envDefault("AGENT_MEMORY_TEST_REGION", "us-east-1"),
		cloudAccount: cloudAccount,
		team:         os.Getenv("TEAM_RESOURCE_TAG"),
		owner:        os.Getenv("OWNER_RESOURCE_TAG"),
		cardType:     envDefault("AGENT_MEMORY_TEST_CARD_TYPE", "Visa"),
		cardLastFour: envDefault("AGENT_MEMORY_TEST_CARD_LAST_FOUR", "5556"),
		redisVersion: redisVersion,
		check:        envchecks.ComposePreChecks(t, checks...),
	}
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
