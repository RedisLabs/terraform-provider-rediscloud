package pro

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func proSubscriptionSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	NewProSubscriptionResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
	return resp
}

func TestProSubscriptionResourceSchemaMatchesModel(t *testing.T) {
	resp := proSubscriptionSchema(t)

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ProSubscriptionResourceModel{
		Allowlist:          types.ListNull(subscriptionAllowlistElemType),
		CloudProvider:      types.ListNull(subscriptionCloudProviderElemType),
		CreationPlan:       types.ListNull(creationPlanElemType),
		MaintenanceWindows: emptyMaintenanceList(),
		Pricing:            types.ListNull(utils.PricingElemType),
		CustomerManagedKey: types.ListNull(customerManagedKeyElemType),
		Timeouts:           timeouts.Value{Object: types.ObjectNull(resp.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)},
	})}
	assert.True(t, resp.Schema.Type().Equal(modelType), "resource schema must match its model type")
	assert.False(t, resp.Schema.ValidateImplementation(context.Background()).HasError())
}

func TestCreateOnlyModifierKeepsState(t *testing.T) {
	state := types.StringValue("prior")
	cases := []struct {
		name     string
		computed bool
		create   bool
		config   attr.Value
		state    attr.Value
		want     bool
	}{
		{"create", false, true, types.StringValue("new"), types.StringNull(), false},
		{"update keeps configured value", false, false, types.StringValue("new"), state, true},
		{"update with removed non-computed value", false, false, types.StringNull(), state, false},
		{"update with removed computed value", true, false, types.StringNull(), state, true},
		{"null prior value", true, false, types.StringValue("new"), types.StringNull(), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, createOnly(c.computed).keepState(c.create, c.config, c.state))
		})
	}
}

func TestCidrValidator(t *testing.T) {
	for value, valid := range map[string]bool{
		"10.0.0.0/24":   true,
		"10.0.0.1/24":   true,
		"192.168.0.0":   false,
		"not-a-network": false,
	} {
		resp := validator.StringResponse{}
		cidrValidator{}.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root("cidr"),
			ConfigValue: types.StringValue(value),
		}, &resp)
		assert.Equal(t, !valid, resp.Diagnostics.HasError(), value)
	}
}

func TestRedisVersionRequiresReplace(t *testing.T) {
	cases := []struct {
		name   string
		config types.String
		state  types.String
		want   bool
	}{
		{"removed", types.StringNull(), types.StringValue("7.4"), false},
		{"empty", types.StringValue(""), types.StringValue("7.4"), false},
		{"unchanged", types.StringValue("7.4"), types.StringValue("7.4"), false},
		{"changed", types.StringValue("latest"), types.StringValue("7.4"), true},
		{"added", types.StringValue("latest"), types.StringNull(), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := stringplanmodifier.RequiresReplaceIfFuncResponse{}
			redisVersionRequiresReplace(context.Background(), planmodifier.StringRequest{ConfigValue: c.config, StateValue: c.state}, &resp)
			assert.Equal(t, c.want, resp.RequiresReplace)
		})
	}
}

func TestSubscriptionRegionsFromAPI(t *testing.T) {
	ctx := context.Background()
	regions, diags := subscriptionRegionsFromAPI(ctx, []*subscriptions.Region{
		{
			Region:                    redis.String("eu-west-1"),
			MultipleAvailabilityZones: redis.Bool(false),
			Networking: []*subscriptions.Networking{{
				DeploymentCIDR: redis.String("10.0.0.0/24"),
				VPCId:          redis.String("vpc-1"),
				SubnetID:       redis.String("subnet-1"),
			}},
		},
		{
			Region:                     redis.String("us-east-1"),
			MultipleAvailabilityZones:  redis.Bool(true),
			PreferredAvailabilityZones: redis.StringSlice("use1-az1", "use1-az2", "use1-az4"),
			Networking: []*subscriptions.Networking{
				{DeploymentCIDR: redis.String("10.1.0.0/24"), VPCId: redis.String("vpc-2"), SubnetID: redis.String("subnet-2")},
			},
		},
	})
	require.False(t, diags.HasError(), "%v", diags)

	var models []SubscriptionRegionModel
	require.False(t, regions.ElementsAs(ctx, &models, false).HasError())
	byRegion := map[string]SubscriptionRegionModel{}
	for _, m := range models {
		byRegion[m.Region.ValueString()] = m
	}

	single := byRegion["eu-west-1"]
	assert.Equal(t, "10.0.0.0/24", single.NetworkingDeploymentCIDR.ValueString())
	assert.Equal(t, "vpc-1", single.NetworkingVPCID.ValueString())
	assert.False(t, single.PreferredAvailabilityZones.IsNull())
	assert.Empty(t, single.PreferredAvailabilityZones.Elements())
	assert.Len(t, single.Networks.Elements(), 1)

	multi := byRegion["us-east-1"]
	assert.Equal(t, "", multi.NetworkingDeploymentCIDR.ValueString())
	assert.Equal(t, "", multi.NetworkingVPCID.ValueString())
	assert.Len(t, multi.PreferredAvailabilityZones.Elements(), 3)
}

