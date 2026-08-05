package activeactive_test

// This file exercises redis_version / redis_version_actual end-to-end through Terraform's plan/apply
// pipeline, WITHOUT provisioning any real infrastructure.
//
// It drives the REAL resource — activeactive.NewActiveActiveDatabaseResource — against the stateful
// in-memory API in ./mock_aa_api_test.go, so production createDatabase/readDatabase/updateDatabase,
// the schema's plan modifiers and the nil-state guards all execute. resource.UnitTest runs real
// Terraform against it, catching plan idempotency and plan/apply consistency behaviour that direct
// PlanModifyString calls cannot.

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const aaResourceName = "test_active_active_subscription_database.aa_db"

type planCheckFunc func(context.Context, plancheck.CheckPlanRequest, *plancheck.CheckPlanResponse)

func (f planCheckFunc) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	f(ctx, req, resp)
}

// newAAFixture returns a fixture with one subscription registered, the provider factories wired to it,
// and that subscription's id for use in configs.
func newAAFixture(t *testing.T) (*mockAAAPI, map[string]func() (tfprotov5.ProviderServer, error), int) {
	t.Helper()

	api := newMockAAAPI()
	subID := api.addSubscription()

	apiClient, err := testhelpers.MockAPIClient(api.buildMock())
	require.NoError(t, err)

	return api, testhelpers.FrameworkProviderFactoriesWithData(
		apiClient,
		activeactive.NewActiveActiveDatabaseResource,
	), subID
}

// Configs set global_password even though no test asserts anything about it — see
// BEHAVIOUR(global_password) at the bottom of this file. Without it, every step that plans an update
// also plans that attribute as unknown, which defeats any empty-plan assertion this file wants to make.
func newAADatabaseConfig(subID int, redisVersion string) string {
	return fmt.Sprintf(`resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = %d
  name               = "aa-db"
  dataset_size_in_gb = 1
  global_password    = "test-password"
  redis_version      = %q
}`, subID, redisVersion)
}

func newAADatabaseConfigNoVersion(subID int) string {
	return fmt.Sprintf(`resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = %d
  name               = "aa-db"
  dataset_size_in_gb = 1
  global_password    = "test-password"
}`, subID)
}

// newAADatabaseConfigNoPassword leaves global_password out — the default configuration, where the API
// generates it. Only the test exercising BEHAVIOUR(global_password) uses this; every other config sets
// it, for the reason described there.
func newAADatabaseConfigNoPassword(subID int, name string) string {
	return fmt.Sprintf(`resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = %d
  name               = %q
  dataset_size_in_gb = 1
  redis_version      = "8.2"
}`, subID, name)
}

func newAADatabaseConfigNamed(subID int, name, redisVersion string) string {
	return fmt.Sprintf(`resource "test_active_active_subscription_database" "aa_db" {
  subscription_id    = %d
  name               = %q
  dataset_size_in_gb = 1
  global_password    = "test-password"
  redis_version      = %q
}`, subID, name, redisVersion)
}

func TestUnitActiveActiveDatabaseVersion_DefaultsWhenOmitted(t *testing.T) {
	api, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfigNoVersion(subID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", defaultRedisVersion),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", defaultRedisVersion),
				),
			},
			{
				Config: newAADatabaseConfigNoVersion(subID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})

	// The version a database is created at goes to the create API, never to the upgrade endpoint, and
	// the second step is a no-op — so nothing here should reach UpgradeRedisVersion.
	assert.Empty(t, api.requestedUpgrades())
}

func TestUnitActiveActiveDatabaseVersion_SatisfiedRequestIsNoOp(t *testing.T) {
	api, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfig(subID, "8.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.4"),
				),
			},
			{
				Config: newAADatabaseConfig(subID, "8.2"), // lower than the running 8.4 -> satisfied
				ConfigPlanChecks: resource.ConfigPlanChecks{
					// RedisVersion keeps the prior value when the running version already satisfies the
					// request, so lowering redis_version produces no diff at all.
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// redis_version keeps the prior 8.4 rather than following config down to 8.2:
					// redis_version is a floor, and the running version already clears it.
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.4"),
				),
			},
		},
	})

	// No upgrade call is issued, and not because the API refused a downgrade — the provider never asks.
	// RedisVersion suppresses the diff at plan time, so updateDatabase's version branch is never reached
	// and its exact-string comparison never gets the chance to misfire.
	assert.Empty(t, api.requestedUpgrades())
}

