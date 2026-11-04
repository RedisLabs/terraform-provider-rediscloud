package pro_test

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// proSubscriptionPlanState is the state of a subscription whose VPC was created by
// Redis Cloud, as both the Framework and SDKv2 2.20.0 read it from the API.
func proSubscriptionPlanState(t *testing.T, s schema.Schema) tftypes.Value {
	t.Helper()
	ctx := context.Background()

	region := types.ObjectValueMust(pro.SubscriptionRegionElemType.AttrTypes, map[string]attr.Value{
		"region":                       types.StringValue("eu-west-1"),
		"multiple_availability_zones":  types.BoolValue(false),
		"preferred_availability_zones": types.ListValueMust(types.StringType, []attr.Value{types.StringValue("eu-west-1c")}),
		"networking_deployment_cidr":   types.StringValue("10.0.0.0/24"),
		"networking_vpc_id":            types.StringValue("vpc-api"),
		"networks": types.ListValueMust(pro.SubscriptionNetworkElemType, []attr.Value{
			types.ObjectValueMust(pro.SubscriptionNetworkElemType.AttrTypes, map[string]attr.Value{
				"networking_subnet_id":       types.StringValue("subnet-api"),
				"networking_deployment_cidr": types.StringValue("10.0.0.0/24"),
				"networking_vpc_id":          types.StringValue("vpc-api"),
			}),
		}),
	})
	creationPlan := types.ObjectValueMust(pro.CreationPlanElemType.AttrTypes, map[string]attr.Value{
		"memory_limit_in_gb":           types.Float64Value(0),
		"dataset_size_in_gb":           types.Float64Value(1),
		"query_performance_factor":     types.StringValue(""),
		"throughput_measurement_by":    types.StringValue("operations-per-second"),
		"throughput_measurement_value": types.Int64Value(1000),
		"average_item_size_in_bytes":   types.Int64Value(0),
		"ram_percentage":               types.Int64Value(0),
		"quantity":                     types.Int64Value(1),
		"support_oss_cluster_api":      types.BoolValue(false),
		"replication":                  types.BoolValue(false),
		"modules":                      types.ListValueMust(types.StringType, []attr.Value{}),
	})
	model := pro.ProSubscriptionResourceModel{
		ID:              types.StringValue("123"),
		Name:            types.StringValue("sub"),
		PaymentMethod:   types.StringValue("credit-card"),
		PaymentMethodID: types.StringValue("1"),
		MemoryStorage:   types.StringValue("ram"),
		Allowlist:       types.ListValueMust(pro.SubscriptionAllowlistElemType, []attr.Value{}),
		CloudProvider: types.ListValueMust(pro.SubscriptionCloudProviderElemType, []attr.Value{
			types.ObjectValueMust(pro.SubscriptionCloudProviderElemType.AttrTypes, map[string]attr.Value{
				"provider":         types.StringValue("AWS"),
				"cloud_account_id": types.StringValue("1"),
				"aws_account_id":   types.StringValue("123456789012"),
				"region":           types.SetValueMust(pro.SubscriptionRegionElemType, []attr.Value{region}),
				"resource_tags":    types.MapNull(types.StringType),
			}),
		}),
		CreationPlan:                          types.ListValueMust(pro.CreationPlanElemType, []attr.Value{creationPlan}),
		RedisVersion:                          types.StringNull(),
		MaintenanceWindows:                    pro.EmptyMaintenanceList(),
		Pricing:                               types.ListValueMust(utils.PricingElemType, []attr.Value{}),
		CustomerManagedKeyEnabled:             types.BoolValue(false),
		CustomerManagedKeyDeletionGracePeriod: types.StringValue("immediate"),
		CustomerManagedKey:                    types.ListValueMust(pro.CustomerManagedKeyElemType, []attr.Value{}),
		CustomerManagedKeyRedisServiceAccount: types.StringNull(),
		CustomerManagedKeyAwsRoleArn:          types.StringNull(),
		PublicEndpointAccess:                  types.BoolValue(true),
		PrometheusEndpoint:                    types.StringValue("prometheus.example:8070"),
		Timeouts:                              timeouts.Value{Object: types.ObjectNull(s.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)},
	}

	state := tfsdk.State{Schema: s}
	require.False(t, state.Set(ctx, &model).HasError())
	return state.Raw
}

