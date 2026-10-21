package acceptance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAccResourceRedisCloudAgentMemory_CRUDI(t *testing.T) {
	f := newFixture(t)
	tracked := &trackedResources{}
	initial := config.Variables{}
	updated := config.Variables{"store_suffix": config.StringVariable("updated"), "short_ttl": config.IntegerVariable(172800)}
	rotated := config.Variables{"store_suffix": config.StringVariable("updated"), "short_ttl": config.IntegerVariable(172800), "key_suffix": config.StringVariable("rotated")}
	noop := f.step(updated)
	noop.PlanOnly = true
	rotatePlan := f.step(rotated)
	rotatePlan.PlanOnly, rotatePlan.ExpectNonEmptyPlan = true, true
	rotatePlan.ConfigPlanChecks = resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(keyResource, plancheck.ResourceActionDestroyBeforeCreate),
	}}
	resource.Test(t, f.testCase(tracked, []resource.TestStep{
		f.step(initial, tracked.remember(false, false), checkRemoteResources),
		f.step(updated, tracked.remember(false, false), checkRemoteResources),
		noop,
		{ResourceName: storeResource, ImportState: true, ImportStateVerify: true},
		{ResourceName: keyResource, ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"api_key"}, ImportStateIdFunc: keyImportID, ImportStateCheck: importedKeyHasNoSecret},
		f.step(updated, tracked.remember(false, false), checkRemoteResources),
		rotatePlan,
		f.step(rotated, tracked.remember(false, true), checkRemoteResources, tracked.checkSuperseded),
		{
			Config: f.config, ConfigVariables: f.step(rotated).ConfigVariables,
			PreConfig: func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				if err := client.MustTestClient(t).Client.AgentMemory.Delete(ctx, tracked.storeID); err != nil {
					t.Fatalf("delete test store outside Terraform: %s", err)
				}
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(true, true), checkRemoteResources, tracked.checkSuperseded),
		},
	}))
}

func TestAccResourceRedisCloudAgentMemory_Replacement(t *testing.T) {
	f := newFixture(t)
	tracked := &trackedResources{}
	initial := config.Variables{"database_count": config.IntegerVariable(2)}
	moved := config.Variables{"database_count": config.IntegerVariable(2), "database_index": config.IntegerVariable(1)}
	redefined := config.Variables{"database_count": config.IntegerVariable(2), "database_index": config.IntegerVariable(1), "field_type": config.StringVariable("bool")}
	resource.Test(t, f.testCase(tracked, []resource.TestStep{
		f.step(initial, tracked.remember(false, false), checkRemoteResources),
		replacementPlan(f.step(moved)),
		f.step(moved, tracked.remember(true, true), checkRemoteResources, tracked.checkSuperseded),
		replacementPlan(f.step(redefined)),
		f.step(redefined, tracked.remember(true, true), checkRemoteResources, tracked.checkSuperseded),
	}))
}

func replacementPlan(step resource.TestStep) resource.TestStep {
	step.PlanOnly, step.ExpectNonEmptyPlan = true, true
	step.ConfigPlanChecks = resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(storeResource, plancheck.ResourceActionDestroyBeforeCreate),
		plancheck.ExpectResourceAction(keyResource, plancheck.ResourceActionDestroyBeforeCreate),
	}}
	return step
}

func keyImportID(state *terraform.State) (string, error) {
	key := state.RootModule().Resources[keyResource]
	if key == nil || key.Primary == nil || key.Primary.Attributes["store_id"] == "" || key.Primary.ID == "" {
		return "", fmt.Errorf("API key identity is missing from state")
	}
	return key.Primary.Attributes["store_id"] + "/" + key.Primary.ID, nil
}

func importedKeyHasNoSecret(states []*terraform.InstanceState) error {
	if len(states) != 1 || states[0] == nil || states[0].Attributes["api_key"] != "" {
		return fmt.Errorf("imported API key should have metadata but no recoverable secret")
	}
	return nil
}
