package activeactive_test

// This file exercises redis_version and redis_version_actual through Terraform's whole plan and apply
// pipeline, without provisioning any real infrastructure.
//
// The tests drive the real resource, activeactive.NewActiveActiveDatabaseResource, against the stateful
// in-memory API (fake_aa_api_test.go). Production createDatabase, readDatabase and updateDatabase all
// execute, along with the schema's plan modifiers and its nil-state guards. Because resource.UnitTest runs
// real Terraform, these tests also cover plan idempotency and plan/apply consistency.

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const aaResourceName = "test_active_active_subscription_database.aa_db"

type planCheckFunc func(context.Context, plancheck.CheckPlanRequest, *plancheck.CheckPlanResponse)

func (f planCheckFunc) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	f(ctx, req, resp)
}

// newAAFixture returns the Active-Active fixture and the provider factories wired to it
func newAAFixture(t *testing.T) (*aaAPI, map[string]func() (tfprotov5.ProviderServer, error)) {
	t.Helper()

	api := newAAAPI()

	return api, testhelpers.FrameworkProviderFactories(
		testhelpers.NewAPIClient(t, api.fake),
		activeactive.NewActiveActiveDatabaseResource,
	)
}

// The configs these tests apply live in testdata/unit, which keeps them apart from the acceptance
// configs in testdata. Each one declares the variables every step that uses it has to supply.
const (
	aaDatabaseConfig           = "./testdata/unit/database.tf"
	aaDatabaseNoVersionConfig  = "./testdata/unit/database_no_version.tf"
	aaDatabaseNoPasswordConfig = "./testdata/unit/database_no_password.tf"
)

func TestUnitActiveActiveDatabaseVersion_DefaultsWhenOmitted(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseNoVersionConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", defaultRedisVersion),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", defaultRedisVersion),
				),
			},
			{
				ConfigFile: config.StaticFile(aaDatabaseNoVersionConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	// The database creation step goes to the create API and never to the upgrade endpoint.
	// The second step is a no-op, so nothing here should have asked for an upgrade.
	assert.Empty(t, api.getRequestedUpgrades())
}

func TestUnitActiveActiveDatabaseVersion_SatisfiedRequestIsNoOp(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.4"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.4"),
				),
			},
			{
				// 8.2 is lower than the running 8.4, so the request is already satisfied.
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.2"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					// redis_version is Optional+Computed and set in config, so the config value lands in state.
					// This is a real update rather than a suppressed one.
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// State records what the user asked for. redis_version_actual still reports what the
					// database is actually running.
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.4"),
				),
			},
		},
	})

	// BUG: the state assertions above pass, but on the way there the provider asked the API to change the
	// running version from 8.4 down to 8.2, which is a downgrade.
	//
	// The version guard in updateDatabase compares the planned redis_version against both redis_version and
	// redis_version_actual in state, and it compares them for string inequality. A target below the running
	// version is unequal to both, so it passes the guard exactly as an upgrade would. Nothing on that path
	// compares the two versions to work out which one is higher.
	//
	// redis_version_actual still reads 8.4 afterwards only because this fixture ignores an upgrade request
	// whose target is not higher than the running version, which is what isUpgrade decides. What the real API
	// does with a downgrade target is unknown, so this test does not show that production state stays correct.
	// It shows only that the provider sends the request.
	//
	// Once the guard is fixed, change this to assert an empty slice.
	assert.Equal(t, []string{"8.2"}, api.getRequestedUpgrades())
}

// A genuine upgrade must allow redis_version_actual to be recomputed during apply and then converge.
func TestUnitActiveActiveDatabaseVersion_GenuineUpgrade(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.2"),
				},
				Check: resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
			},
			{
				// 8.4 is higher than the running 8.2, so this is a genuine upgrade.
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.4"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.4"),
				),
			},
		},
	})

	// A genuine upgrade reaches the endpoint exactly once with the configured target.
	assert.Equal(t, []string{"8.4"}, api.getRequestedUpgrades())
}

// A background auto-upgrade moves the running version past the requested one. A later config change to a
// version that is still lower than the running one must be a no-op, never an in-place downgrade.
func TestUnitActiveActiveDatabaseVersion_BackgroundUpgradeThenLowerRequestIsNoOp(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.2"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
				),
			},
			{
				PreConfig: func() {
					api.setRunningVersion("8.6") // simulate a background auto minor upgrade
				},
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.4"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					// The config change from 8.2 to 8.4 is a real update. redis_version is Optional+Computed and
					// set, so state follows config even though the running 8.6 already satisfies it.
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
					},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// State records the requested 8.4, and redis_version_actual keeps the 8.6 that the background
					// upgrade moved it to. The database is not moved back down to 8.4, which is the whole point.
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.6"),
				),
			},
		},
	})

	// BUG: the same guard as in SatisfiedRequestIsNoOp, in the scenario this resource most needs to get right.
	// A background auto minor upgrade took the database to 8.6, the customer then asked for 8.4, and the
	// provider asked the API to change the running version from 8.6 down to 8.4, which is a downgrade.
	//
	// redis_version_actual still reads 8.6 afterwards only because this fixture ignores an upgrade request
	// whose target is not higher than the running version, which is what isUpgrade decides. A real API that
	// honoured the request would take a customer's database backwards.
	//
	// Once the guard is fixed, change this to assert an empty slice.
	assert.Equal(t, []string{"8.4"}, api.getRequestedUpgrades())
}