// A genuine upgrade applies cleanly. RedisVersionActual leaves redis_version_actual unknown when the
// running version does not satisfy the requested one, so apply is free to write the new running
// version. With plain UseStateForUnknown the planned value was pinned to the prior version and apply
// contradicted it — "Provider produced inconsistent result after apply".
func TestUnitActiveActiveDatabaseVersion_GenuineUpgrade(t *testing.T) {
	api, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfig(subID, "8.2"),
				Check:  resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
			},
			{
				Config: newAADatabaseConfig(subID, "8.4"), // higher than the running 8.2 -> genuine upgrade
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

	// The one path that SHOULD reach the upgrade endpoint: exactly one request, for the version config
	// asked for — not skipped, not repeated.
	assert.Equal(t, []string{"8.4"}, api.requestedUpgrades())
}

// A background auto-upgrade moves the running version past the requested one; a subsequent config
// change to a still-lower version is a no-op, and no in-place downgrade is even attempted.
func TestUnitActiveActiveDatabaseVersion_BackgroundUpgradeThenLowerRequestIsNoOp(t *testing.T) {
	api, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfig(subID, "8.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
				),
			},
			{
				PreConfig: func() {
					api.setRunningVersion("8.6") // simulate a background auto minor upgrade
				},
				Config: newAADatabaseConfig(subID, "8.4"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					// Running 8.6 satisfies the requested 8.4, so RedisVersion suppresses the diff entirely.
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					// redis_version keeps the prior 8.2 — not the 8.4 in config — because the running 8.6
					// already satisfies the request. redis_version_actual tracks the background upgrade.
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.6"),
				),
			},
		},
	})

	// The scenario this resource most needs to get right: a background auto minor upgrade took the
	// database to 8.6, and the provider does NOT respond by requesting 8.4. No in-place downgrade is even
	// attempted, and that no longer depends on the API refusing one.
	assert.Empty(t, api.requestedUpgrades())
}

// Changing an attribute other than redis_version must not touch the version endpoint. The guard's
// first half (planned redis_version differs from prior state) is what prevents it, and this is the
// only test that exercises a non-version update at all.
func TestUnitActiveActiveDatabaseVersion_UnrelatedChangeDoesNotUpgrade(t *testing.T) {
	api, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfigNamed(subID, "aa-db", "8.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(aaResourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
				),
			},
			{
				// Rename only — redis_version is untouched.
				Config: newAADatabaseConfigNamed(subID, "aa-db-renamed", "8.2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
						// Asserted on the PLAN, not just on state after apply: the point is that neither
						// version attribute goes unknown ("known after apply") on an update that has nothing
						// to do with versions. A post-apply state check cannot see that — the state value is
						// 8.2 either way — so without these two checks, removing the plan modifier from
						// redis_version_actual would only be caught by GenuineUpgrade.
						plancheck.ExpectKnownValue(aaResourceName, tfjsonpath.New("redis_version"), knownvalue.StringExact("8.2")),
						plancheck.ExpectKnownValue(aaResourceName, tfjsonpath.New("redis_version_actual"), knownvalue.StringExact("8.2")),
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

	assert.Empty(t, api.requestedUpgrades())
}

// A background auto-upgrade can land after Terraform has saved the plan but before apply. Since
// redis_version_actual is pinned to its prior value for an unrelated update, the read after apply
// contradicts that plan. A PreApply plan check provides the exact boundary needed to reproduce the
// race: the saved plan contains 8.2, then the fixture starts reporting 8.4 before apply begins.
func TestUnitActiveActiveDatabaseVersion_BackgroundUpgradeAfterPlanIsInconsistent(t *testing.T) {
	api, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfigNamed(subID, "aa-db", "8.2"),
				Check:  resource.TestCheckResourceAttr(aaResourceName, "redis_version_actual", "8.2"),
			},
			{
				Config: newAADatabaseConfigNamed(subID, "aa-db-renamed", "8.2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
						planCheckFunc(func(_ context.Context, _ plancheck.CheckPlanRequest, _ *plancheck.CheckPlanResponse) {
							api.setRunningVersion("8.4")
						}),
					},
				},
				ExpectError: regexp.MustCompile(`(?s)Provider produced inconsistent result after apply.*redis_version_actual.*8\.2.*8\.4`),
			},
		},
	})

	assert.Empty(t, api.requestedUpgrades())
}