// configFromState returns the configuration that omits every computed attribute of state.
func configFromState(t *testing.T, s schema.Schema, state tftypes.Value) tftypes.Value {
	t.Helper()
	config, err := tftypes.Transform(state, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		// Blocks and the root have no attribute schema and keep their value.
		if a, err := s.AttributeAtTerraformPath(context.Background(), p); err == nil && a.IsComputed() {
			return tftypes.NewValue(v.Type(), nil), nil
		}
		return v, nil
	})
	require.NoError(t, err)
	return config
}

func isRegionAttr(name string) func(*tftypes.AttributePath) bool {
	return func(p *tftypes.AttributePath) bool {
		steps := p.Steps()
		return len(steps) == 5 && steps[4].Equal(tftypes.AttributeName(name))
	}
}

func setValues(t *testing.T, v tftypes.Value, match func(*tftypes.AttributePath) bool, value tftypes.Value) tftypes.Value {
	t.Helper()
	out, err := tftypes.Transform(v, func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
		if match(p) {
			return value, nil
		}
		return v, nil
	})
	require.NoError(t, err)
	return out
}

func isRegions(p *tftypes.AttributePath) bool {
	return p.Equal(tftypes.NewAttributePath().WithAttributeName("cloud_provider").WithElementKeyInt(0).WithAttributeName("region"))
}

func isName(p *tftypes.AttributePath) bool {
	return p.Equal(tftypes.NewAttributePath().WithAttributeName("name"))
}

func planProSubscription(t *testing.T, s schema.Schema, prior, config, proposed tftypes.Value) (tftypes.Value, *tfprotov5.PlanResourceChangeResponse) {
	t.Helper()
	ctx := context.Background()
	// A plan that replaces the subscription reads it; the empty fake API fails that read.
	api := testhelpers.NewAPIClient(t, testhelpers.NewFakeAPI())
	server, err := testhelpers.FrameworkProviderFactories(api, pro.NewProSubscriptionResource)[testhelpers.FrameworkTestProviderTypeName]()
	require.NoError(t, err)
	_, err = server.ConfigureProvider(ctx, &tfprotov5.ConfigureProviderRequest{})
	require.NoError(t, err)

	typ := s.Type().TerraformType(ctx)
	dynamic := func(v tftypes.Value) *tfprotov5.DynamicValue {
		dv, err := tfprotov5.NewDynamicValue(typ, v)
		require.NoError(t, err)
		return &dv
	}
	resp, err := server.PlanResourceChange(ctx, &tfprotov5.PlanResourceChangeRequest{
		TypeName:         testhelpers.FrameworkTestProviderTypeName + "_subscription",
		PriorState:       dynamic(prior),
		ProposedNewState: dynamic(proposed),
		Config:           dynamic(config),
	})
	require.NoError(t, err)
	for _, d := range resp.Diagnostics {
		t.Fatalf("plan diagnostic: %s: %s", d.Summary, d.Detail)
	}
	planned, err := resp.PlannedState.Unmarshal(typ)
	require.NoError(t, err)
	return planned, resp
}

func regionsOf(t *testing.T, v tftypes.Value) tftypes.Value {
	t.Helper()
	found, _, err := tftypes.WalkAttributePath(v, tftypes.NewAttributePath().WithAttributeName("cloud_provider").WithElementKeyInt(0).WithAttributeName("region"))
	require.NoError(t, err)
	return found.(tftypes.Value)
}

func proSubscriptionSchema(t *testing.T) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	pro.NewProSubscriptionResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	return resp.Schema
}