func TestSubscriptionCloudProvidersFromAPIResourceTags(t *testing.T) {
	ctx := context.Background()
	details := []*subscriptions.CloudDetail{{
		Provider:       redis.String("AWS"),
		CloudAccountID: redis.Int(1),
	}}

	withoutTags, diags := subscriptionCloudProvidersFromAPI(ctx, details, false)
	require.False(t, diags.HasError())
	tags, diags := cloudProviderResourceTags(ctx, withoutTags)
	require.False(t, diags.HasError())
	assert.True(t, tags.IsNull())

	withEmptyTags, diags := subscriptionCloudProvidersFromAPI(ctx, details, true)
	require.False(t, diags.HasError())
	tags, diags = cloudProviderResourceTags(ctx, withEmptyTags)
	require.False(t, diags.HasError())
	assert.False(t, tags.IsNull())
	assert.Empty(t, tags.Elements())
}

func TestFillRegionComputedValues(t *testing.T) {
	networks := types.ListValueMust(subscriptionNetworkElemType, []attr.Value{})
	azs := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("eu-west-1a")})
	region := func(name string, maz bool, n types.List, a types.List) SubscriptionRegionModel {
		return SubscriptionRegionModel{
			Region:                     types.StringValue(name),
			MultipleAvailabilityZones:  types.BoolValue(maz),
			PreferredAvailabilityZones: a,
			NetworkingDeploymentCIDR:   types.StringValue("10.0.0.0/24"),
			NetworkingVPCID:            types.StringValue(""),
			Networks:                   n,
		}
	}
	state := []SubscriptionRegionModel{region("eu-west-1", false, networks, azs)}

	state[0].NetworkingVPCID = types.StringValue("vpc-1")

	plan := []SubscriptionRegionModel{region("eu-west-1", false, types.ListUnknown(subscriptionNetworkElemType), types.ListUnknown(types.StringType))}
	plan[0].NetworkingVPCID = types.StringUnknown()
	assert.False(t, fillRegionComputedValues(plan, state))
	assert.True(t, plan[0].Networks.Equal(networks))
	assert.True(t, plan[0].PreferredAvailabilityZones.Equal(azs))
	assert.Equal(t, "vpc-1", plan[0].NetworkingVPCID.ValueString())

	plan = []SubscriptionRegionModel{region("eu-west-1", false, networks, azs)}
	plan[0].NetworkingVPCID = types.StringValue("vpc-2")
	assert.False(t, fillRegionComputedValues(plan, state))
	assert.Equal(t, "vpc-2", plan[0].NetworkingVPCID.ValueString())

	plan = []SubscriptionRegionModel{region("eu-west-1", true, types.ListUnknown(subscriptionNetworkElemType), types.ListUnknown(types.StringType))}
	assert.True(t, fillRegionComputedValues(plan, state))
	assert.True(t, plan[0].Networks.IsUnknown())

	plan = []SubscriptionRegionModel{region("eu-west-1", false, networks, azs), region("us-east-1", false, networks, azs)}
	assert.True(t, fillRegionComputedValues(plan, state))
}

func TestMaintenanceWindowsFromAPI(t *testing.T) {
	ctx := context.Background()
	automatic := &maintenance.Maintenance{Mode: redis.String("automatic")}
	manual := &maintenance.Maintenance{
		Mode: redis.String("manual"),
		Windows: []*maintenance.Window{{
			StartHour:       redis.Int(22),
			DurationInHours: redis.Int(8),
			Days:            redis.StringSlice("Monday"),
		}},
	}

	cases := []struct {
		name  string
		api   *maintenance.Maintenance
		cfg   *proSubscriptionConfig
		count int
	}{
		{"configured automatic", automatic, &proSubscriptionConfig{MaintenanceWindows: true}, 1},
		{"not configured", manual, &proSubscriptionConfig{}, 0},
		{"unknown configuration with automatic mode", automatic, nil, 0},
		{"unknown configuration with manual mode", manual, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			windows, diags := maintenanceWindowsFromAPI(ctx, c.api, c.cfg)
			require.False(t, diags.HasError(), "%v", diags)
			assert.False(t, windows.IsNull())
			assert.Len(t, windows.Elements(), c.count)
		})
	}
}

