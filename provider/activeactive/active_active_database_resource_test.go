package activeactive_test

// This file exercises the real redis_version / redis_version_actual plan modifiers
// (activeactive.RedisVersion / RedisVersionActual) end-to-end through Terraform's plan/apply
// pipeline, WITHOUT provisioning any real infrastructure.
//
// It defines a minimal "test_active_active_subscription_database" resource that wires those exact
// modifiers onto two attributes and backs its CRUD with an in-memory "running version" (the api
// stub). The generic throwaway provider that serves it lives in testhelpers.FrameworkProviderFactories.
// resource.UnitTest drives real Terraform against it, so the modifiers run as Terraform would run
// them — catching plan idempotency and plan/apply consistency behaviour that direct
// PlanModifyString calls cannot.

import (
	"context"
	"fmt"
	"testing"

	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const defaultRedisVersion = "8.0"

type api struct {
	redisVersion string
}

type activeActiveDatabaseResource struct{ api *api }

var _ tfresource.Resource = &activeActiveDatabaseResource{}

type activeActiveDatabaseModel struct {
	ID                 types.String `tfsdk:"id"`
	RedisVersion       types.String `tfsdk:"redis_version"`
	RedisVersionActual types.String `tfsdk:"redis_version_actual"`
}

func newActiveActiveDatabaseResource(api *api) func() tfresource.Resource {
	return func() tfresource.Resource { return &activeActiveDatabaseResource{api: api} }
}

func (r *activeActiveDatabaseResource) Metadata(_ context.Context, req tfresource.MetadataRequest, resp *tfresource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_subscription_database"
}

func (r *activeActiveDatabaseResource) Schema(_ context.Context, _ tfresource.SchemaRequest, resp *tfresource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"redis_version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					activeactive.RedisVersion(),
				},
			},
			"redis_version_actual": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					activeactive.RedisVersionActual(),
				},
			},
		},
	}
}

// Keep in sync: ./resource_active_active_database_crud.go (createDatabase)
func (r *activeActiveDatabaseResource) Create(ctx context.Context, req tfresource.CreateRequest, resp *tfresource.CreateResponse) {
	var plan activeActiveDatabaseModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mirror the real create API: when redis_version is omitted, the database is created at a
	// server-chosen default version rather than failing.
	requested := defaultRedisVersion
	if !plan.RedisVersion.IsNull() && plan.RedisVersion.ValueString() != "" {
		requested = plan.RedisVersion.ValueString()
	}

	r.api.redisVersion = requested // created at the requested (or default) version

	plan.ID = types.StringValue(acctest.RandomWithPrefix("aa-db"))
	plan.RedisVersion = types.StringValue(requested)
	plan.RedisVersionActual = types.StringValue(r.api.redisVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Keep in sync: ./resource_active_active_database_crud.go (readDatabase)
func (r *activeActiveDatabaseResource) Read(ctx context.Context, req tfresource.ReadRequest, resp *tfresource.ReadResponse) {
	var state activeActiveDatabaseModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.RedisVersion.IsNull() || state.RedisVersion.ValueString() == "" {
		state.RedisVersion = types.StringValue(r.api.redisVersion)
	}
	state.RedisVersionActual = types.StringValue(r.api.redisVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Keep in sync: ./resource_active_active_database_crud.go (updateDatabase, readDatabase)
func (r *activeActiveDatabaseResource) Update(ctx context.Context, req tfresource.UpdateRequest, resp *tfresource.UpdateResponse) {
	var plan activeActiveDatabaseModel
	state := &activeActiveDatabaseModel{}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Mirror updateDatabase's upgrade guard: only issue a version upgrade when the requested
	// redis_version differs from both the prior requested value AND the running version. Together
	// with the redis_version plan modifier (which keeps the prior value when the request is already
	// satisfied) this is what prevents an in-place downgrade after a background auto minor upgrade.
	if !plan.RedisVersion.IsNull() && state != nil && !state.RedisVersion.IsNull() && plan.RedisVersion.ValueString() != state.RedisVersion.ValueString() && plan.RedisVersion.ValueString() != state.RedisVersionActual.ValueString() { //nolint:govet // state != nil kept to mirror production's updateDatabase guard verbatim
		r.api.redisVersion = plan.RedisVersion.ValueString() // UpgradeRedisVersion(target)
	}

	// Mirror readDatabase: heal an empty redis_version from the running version, then always reflect
	// the running version in redis_version_actual. On a genuine upgrade the redis_version_actual
	// modifier left this unknown, so writing the new running version resolves it; otherwise it
	// re-writes the unchanged running version.
	if plan.RedisVersion.IsNull() || plan.RedisVersion.ValueString() == "" {
		plan.RedisVersion = types.StringValue(r.api.redisVersion)
	}
	plan.RedisVersionActual = types.StringValue(r.api.redisVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *activeActiveDatabaseResource) Delete(_ context.Context, _ tfresource.DeleteRequest, _ *tfresource.DeleteResponse) {
}

func newAADatabaseConfigNoVersion() string {
	return `resource "test_active_active_subscription_database" "aa_db" {
}`
}

func newAADatabaseConfig(redisVersion string) string {
	return fmt.Sprintf(`resource "test_active_active_subscription_database" "aa_db" {
  redis_version = %q
}`, redisVersion)
}

func TestUnitActiveActiveDatabaseVersion_DefaultsWhenOmitted(t *testing.T) {
	api := &api{}

	resourceName := "test_active_active_subscription_database.aa_db"
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(newActiveActiveDatabaseResource(api)),
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfigNoVersion(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.0"),
					resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.0"),
				),
			},
			{
				Config: newAADatabaseConfigNoVersion(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestUnitActiveActiveDatabaseVersion_SatisfiedRequestIsNoOp(t *testing.T) {
	api := &api{}

	resourceName := "test_active_active_subscription_database.aa_db"
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(newActiveActiveDatabaseResource(api)),
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfig("8.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.4"),
				),
			},
			{
				Config: newAADatabaseConfig("8.2"), // lower than running 8.4 -> satisfied
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.4"),
				),
			},
		},
	})
}

// A genuine upgrade recomputes the running version without a plan/apply inconsistency (which
// would otherwise surface as "Provider produced inconsistent result after apply").
func TestUnitActiveActiveDatabaseVersion_GenuineUpgrade(t *testing.T) {
	api := &api{}

	resourceName := "test_active_active_subscription_database.aa_db"
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(newActiveActiveDatabaseResource(api)),
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfig("8.2"),
				Check:  resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.2"),
			},
			{
				Config: newAADatabaseConfig("8.4"), // higher than running 8.2 -> genuine upgrade
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.4"),
					resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.4"),
				),
			},
		},
	})
}

// A background auto-upgrade moves the running version past the requested one; a subsequent config
// change to a still-lower version must be a no-op (never an in-place downgrade).
func TestUnitActiveActiveDatabaseVersion_BackgroundUpgradeThenLowerRequestIsNoOp(t *testing.T) {
	api := &api{}

	resourceName := "test_active_active_subscription_database.aa_db"
	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(newActiveActiveDatabaseResource(api)),
		Steps: []resource.TestStep{
			{
				Config: newAADatabaseConfig("8.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.2"),
				),
			},
			{
				PreConfig: func() {
					api.redisVersion = "8.6" // simulate background auto minor upgrade
				},
				Config: newAADatabaseConfig("8.4"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "redis_version", "8.2"),
					resource.TestCheckResourceAttr(resourceName, "redis_version_actual", "8.6"),
				),
			},
		},
	})
}