func TestProSubscriptionPlanKeepsAPIVPCIDWhenOmitted(t *testing.T) {
	s := proSubscriptionSchema(t)
	prior := proSubscriptionPlanState(t, s)
	config := configFromState(t, s, prior)

	planned, resp := planProSubscription(t, s, prior, config, prior)
	assert.True(t, planned.Equal(prior), "omitting networking_vpc_id must give an empty plan, got %s", planned)
	assert.Empty(t, resp.RequiresReplace)
}

func TestProSubscriptionPlanKeepsAPIVPCIDOnOtherChange(t *testing.T) {
	s := proSubscriptionSchema(t)
	prior := proSubscriptionPlanState(t, s)
	renamed := tftypes.NewValue(tftypes.String, "renamed")
	config := setValues(t, configFromState(t, s, prior), isName, renamed)
	proposed := setValues(t, prior, isName, renamed)

	planned, resp := planProSubscription(t, s, prior, config, proposed)
	assert.True(t, regionsOf(t, planned).Equal(regionsOf(t, prior)), "regions must keep their state values, got %s", regionsOf(t, planned))
	assert.Empty(t, resp.RequiresReplace)
}

func TestProSubscriptionPlanIgnoresChangedRegionValues(t *testing.T) {
	s := proSubscriptionSchema(t)
	prior := proSubscriptionPlanState(t, s)
	zones := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "eu-west-1a")})
	for name, change := range map[string]struct {
		attr  string
		value tftypes.Value
	}{
		"vpc":   {"networking_vpc_id", tftypes.NewValue(tftypes.String, "vpc-configured")},
		"cidr":  {"networking_deployment_cidr", tftypes.NewValue(tftypes.String, "10.1.0.0/24")},
		"zones": {"preferred_availability_zones", zones},
	} {
		t.Run(name, func(t *testing.T) {
			config := setValues(t, configFromState(t, s, prior), isRegionAttr(change.attr), change.value)
			// Terraform matches no prior region to the changed one, so it proposes the configured region.
			proposed := setValues(t, prior, isRegions, regionsOf(t, config))

			planned, resp := planProSubscription(t, s, prior, config, proposed)
			assert.True(t, planned.Equal(prior), "Update never changes a region, so the plan must be empty, got %s", planned)
			assert.Empty(t, resp.RequiresReplace)
		})
	}
}

func TestProSubscriptionPlanMultiAZRegionIsStable(t *testing.T) {
	s := proSubscriptionSchema(t)
	singleAZ := proSubscriptionPlanState(t, s)
	multiAZ := setValues(t, singleAZ, isRegionAttr("multiple_availability_zones"), tftypes.NewValue(tftypes.Bool, true))
	config := setValues(t, configFromState(t, s, multiAZ), isRegionAttr("multiple_availability_zones"), tftypes.NewValue(tftypes.Bool, true))
	// Read stores no CIDR or VPC id for a multi-AZ region, as SDKv2 2.20.0 did.
	prior := setValues(t, multiAZ, isRegionAttr("networking_deployment_cidr"), tftypes.NewValue(tftypes.String, ""))
	prior = setValues(t, prior, isRegionAttr("networking_vpc_id"), tftypes.NewValue(tftypes.String, ""))
	proposed := setValues(t, prior, isRegions, regionsOf(t, config))

	planned, resp := planProSubscription(t, s, prior, config, proposed)
	assert.True(t, planned.Equal(prior), "a multi-AZ region read from the API must give an empty plan, got %s", planned)
	assert.Empty(t, resp.RequiresReplace)
}

func TestProSubscriptionPlanConfiguredMultiAZRegionIsStable(t *testing.T) {
	s := proSubscriptionSchema(t)
	// Create stores the configured CIDR and the empty VPC id that the API mapping gives a multi-AZ region.
	prior := setValues(t, proSubscriptionPlanState(t, s), isRegionAttr("multiple_availability_zones"), tftypes.NewValue(tftypes.Bool, true))
	prior = setValues(t, prior, isRegionAttr("networking_vpc_id"), tftypes.NewValue(tftypes.String, ""))
	config := setValues(t, configFromState(t, s, prior), isRegionAttr("multiple_availability_zones"), tftypes.NewValue(tftypes.Bool, true))

	// Terraform matches the configured region to the prior one and proposes the prior values.
	planned, resp := planProSubscription(t, s, prior, config, prior)
	assert.True(t, planned.Equal(prior), "a configured multi-AZ region must give an empty plan, got %s", regionsOf(t, planned))
	assert.Empty(t, resp.RequiresReplace)
}