// Changing an attribute other than redis_version must not touch the version endpoint. What prevents it is
// the first half of the guard, which requires the planned redis_version to differ from prior state. This is
// also the only test here that exercises an update unrelated to versions.
func TestUnitActiveActiveDatabaseVersion_UnrelatedChangeDoesNotUpgrade(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.2"),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
				),
			},
			{
				// A rename only. redis_version is untouched.
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db-renamed"),
					"redis_version":   config.StringVariable("8.2"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
						// The configured target remains known, while the computed running version is
						// refreshed during any update.
						plancheck.ExpectKnownValue(aaResourceName, tfjsonpath.New("redis_version"), knownvalue.StringExact("8.2")),
						plancheck.ExpectUnknownValue(aaResourceName, tfjsonpath.New("redis_version_actual")),
					},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "name", "aa-db-renamed"),
					// Both version attributes stay put.
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
				),
			},
		},
	})

	assert.Empty(t, api.getRequestedUpgrades())
}

// A background upgrade that lands after planning must be accepted when apply refreshes the computed
// running version.
func TestUnitActiveActiveDatabaseVersion_BackgroundUpgradeAfterPlanConverges(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.2"),
				},
				Check: resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
			},
			{
				ConfigFile: config.StaticFile(aaDatabaseConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db-renamed"),
					"redis_version":   config.StringVariable("8.2"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
						planCheckFunc(func(_ context.Context, _ plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
							api.setRunningVersion("8.4")
						}),
					},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.4"),
				),
			},
		},
	})

	assert.Empty(t, api.getRequestedUpgrades())
}

// The note on global_password.
//
// In the default configuration, where config leaves global_password null and the database is not
// passwordless, any plan that changes another attribute also plans global_password as unknown. The test
// below pins that behaviour.
//
// Whether it is a defect is unclear, so this records it rather than judging it. It is arguably correct,
// because the attribute is Optional+Computed and null in config, so Terraform cannot know what the API will
// return and marking the value unknown is conservative. It is arguably broken, because ModifyPlan preserves
// the prior value specifically to avoid this and does not succeed.
//
// The behaviour does settle, which the test asserts. The plan straight after an apply is clean, so this is
// neither drift nor a perpetual diff.
//
// The attribute deliberately has no attribute-level plan modifier. UseStateForUnknown caused
// "inconsistent values for sensitive attribute" errors on passwordless transitions, so ModifyPlan preserves
// the value by hand instead. Marking the attribute Sensitive is not the cause of any of this, since removing
// that changes nothing.
//
// The provider was instrumented and traced. It returns a known value while Terraform records the attribute
// as unknown, so the gap appears to sit below the provider, but that investigation reached no verdict.
// Chasing it further needs a minimal reproduction outside this repo rather than a third workaround in the
// resource.
//
// The practical consequence is that a test cannot assert a genuinely empty plan for a no-op config change
// unless global_password is set in config, which is why every config in testdata/unit except
// database_no_password.tf sets it.
//
// None of this is verified against the real API. If that API returns the password only on create, then
// readDatabase would write null to state on refresh, which would be a separate and worse problem that this
// fixture cannot reproduce.

// This test pins the global_password behaviour described in the note above. The change it makes is a rename,
// which has nothing to do with the password, and yet the plan shows global_password as unknown. It then
// asserts that the behaviour settles, because the plan straight after the apply is clean.
func TestUnitActiveActiveDatabase_GlobalPasswordLeftToAPIIsPlannedUnknown(t *testing.T) {
	api, factories := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile(aaDatabaseNoPasswordConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db"),
					"redis_version":   config.StringVariable("8.2"),
				},
				// The generated password is persisted, so a missing state value is not the cause.
				Check: resource.TestCheckResourceAttrSet(aaResourceName, "global_password"),
			},
			{
				ConfigFile: config.StaticFile(aaDatabaseNoPasswordConfig),
				ConfigVariables: config.Variables{
					"subscription_id": config.IntegerVariable(api.subID),
					"name":            config.StringVariable("aa-db-renamed"),
					"redis_version":   config.StringVariable("8.2"),
				},
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
						// This is the questionable part. Nothing in config or state justifies the value being
						// unknown.
						plancheck.ExpectUnknownValue(aaResourceName, tfjsonpath.New("global_password")),
					},
					// The behaviour settles. Once there is nothing else to change, the plan is clean.
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "name", "aa-db-renamed"),
					resource.TestCheckResourceAttrSet(aaResourceName, "global_password"),
				),
			},
		},
	})
}
