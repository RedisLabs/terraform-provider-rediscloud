package acceptance

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
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
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources),
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources),
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
			},
			PlanOnly: true,
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(0),
			},
			// Some validation responses omit the field name; check the status and read back the TTL.
			ExpectError: regexp.MustCompile(`(?s)Failed to update Agent Memory service.*: 400 -`),
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
			},
			PreConfig: func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				store, err := client.MustTestClient(t).Client.AgentMemory.Get(ctx, tracked.storeID)
				if err != nil {
					t.Fatalf("read store after rejected update: %s", err)
				}
				if store.ShortMemory == nil || redis.IntValue(store.ShortMemory.TTLSeconds) != 172800 {
					t.Fatal("rejected update changed the remote short-term TTL")
				}
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources),
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
			},
			ResourceName: storeResource, ImportState: true, ImportStateVerify: true,
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
			},
			ResourceName: keyResource, ImportState: true, ImportStateVerify: true,
			ImportStateVerifyIgnore: []string{"api_key"},
			ImportStateIdFunc:       keyImportID,
			ImportStateCheck:        importedKeyHasNoSecret,
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources),
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
				"key_suffix":         config.StringVariable("rotated"),
			},
			PlanOnly: true, ExpectNonEmptyPlan: true,
			ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(keyResource, plancheck.ResourceActionDestroyBeforeCreate),
			}},
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
				"key_suffix":         config.StringVariable("rotated"),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, true), checkRemoteResources, tracked.checkSuperseded),
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
				"store_suffix":       config.StringVariable("updated"),
				"short_ttl":          config.IntegerVariable(172800),
				"key_suffix":         config.StringVariable("rotated"),
			},
			PreConfig: func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				// Keep the cleanup rehearsal isolated from concurrent fixtures using the default prefix.
				t.Setenv("TEST_RESOURCE_PREFIX", strings.TrimSuffix(f.name, "-am"))
				t.Setenv("SWEEP_AGE_THRESHOLD", "0s")
				// The shared sweeper lives in another test package and must run through TestMain.
				//lintignore:XR008
				cmd := exec.CommandContext(ctx, "go", "test", "./provider", "-run=^$", "-sweep=ALL", "-sweep-run=rediscloud_agent_memory", "-timeout=15m")
				cmd.Dir = filepath.Join("..", "..", "..", "..")
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("sweep interrupted-run fixture: %v\n%s", err, output)
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
				"database_count":     config.IntegerVariable(2),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(false, false), checkRemoteResources),
		},
		replacementPlan(resource.TestStep{
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
				"database_count":     config.IntegerVariable(2),
				"database_index":     config.IntegerVariable(1),
			},
		}),
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
				"database_count":     config.IntegerVariable(2),
				"database_index":     config.IntegerVariable(1),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(true, true), checkRemoteResources, tracked.checkSuperseded),
		},
		replacementPlan(resource.TestStep{
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
				"database_count":     config.IntegerVariable(2),
				"database_index":     config.IntegerVariable(1),
				"field_type":         config.StringVariable("bool"),
			},
		}),
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
				"database_count":     config.IntegerVariable(2),
				"database_index":     config.IntegerVariable(1),
				"field_type":         config.StringVariable("bool"),
			},
			Check: resource.ComposeAggregateTestCheckFunc(tracked.remember(true, true), checkRemoteResources, tracked.checkSuperseded),
		},
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
