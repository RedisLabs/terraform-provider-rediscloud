package acceptance

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAccResourceRedisCloudAgentMemory_CRUDI(t *testing.T) {
	f := newFixture(t, "testdata/store.tf")
	tracked := &trackedResources{}
	initial := config.Variables{}
	updated := config.Variables{"store_suffix": config.StringVariable("updated"), "short_ttl": config.IntegerVariable(172800)}
	rotated := config.Variables{"store_suffix": config.StringVariable("updated"), "short_ttl": config.IntegerVariable(172800), "key_suffix": config.StringVariable("rotated")}
	noop := f.step(updated)
	noop.PlanOnly = true
	invalid := f.step(config.Variables{"store_suffix": config.StringVariable("updated"), "short_ttl": config.IntegerVariable(0)})
	invalid.Check = nil
	// Some validation responses omit the field name; verify HTTP 400 and read back the TTL separately.
	invalid.ExpectError = regexp.MustCompile(`(?s)Failed to update Agent Memory service.*: 400 -`)
	afterRejection := f.step(updated, tracked.remember(false, false), checkRemoteResources)
	afterRejection.PreConfig = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		store, err := client.MustTestClient(t).Client.AgentMemory.Get(ctx, tracked.storeID)
		if err != nil {
			t.Fatalf("read store after rejected update: %s", err)
		}
		if store.ShortMemory == nil || redis.IntValue(store.ShortMemory.TTLSeconds) != 172800 {
			t.Fatal("rejected update changed the remote short-term TTL")
		}
	}
	rotatePlan := f.step(rotated)
	rotatePlan.PlanOnly, rotatePlan.ExpectNonEmptyPlan = true, true
	rotatePlan.ConfigPlanChecks = resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(keyResource, plancheck.ResourceActionDestroyBeforeCreate),
	}}
	keyImport := f.importStep(updated, keyResource)
	keyImport.ImportStateVerifyIgnore = []string{"api_key"}
	keyImport.ImportStateIdFunc = keyImportID
	keyImport.ImportStateCheck = importedKeyHasNoSecret
	resource.Test(t, f.testCase(tracked, []resource.TestStep{
		f.step(initial, tracked.remember(false, false), checkRemoteResources),
		f.step(updated, tracked.remember(false, false), checkRemoteResources),
		noop,
		invalid,
		afterRejection,
		f.importStep(updated, storeResource),
		keyImport,
		f.step(updated, tracked.remember(false, false), checkRemoteResources),
		rotatePlan,
		f.step(rotated, tracked.remember(false, true), checkRemoteResources, tracked.checkSuperseded),
		{
			Config: f.config, ConfigVariables: f.step(rotated).ConfigVariables,
			PreConfig: func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				// Keep the cleanup rehearsal isolated from concurrent fixtures using the default prefix.
				t.Setenv("TEST_RESOURCE_PREFIX", strings.TrimSuffix(f.name, "-am"))
				t.Setenv("SWEEP_AGE_THRESHOLD", "0s")
				if err := sweepAgentMemory(""); err != nil {
					t.Fatalf("sweep interrupted-run fixture: %s", err)
				}
				if err := storeAbsent(ctx, client.MustTestClient(t), tracked.storeID); err != nil {
					t.Fatalf("fixture was not removed by the sweeper: %s", err)
				}
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(true, true), checkRemoteResources, tracked.checkSuperseded),
		},
	}))
}

func TestAccResourceRedisCloudAgentMemory_Replacement(t *testing.T) {
	f := newFixture(t, "testdata/store.tf")
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