func TestProSubscriptionPlanConfiguredRegionValuesMatchingState(t *testing.T) {
	s := proSubscriptionSchema(t)
	prior := proSubscriptionPlanState(t, s)
	config := setValues(t, configFromState(t, s, prior), isRegions, regionsOf(t, prior))
	config = setValues(t, config, isRegionAttr("networks"), tftypes.NewValue(types.ListType{ElemType: pro.SubscriptionNetworkElemType}.TerraformType(context.Background()), nil))

	planned, resp := planProSubscription(t, s, prior, config, prior)
	assert.True(t, planned.Equal(prior), "configuring the API values must give an empty plan, got %s", planned)
	assert.Empty(t, resp.RequiresReplace)
}

func isCreationPlanAttr(name string) func(*tftypes.AttributePath) bool {
	return func(p *tftypes.AttributePath) bool {
		steps := p.Steps()
		return len(steps) == 3 && steps[0].Equal(tftypes.AttributeName("creation_plan")) && steps[2].Equal(tftypes.AttributeName(name))
	}
}

func TestProSubscriptionPlanIgnoresLegacyCreationPlanZeroValues(t *testing.T) {
	s := proSubscriptionSchema(t)
	// SDKv2 2.20.0 stored 0 for the creation plan sizes that were not configured.
	prior := proSubscriptionPlanState(t, s)
	config := setValues(t, configFromState(t, s, prior), isCreationPlanAttr("memory_limit_in_gb"), tftypes.NewValue(tftypes.Number, nil))
	proposed := setValues(t, prior, isCreationPlanAttr("memory_limit_in_gb"), tftypes.NewValue(tftypes.Number, nil))

	planned, resp := planProSubscription(t, s, prior, config, proposed)
	assert.True(t, planned.Equal(prior), "creation_plan changes are ignored after creation, got %s", planned)
	assert.Empty(t, resp.RequiresReplace)
}

func TestProSubscriptionPlanIgnoresCreationPlanChanges(t *testing.T) {
	s := proSubscriptionSchema(t)
	prior := proSubscriptionPlanState(t, s)
	for name, value := range map[string]tftypes.Value{
		"removed": tftypes.NewValue(tftypes.Number, nil),
		"changed": tftypes.NewValue(tftypes.Number, 2),
	} {
		t.Run(name, func(t *testing.T) {
			config := setValues(t, configFromState(t, s, prior), isCreationPlanAttr("dataset_size_in_gb"), value)
			proposed := setValues(t, prior, isCreationPlanAttr("dataset_size_in_gb"), value)

			planned, resp := planProSubscription(t, s, prior, config, proposed)
			assert.True(t, planned.Equal(prior), "creation_plan changes are ignored after creation, got %s", planned)
			assert.Empty(t, resp.RequiresReplace)
		})
	}
}

func isMaintenanceWindows(p *tftypes.AttributePath) bool {
	return p.Equal(tftypes.NewAttributePath().WithAttributeName("maintenance_windows"))
}

func maintenanceWindowsValue(t *testing.T, m *maintenance.Maintenance) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	list, diags := customtypes.NewMaintenanceList(ctx, m)
	require.False(t, diags.HasError(), "%v", diags)
	v, err := list.ToTerraformValue(ctx)
	require.NoError(t, err)
	return v
}