// BEHAVIOUR(global_password): in the DEFAULT configuration — no global_password in config and not
// passwordless, so the API generates one — any plan that changes another attribute also plans
// global_password as unknown ("known after apply"). Exercised by
// TestUnitActiveActiveDatabase_GlobalPasswordLeftToAPIIsPlannedUnknown.
//
// Whether this is a defect is genuinely unclear, which is why it is recorded rather than judged:
//   - Arguably correct. The attribute is Optional+Computed and null in config, so Terraform cannot know
//     what the API will return — it could rotate the password. Marking it unknown is conservative, and it
//     settles: the plan straight after apply is clean, so this is not drift or a perpetual diff.
//   - Arguably broken. ModifyPlan (resource_active_active_database.go:597-606) exists specifically to
//     prevent it, and demonstrably does not. Either that code is unnecessary and should go, or it is
//     necessary and something is defeating it.
//
// What the attribute looks like: Optional+Computed, Sensitive, with no Default and no attribute-level
// plan modifier — the only such attribute on this resource. The absence is deliberate; the comment on
// ModifyPlan records that UseStateForUnknown caused "inconsistent values for sensitive attribute" errors
// on passwordless transitions, so preservation was moved into ModifyPlan by hand.
//
// Ruled out by instrumentation and a TF_LOG=trace run, so nobody repeats it:
//   - state DOES hold the password, and the fixture returns it on every read
//   - ModifyPlan's preservation branch runs on every invocation with prior state; SetAttribute returns
//     zero diagnostics; the raw value the provider returns is tftypes.String<"..."> — known, not unknown
//   - the framework's ordering is correct: MarkComputedNilsAsUnknown, attribute modifiers, ModifyPlan,
//     then resp.PlannedState = the modified plan, with nothing after it touching the value
//   - Sensitive: true is NOT the cause — removing it changes nothing
//   - the "Value switched to prior value due to semantic equality logic" lines in the trace are a red
//     herring: that message fires whenever newValue equals priorValue (value_semantic_equality.go:90),
//     and only during ReadResource, not PlanResourceChange
//
// So the provider returns a known value and Terraform records after_unknown: true. If this is worth
// chasing, the next step is a minimal resource with one Optional+Computed Sensitive attribute and a
// ModifyPlan that preserves it, to establish whether the gap is in terraform-plugin-go's proto conversion
// or in Terraform core — rather than adding a third workaround here.
//
// Practical consequence either way: a test cannot assert a genuinely empty plan for a no-op config change
// unless global_password is set in config, which is why every config here except
// newAADatabaseConfigNoPassword sets it.
//
// NOT verified against the real API. One thing could differ there: if the API returns the password only
// on create, readDatabase writes null to state on refresh, which would be a separate and worse problem
// this fixture cannot reproduce.

// Exercises BEHAVIOUR(global_password) below. In the default configuration — no global_password in config, not
// passwordless, so the API generates one — any plan that changes another attribute ALSO shows
// global_password as "known after apply", because the framework marks computed attributes unknown
// whenever config leaves them null. Here the change is a rename: nothing to do with versions.
//
// Scope, verified rather than assumed: this is confined to the plan of a step that changes something.
// The plan straight after apply is CLEAN — asserted below — so it is not drift or a perpetual diff.
func TestUnitActiveActiveDatabase_GlobalPasswordLeftToAPIIsPlannedUnknown(t *testing.T) {
	_, factories, subID := newAAFixture(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfigNoPassword(subID, "aa-db"),
				// The generated password IS persisted, so a missing state value is not the cause.
				Check: resource.TestCheckResourceAttrSet(aaResourceName, "global_password"),
			},
			{
				Config: newAADatabaseConfigNoPassword(subID, "aa-db-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(aaResourceName, plancheck.ResourceActionUpdate),
						// The bug: nothing in config or state justifies this being unknown.
						plancheck.ExpectUnknownValue(aaResourceName, tfjsonpath.New("global_password")),
					},
					// And it settles: once there is nothing else to change, the plan is clean.
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
