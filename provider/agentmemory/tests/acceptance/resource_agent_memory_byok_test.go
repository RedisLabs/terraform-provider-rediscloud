package acceptance

import (
	"errors"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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
	values := config.Variables{
		"llm_model":       config.StringVariable(envDefault("AGENT_MEMORY_MODEL", "gpt-4o-mini")),
		"embedding_model": config.StringVariable(envDefault("AGENT_MEMORY_EMBEDDING_MODEL", "text-embedding-3-small")),
	}
	keys := modelKeys{llmKey, embeddingKey}
	create := f.step(values, tracked.remember(false, false), checkRemoteResources, keys.checkState)
	create.ConfigStateChecks = modelCredentialSensitivity()
	// Incrementing the version exercises credential submission with the same valid keys.
	// Distinct replacement credentials are not required to run this test.
	values["key_version"] = config.IntegerVariable(2)
	update := f.step(values, tracked.remember(false, false), checkRemoteResources, keys.checkState)
	update.ConfigStateChecks = modelCredentialSensitivity()
	noop := f.step(values)
	noop.PlanOnly = true
	noop.ConfigPlanChecks = resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{
		plancheck.ExpectSensitiveValue(storeResource, tfjsonpath.New("llm").AtMapKey("credentials").AtMapKey("api_key")),
		plancheck.ExpectSensitiveValue(storeResource, tfjsonpath.New("embedding").AtMapKey("credentials").AtMapKey("api_key")),
	}}
	resource.Test(t, f.testCase(tracked, []resource.TestStep{create, update, noop}))
}

type modelKeys []string

func (keys modelKeys) checkState(state *terraform.State) error {
	store := state.RootModule().Resources[storeResource]
	if store == nil || store.Primary == nil || store.Primary.Attributes["llm.credentials.api_key"] != keys[0] || store.Primary.Attributes["embedding.credentials.api_key"] != keys[1] {
		return errors.New("model credentials were not preserved in state")
	}
	return nil
}

func modelCredentialSensitivity() []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectSensitiveValue(storeResource, tfjsonpath.New("llm").AtMapKey("credentials").AtMapKey("api_key")),
		statecheck.ExpectSensitiveValue(storeResource, tfjsonpath.New("embedding").AtMapKey("credentials").AtMapKey("api_key")),
	}
}