func valueToJSON(t *testing.T, v tftypes.Value) any {
	t.Helper()
	if v.IsNull() {
		return nil
	}
	switch {
	case v.Type().Is(tftypes.String):
		var s string
		require.NoError(t, v.As(&s))
		return s
	case v.Type().Is(tftypes.Bool):
		var b bool
		require.NoError(t, v.As(&b))
		return b
	case v.Type().Is(tftypes.Number):
		var n big.Float
		require.NoError(t, v.As(&n))
		return json.Number(n.Text('g', -1))
	case v.Type().Is(tftypes.Object{}), v.Type().Is(tftypes.Map{}):
		var m map[string]tftypes.Value
		require.NoError(t, v.As(&m))
		out := make(map[string]any, len(m))
		for k, e := range m {
			out[k] = valueToJSON(t, e)
		}
		return out
	default:
		var l []tftypes.Value
		require.NoError(t, v.As(&l))
		out := make([]any, 0, len(l))
		for _, e := range l {
			out = append(out, valueToJSON(t, e))
		}
		return out
	}
}

// upgradeSDKProSubscriptionState runs UpgradeResourceState on a state stored by SDKv2
// 2.20.0 (schema version 0), as Terraform does before planning.
func upgradeSDKProSubscriptionState(t *testing.T, s schema.Schema, sdkState tftypes.Value) tftypes.Value {
	t.Helper()
	raw, err := json.Marshal(valueToJSON(t, sdkState))
	require.NoError(t, err)
	return upgradeSDKProSubscriptionJSON(t, s, raw)
}

func upgradeSDKProSubscriptionJSON(t *testing.T, s schema.Schema, raw []byte) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	server, err := testhelpers.FrameworkProviderFactories(nil, pro.NewProSubscriptionResource)[testhelpers.FrameworkTestProviderTypeName]()
	require.NoError(t, err)

	resp, err := server.UpgradeResourceState(ctx, &tfprotov5.UpgradeResourceStateRequest{
		TypeName: testhelpers.FrameworkTestProviderTypeName + "_subscription",
		Version:  0,
		RawState: &tfprotov5.RawState{JSON: raw},
	})
	require.NoError(t, err)
	require.Empty(t, resp.Diagnostics)
	upgraded, err := resp.UpgradedState.Unmarshal(s.Type().TerraformType(ctx))
	require.NoError(t, err)
	return upgraded
}

func omittedMaintenanceWindows(t *testing.T) tftypes.Value {
	t.Helper()
	omitted, err := pro.EmptyMaintenanceList().ToTerraformValue(context.Background())
	require.NoError(t, err)
	return omitted
}

func manualMaintenanceWindows(t *testing.T) tftypes.Value {
	t.Helper()
	return maintenanceWindowsValue(t, &maintenance.Maintenance{
		Mode: redis.String("manual"),
		Windows: []*maintenance.Window{{
			StartHour:       redis.Int(22),
			DurationInHours: redis.Int(8),
			Days:            redis.StringSlice("Tuesday"),
		}},
	})
}

// assertEmptyPlanWithoutMaintenanceWindows plans an upgraded SDKv2 state with a
// configuration that omits the maintenance_windows block.
func assertEmptyPlanWithoutMaintenanceWindows(t *testing.T, s schema.Schema, prior tftypes.Value) {
	t.Helper()
	omitted := omittedMaintenanceWindows(t)
	config := setValues(t, configFromState(t, s, prior), isMaintenanceWindows, omitted)
	proposed := setValues(t, prior, isMaintenanceWindows, omitted)

	planned, resp := planProSubscription(t, s, prior, config, proposed)
	assert.True(t, planned.Equal(prior), "an omitted maintenance_windows block must give an empty plan, got %s", planned)
	assert.Empty(t, resp.RequiresReplace)
}

func TestProSubscriptionPlanSDKAutomaticMaintenanceWindowsWithoutConfig(t *testing.T) {
	s := proSubscriptionSchema(t)
	// SDKv2 2.20.0 filled the optional and computed maintenance_windows from the API.
	sdkState := setValues(t, proSubscriptionPlanState(t, s), isMaintenanceWindows, maintenanceWindowsValue(t, &maintenance.Maintenance{Mode: redis.String("automatic")}))
	assertEmptyPlanWithoutMaintenanceWindows(t, s, upgradeSDKProSubscriptionState(t, s, sdkState))
}