func TestMaintenanceRequestFromModel(t *testing.T) {
	ctx := context.Background()

	request, diags := maintenanceRequestFromModel(ctx, emptyMaintenanceList())
	require.False(t, diags.HasError())
	assert.Equal(t, "automatic", redis.StringValue(request.Mode))
	assert.Nil(t, request.Windows)

	windows, diags := customtypes.NewMaintenanceList(ctx, &maintenance.Maintenance{
		Mode: redis.String("manual"),
		Windows: []*maintenance.Window{{
			StartHour:       redis.Int(22),
			DurationInHours: redis.Int(8),
			Days:            redis.StringSlice("Monday", "Thursday"),
		}},
	})
	require.False(t, diags.HasError())
	request, diags = maintenanceRequestFromModel(ctx, windows)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "manual", redis.StringValue(request.Mode))
	require.Len(t, request.Windows, 1)
	assert.Equal(t, 22, redis.IntValue(request.Windows[0].StartHour))
	assert.Equal(t, 8, redis.IntValue(request.Windows[0].DurationInHours))
	assert.Equal(t, []string{"Monday", "Thursday"}, redis.StringSliceValue(request.Windows[0].Days...))
}

func TestFillUnknownValues(t *testing.T) {
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"a": tftypes.String, "b": tftypes.String}}
	target := tftypes.NewValue(objType, map[string]tftypes.Value{
		"a": tftypes.NewValue(tftypes.String, "planned"),
		"b": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})
	source := tftypes.NewValue(objType, map[string]tftypes.Value{
		"a": tftypes.NewValue(tftypes.String, "api"),
		"b": tftypes.NewValue(tftypes.String, "api"),
	})

	filled, err := fillUnknownValues(target, source, false)
	require.NoError(t, err)
	assert.True(t, filled.Equal(tftypes.NewValue(objType, map[string]tftypes.Value{
		"a": tftypes.NewValue(tftypes.String, "planned"),
		"b": tftypes.NewValue(tftypes.String, "api"),
	})))

	nulled, err := fillUnknownValues(target, tftypes.NewValue(objType, nil), true)
	require.NoError(t, err)
	assert.True(t, nulled.Equal(tftypes.NewValue(objType, map[string]tftypes.Value{
		"a": tftypes.NewValue(tftypes.String, "planned"),
		"b": tftypes.NewValue(tftypes.String, nil),
	})))

	kept, err := fillUnknownValues(target, tftypes.NewValue(objType, nil), false)
	require.NoError(t, err)
	assert.False(t, kept.IsFullyKnown())
}

func TestRequiresReplacement(t *testing.T) {
	cloudProvider := func(provider string) types.List {
		return types.ListValueMust(subscriptionCloudProviderElemType, []attr.Value{
			types.ObjectValueMust(subscriptionCloudProviderElemType.AttrTypes, map[string]attr.Value{
				"provider":         types.StringValue(provider),
				"cloud_account_id": types.StringValue("1"),
				"aws_account_id":   types.StringNull(),
				"region":           types.SetNull(subscriptionRegionElemType),
				"resource_tags":    types.MapNull(types.StringType),
			}),
		})
	}
	model := func(memoryStorage, redisVersion, provider string) *ProSubscriptionResourceModel {
		return &ProSubscriptionResourceModel{
			MemoryStorage: types.StringValue(memoryStorage),
			RedisVersion:  types.StringValue(redisVersion),
			CloudProvider: cloudProvider(provider),
		}
	}
	state := model("ram", "7.4", "AWS")

	assert.False(t, requiresReplacement(model("ram", "", "AWS"), state))
	assert.False(t, requiresReplacement(model("ram", "7.4", "AWS"), state))
	assert.True(t, requiresReplacement(model("ram-and-flash", "", "AWS"), state))
	assert.True(t, requiresReplacement(model("ram", "latest", "AWS"), state))
	assert.True(t, requiresReplacement(model("ram", "", "GCP"), state))
}
