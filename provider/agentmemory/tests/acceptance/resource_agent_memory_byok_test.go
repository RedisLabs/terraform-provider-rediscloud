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
	f := newFixture(t)
	tracked := &trackedResources{}
	values := config.Variables{
		"customer_models": config.BoolVariable(true),
		"llm_model":       config.StringVariable(envDefault("AGENT_MEMORY_MODEL", "gpt-4o-mini")),
		"embedding_model": config.StringVariable(envDefault("AGENT_MEMORY_EMBEDDING_MODEL", "text-embedding-3-small")),
	}
	keys := modelKeys{llmKey, embeddingKey}
	create := f.step(values, tracked.remember(false, false), checkRemoteResources, keys.checkState)
	// Incrementing the version exercises credential submission with the same valid keys.
	// Distinct replacement credentials are not required to run this test.
	values["key_version"] = config.IntegerVariable(2)
	update := f.step(values, tracked.remember(false, false), checkRemoteResources, keys.checkState)
	noop := f.step(values)
	noop.PlanOnly = true
	noop.ConfigPlanChecks = resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{keys}}
	resource.Test(t, f.testCase(tracked, []resource.TestStep{create, update, noop}))
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
