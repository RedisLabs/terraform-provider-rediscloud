package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccResourceRedisCloudAgentMemory_CustomerModels(t *testing.T) {
	llmKey, embeddingKey := os.Getenv("AGENT_MEMORY_LLM_API_KEY"), os.Getenv("AGENT_MEMORY_EMBEDDING_API_KEY")
	if llmKey == "" || embeddingKey == "" {
		t.Skip("Customer-managed model tests require AGENT_MEMORY_LLM_API_KEY and AGENT_MEMORY_EMBEDDING_API_KEY")
	}
	t.Setenv("TF_VAR_llm_api_key", llmKey)
	t.Setenv("TF_VAR_embedding_api_key", embeddingKey)
	f := newFixture(t, "testdata/store_byok.tf")
	tracked := &trackedResources{}
	keys := modelKeys{llmKey, embeddingKey}
	resource.Test(t, f.testCase(tracked, []resource.TestStep{
		{
			ConfigDirectory: config.StaticDirectory(f.directory),
			ConfigVariables: config.Variables{
				"name":               config.StringVariable(f.name),
				"region":             config.StringVariable(f.region),
				"cloud_account_name": config.StringVariable(f.cloudAccount),
				"team":               config.StringVariable(f.team),
				"owner":              config.StringVariable(f.owner),
				"card_type":          config.StringVariable(f.cardType),
				"card_last_four":     config.StringVariable(f.cardLastFour),
				"redis_version":      f.redisVersion,
				"llm_model":          config.StringVariable(envDefault("AGENT_MEMORY_MODEL", "gpt-4o-mini")),
				"embedding_model":    config.StringVariable(envDefault("AGENT_MEMORY_EMBEDDING_MODEL", "text-embedding-3-small")),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources, keys.checkState),
		},
		// Submit the same valid keys with a new credential version.
		{
			ConfigDirectory: config.StaticDirectory(f.directory),
			ConfigVariables: config.Variables{
				"name":               config.StringVariable(f.name),
				"region":             config.StringVariable(f.region),
				"cloud_account_name": config.StringVariable(f.cloudAccount),
				"team":               config.StringVariable(f.team),
				"owner":              config.StringVariable(f.owner),
				"card_type":          config.StringVariable(f.cardType),
				"card_last_four":     config.StringVariable(f.cardLastFour),
				"redis_version":      f.redisVersion,
				"llm_model":          config.StringVariable(envDefault("AGENT_MEMORY_MODEL", "gpt-4o-mini")),
				"embedding_model":    config.StringVariable(envDefault("AGENT_MEMORY_EMBEDDING_MODEL", "text-embedding-3-small")),
				"key_version":        config.IntegerVariable(2),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources, keys.checkState),
		},
		{
			ConfigDirectory: config.StaticDirectory(f.directory),
			ConfigVariables: config.Variables{
				"name":               config.StringVariable(f.name),
				"region":             config.StringVariable(f.region),
				"cloud_account_name": config.StringVariable(f.cloudAccount),
				"team":               config.StringVariable(f.team),
				"owner":              config.StringVariable(f.owner),
				"card_type":          config.StringVariable(f.cardType),
				"card_last_four":     config.StringVariable(f.cardLastFour),
				"redis_version":      f.redisVersion,
				"llm_model":          config.StringVariable(envDefault("AGENT_MEMORY_MODEL", "gpt-4o-mini")),
				"embedding_model":    config.StringVariable(envDefault("AGENT_MEMORY_EMBEDDING_MODEL", "text-embedding-3-small")),
				"key_version":        config.IntegerVariable(2),
			},
			PlanOnly:         true,
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{keys}},
		},
	}))
}

type modelKeys []string

func (keys modelKeys) checkState(state *terraform.State) error {
	return keys.absentFrom(state)
}

func (keys modelKeys) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	resp.Error = keys.absentFrom(req.Plan)
}

func (keys modelKeys) absentFrom(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if strings.Contains(string(data), key) {
			return errors.New("write-only model credential was persisted in Terraform plan or state")
		}
	}
	return nil
}