func TestProSubscriptionPlanSDKManualMaintenanceWindowsWithoutConfig(t *testing.T) {
	s := proSubscriptionSchema(t)
	// Manual windows set outside Terraform, or kept by SDKv2 after the block was removed.
	sdkState := setValues(t, proSubscriptionPlanState(t, s), isMaintenanceWindows, manualMaintenanceWindows(t))

	prior := upgradeSDKProSubscriptionState(t, s, sdkState)
	assert.True(t, prior.Equal(setValues(t, sdkState, isMaintenanceWindows, omittedMaintenanceWindows(t))), "the upgrade must only drop maintenance_windows, got %s", prior)
	assertEmptyPlanWithoutMaintenanceWindows(t, s, prior)
}

func TestProSubscriptionPlanSDKManualMaintenanceWindowsConfigured(t *testing.T) {
	s := proSubscriptionSchema(t)
	manual := manualMaintenanceWindows(t)
	sdkState := setValues(t, proSubscriptionPlanState(t, s), isMaintenanceWindows, manual)

	prior := upgradeSDKProSubscriptionState(t, s, sdkState)
	config := setValues(t, configFromState(t, s, sdkState), isMaintenanceWindows, manual)
	proposed := setValues(t, prior, isMaintenanceWindows, manual)

	// The configured block is applied once, which sends the same windows to the API.
	planned, resp := planProSubscription(t, s, prior, config, proposed)
	found, _, err := tftypes.WalkAttributePath(planned, tftypes.NewAttributePath().WithAttributeName("maintenance_windows"))
	require.NoError(t, err)
	assert.True(t, found.(tftypes.Value).Equal(manual), "the configured block must be planned, got %s", found)
	assert.Empty(t, resp.RequiresReplace)
}

func elementsAt(t *testing.T, v tftypes.Value, p *tftypes.AttributePath) []tftypes.Value {
	t.Helper()
	found, _, err := tftypes.WalkAttributePath(v, p)
	require.NoError(t, err)
	value := found.(tftypes.Value)
	if value.Type().Is(tftypes.Map{}) {
		var m map[string]tftypes.Value
		require.NoError(t, value.As(&m))
		out := make([]tftypes.Value, 0, len(m))
		for _, e := range m {
			out = append(out, e)
		}
		return out
	}
	var l []tftypes.Value
	require.NoError(t, value.As(&l))
	return l
}

// TestProSubscriptionPlanSDK2200StateFixture upgrades a state in the JSON shape that
// SDKv2 2.20.0 writes, including timeouts and empty string attributes.
func TestProSubscriptionPlanSDK2200StateFixture(t *testing.T) {
	s := proSubscriptionSchema(t)
	raw, err := os.ReadFile("testdata/sdk_2_20_0_subscription_state.json")
	require.NoError(t, err)

	for _, mode := range []string{"automatic", "manual"} {
		t.Run(mode, func(t *testing.T) {
			var state map[string]any
			require.NoError(t, json.Unmarshal(raw, &state))
			if mode == "manual" {
				state["maintenance_windows"] = []any{map[string]any{
					"mode": "manual",
					"window": []any{map[string]any{
						"start_hour":        22,
						"duration_in_hours": 8,
						"days":              []any{"Tuesday"},
					}},
				}}
			}
			patched, err := json.Marshal(state)
			require.NoError(t, err)

			prior := upgradeSDKProSubscriptionJSON(t, s, patched)
			assert.Empty(t, elementsAt(t, prior, tftypes.NewAttributePath().WithAttributeName("maintenance_windows")))
			assert.Len(t, elementsAt(t, prior, tftypes.NewAttributePath().WithAttributeName("pricing")), 1)
			assert.Len(t, elementsAt(t, prior, tftypes.NewAttributePath().WithAttributeName("cloud_provider").WithElementKeyInt(0).WithAttributeName("resource_tags")), 2)
			assertEmptyPlanWithoutMaintenanceWindows(t, s, prior)
		})
	}
}
