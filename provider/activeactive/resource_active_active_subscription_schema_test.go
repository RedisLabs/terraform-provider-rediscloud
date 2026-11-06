package activeactive

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestActiveActiveSubscriptionResourceSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	model := blankSubscriptionModel(t)

	var response resource.SchemaResponse
	NewActiveActiveSubscriptionResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(model)}
	assert.Equal(t, modelType, response.Schema.Type(), "resource schema must match its model type")

	creationPlan := response.Schema.Blocks["creation_plan"].(schema.ListNestedBlock)
	assert.True(t, NewCreationPlanListType().ElementType().Equal(creationPlan.NestedObject.Type()))
	region := creationPlan.NestedObject.Blocks["region"].(schema.SetNestedBlock)
	assert.True(t, NewCreationPlanRegionSetType().ElementType().Equal(region.NestedObject.Type()))

	maintenanceBlock := response.Schema.Blocks["maintenance_windows"].(schema.ListNestedBlock)
	assert.True(t, customtypes.NewMaintenanceListType().ElementType().Equal(maintenanceBlock.NestedObject.Type()))
	window := maintenanceBlock.NestedObject.Blocks["window"].(schema.ListNestedBlock)
	assert.True(t, customtypes.NewMaintenanceWindowListType().ElementType().Equal(window.NestedObject.Type()))

	keys := response.Schema.Blocks["customer_managed_key"].(schema.ListNestedBlock)
	assert.True(t, NewCustomerManagedKeyListType().ElementType().Equal(keys.NestedObject.Type()))

	_, pricingIsAttribute := response.Schema.Attributes["pricing"]
	assert.False(t, pricingIsAttribute, "pricing stays a nested block so protocol v5 state still applies")
	_, maintenanceIsAttribute := response.Schema.Attributes["maintenance_windows"]
	assert.False(t, maintenanceIsAttribute)
}

func TestCreationPlanCustomTypes(t *testing.T) {
	ctx := context.Background()
	regions, diags := NewCreationPlanRegionSetValue(ctx, []CreationPlanRegionModel{{
		Region:                   types.StringValue("us-east-1"),
		NetworkingDeploymentCIDR: types.StringValue("10.0.0.0/24"),
		WriteOperationsPerSecond: types.Int64Value(1000),
		ReadOperationsPerSecond:  types.Int64Value(2000),
	}})
	require.False(t, diags.HasError())

	nilRegions, diags := NewCreationPlanRegionSetValue(ctx, nil)
	require.False(t, diags.HasError())
	assert.True(t, nilRegions.IsNull())
	emptyRegions, diags := NewCreationPlanRegionSetValue(ctx, []CreationPlanRegionModel{})
	require.False(t, diags.HasError())
	assert.False(t, emptyRegions.IsNull())
	assert.Zero(t, len(emptyRegions.Elements()))
	assert.True(t, unknownCreationPlanRegionSetValue().IsUnknown())
	assert.False(t, NewCreationPlanRegionSetType().Equal(types.SetType{ElemType: creationPlanRegionElemType}))

	modules, diags := types.ListValueFrom(ctx, types.StringType, []string{"RedisJSON"})
	require.False(t, diags.HasError())
	original, diags := NewCreationPlanListValue(ctx, []CreationPlanModel{{
		MemoryLimitInGB: types.Float64Null(),
		DatasetSizeInGB: types.Float64Value(1),
		Quantity:        types.Int64Value(1),
		Modules:         modules,
		Region:          regions,
	}})
	require.False(t, diags.HasError())

	tf, err := original.ToTerraformValue(ctx)
	require.NoError(t, err)
	roundTripped, err := NewCreationPlanListType().ValueFromTerraform(ctx, tf)
	require.NoError(t, err)
	assert.True(t, original.Equal(roundTripped))

	decoded, diags := original.AsModels(ctx)
	require.False(t, diags.HasError())
	require.Len(t, decoded, 1)
	assert.Equal(t, 1.0, decoded[0].DatasetSizeInGB.ValueFloat64())
	regionModels, diags := decoded[0].Region.AsModels(ctx)
	require.False(t, diags.HasError())
	require.Len(t, regionModels, 1)
	assert.Equal(t, "us-east-1", regionModels[0].Region.ValueString())

	nilPlan, diags := NewCreationPlanListValue(ctx, nil)
	require.False(t, diags.HasError())
	assert.True(t, nilPlan.IsNull())
	assert.True(t, unknownCreationPlanListValue().IsUnknown())
	assert.False(t, NewCreationPlanListType().Equal(types.ListType{ElemType: creationPlanElemType}))
}

func TestCustomerManagedKeyCustomType(t *testing.T) {
	ctx := context.Background()
	original, diags := NewCustomerManagedKeyListValue(ctx, []CustomerManagedKeyModel{{
		ResourceName: types.StringValue("projects/test/keys/key"),
		Region:       types.StringValue("us-east1"),
	}})
	require.False(t, diags.HasError())

	tf, err := original.ToTerraformValue(ctx)
	require.NoError(t, err)
	roundTripped, err := NewCustomerManagedKeyListType().ValueFromTerraform(ctx, tf)
	require.NoError(t, err)
	assert.True(t, original.Equal(roundTripped))

	decoded, diags := original.AsModels(ctx)
	require.False(t, diags.HasError())
	require.Len(t, decoded, 1)
	assert.Equal(t, "us-east1", decoded[0].Region.ValueString())

	nilKeys, diags := NewCustomerManagedKeyListValue(ctx, nil)
	require.False(t, diags.HasError())
	assert.True(t, nilKeys.IsNull())
	emptyKeys, diags := NewCustomerManagedKeyListValue(ctx, []CustomerManagedKeyModel{})
	require.False(t, diags.HasError())
	assert.False(t, emptyKeys.IsNull())
	assert.True(t, unknownCustomerManagedKeyListValue().IsUnknown())
	assert.False(t, NewCustomerManagedKeyListType().Equal(types.ListType{ElemType: customerManagedKeyElemType}))
}

func TestActiveActiveSubscriptionPlanModifiers(t *testing.T) {
	ctx := context.Background()
	present := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "present")}
	create := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, nil)}

	t.Run("payment method keeps state after create", func(t *testing.T) {
		var response planmodifier.StringResponse
		useStateOnUpdateString().PlanModifyString(ctx, planmodifier.StringRequest{
			State:      present,
			StateValue: types.StringValue("credit-card"),
			PlanValue:  types.StringValue("marketplace"),
		}, &response)
		assert.Equal(t, "credit-card", response.PlanValue.ValueString())

		response = planmodifier.StringResponse{}
		useStateOnUpdateString().PlanModifyString(ctx, planmodifier.StringRequest{
			State:     create,
			PlanValue: types.StringValue("marketplace"),
		}, &response)
		assert.True(t, response.PlanValue.IsNull())
	})

	t.Run("redis version replaces only a real change", func(t *testing.T) {
		var response planmodifier.StringResponse
		redisVersionModifier{}.PlanModifyString(ctx, planmodifier.StringRequest{
			State:       present,
			StateValue:  types.StringValue("7.2"),
			ConfigValue: types.StringValue("8.0"),
			PlanValue:   types.StringValue("8.0"),
		}, &response)
		assert.True(t, response.RequiresReplace)

		response = planmodifier.StringResponse{}
		redisVersionModifier{}.PlanModifyString(ctx, planmodifier.StringRequest{
			State:       present,
			StateValue:  types.StringValue("7.2"),
			ConfigValue: types.StringNull(),
			PlanValue:   types.StringNull(),
		}, &response)
		assert.False(t, response.RequiresReplace)
		assert.Equal(t, "7.2", response.PlanValue.ValueString())

		response = planmodifier.StringResponse{}
		redisVersionModifier{}.PlanModifyString(ctx, planmodifier.StringRequest{
			State:       present,
			StateValue:  types.StringValue("7.2"),
			ConfigValue: types.StringValue(""),
		}, &response)
		assert.False(t, response.RequiresReplace)
		assert.Equal(t, "7.2", response.PlanValue.ValueString())

		response = planmodifier.StringResponse{}
		redisVersionModifier{}.PlanModifyString(ctx, planmodifier.StringRequest{
			State:       create,
			ConfigValue: types.StringValue("7.2"),
		}, &response)
		assert.False(t, response.RequiresReplace)
	})

	t.Run("omitted list keeps state", func(t *testing.T) {
		stored, diags := types.ListValueFrom(ctx, types.StringType, []string{"kept"})
		require.False(t, diags.HasError())
		var response planmodifier.ListResponse
		keepStateWhenUnsetList().PlanModifyList(ctx, planmodifier.ListRequest{
			State:       present,
			StateValue:  stored,
			ConfigValue: types.ListNull(types.StringType),
			PlanValue:   types.ListNull(types.StringType),
		}, &response)
		assert.True(t, stored.Equal(response.PlanValue))

		response = planmodifier.ListResponse{}
		configured, diags := types.ListValueFrom(ctx, types.StringType, []string{"configured"})
		require.False(t, diags.HasError())
		keepStateWhenUnsetList().PlanModifyList(ctx, planmodifier.ListRequest{
			State:       present,
			StateValue:  stored,
			ConfigValue: configured,
			PlanValue:   configured,
		}, &response)
		assert.True(t, response.PlanValue.IsNull())

		response = planmodifier.ListResponse{PlanValue: types.ListUnknown(types.StringType)}
		keepStateWhenUnsetList().PlanModifyList(ctx, planmodifier.ListRequest{
			State:       present,
			StateValue:  stored,
			ConfigValue: configured,
			PlanValue:   types.ListUnknown(types.StringType),
		}, &response)
		assert.True(t, response.PlanValue.IsUnknown())

		response = planmodifier.ListResponse{}
		keepStateWhenUnsetList().PlanModifyList(ctx, planmodifier.ListRequest{
			State:       present,
			StateValue:  stored,
			ConfigValue: types.ListNull(types.StringType),
			PlanValue:   types.ListUnknown(types.StringType),
		}, &response)
		assert.True(t, stored.Equal(response.PlanValue))
	})
}

func TestActiveActiveSubscriptionModifyPlan(t *testing.T) {
	ctx := context.Background()
	resourceSchema := subscriptionSchema(t)

	t.Run("creation plan required on create", func(t *testing.T) {
		response := modifySubscriptionPlan(t, resourceSchema, blankSubscriptionModel(t), nil)
		require.True(t, response.Diagnostics.HasError())
		assert.Equal(t, `the "creation_plan" block is required`, response.Diagnostics[0].Summary())
	})

	t.Run("region required on create", func(t *testing.T) {
		model := blankSubscriptionModel(t)
		modules, diags := types.ListValueFrom(ctx, types.StringType, []string{})
		require.False(t, diags.HasError())
		model.CreationPlan, diags = NewCreationPlanListValue(ctx, []CreationPlanModel{{
			Quantity: types.Int64Value(1),
			Modules:  modules,
			Region:   nullCreationPlanRegionSetValue(),
		}})
		require.False(t, diags.HasError())

		response := modifySubscriptionPlan(t, resourceSchema, model, nil)
		require.True(t, response.Diagnostics.HasError())
		assert.Equal(t, `the "creation_plan.region" block is required`, response.Diagnostics[0].Summary())
	})

	t.Run("grace period upgrade diff is suppressed", func(t *testing.T) {
		state := blankSubscriptionModel(t)
		state.ID = types.StringValue("123")
		state.CustomerManagedKeyDeletionGracePeriod = types.StringValue("")
		state.CustomerManagedKeyEnabled = types.BoolValue(false)
		plan := state
		plan.CustomerManagedKeyDeletionGracePeriod = types.StringValue("immediate")

		response := modifySubscriptionPlan(t, resourceSchema, plan, &state)
		require.False(t, response.Diagnostics.HasError())
		var got ActiveActiveSubscriptionResourceModel
		require.False(t, response.Plan.Get(ctx, &got).HasError())
		assert.Equal(t, "", got.CustomerManagedKeyDeletionGracePeriod.ValueString())
	})

	t.Run("enabled customer managed keys are not suppressed", func(t *testing.T) {
		state := blankSubscriptionModel(t)
		state.ID = types.StringValue("123")
		state.CustomerManagedKeyDeletionGracePeriod = types.StringValue("")
		plan := state
		plan.CustomerManagedKeyEnabled = types.BoolValue(true)
		plan.CustomerManagedKeyDeletionGracePeriod = types.StringValue("immediate")

		response := modifySubscriptionPlan(t, resourceSchema, plan, &state)
		require.False(t, response.Diagnostics.HasError())
		var got ActiveActiveSubscriptionResourceModel
		require.False(t, response.Plan.Get(ctx, &got).HasError())
		assert.Equal(t, "immediate", got.CustomerManagedKeyDeletionGracePeriod.ValueString())
	})
}

func TestActiveActiveSubscriptionCreateRequest(t *testing.T) {
	ctx := context.Background()
	model := blankSubscriptionModel(t)
	model.Name = types.StringValue("prod")
	model.PaymentMethod = types.StringValue("marketplace")
	model.PaymentMethodID = types.StringValue("7")
	model.CloudProvider = types.StringValue("AWS")
	model.RedisVersion = types.StringValue("7.2")
	model.CustomerManagedKeyEnabled = types.BoolValue(true)
	model.PublicEndpointAccess = types.BoolValue(false)
	tags, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"zeta": "last", "alpha": "first"})
	require.False(t, diags.HasError())
	model.ResourceTags = tags
	model.CreationPlan = testCreationPlan(t, ctx)

	got, diags := newActiveActiveCreateSubscription(ctx, model)
	require.False(t, diags.HasError())
	assert.Equal(t, subscriptions.SubscriptionDeploymentTypeActiveActive, redis.StringValue(got.DeploymentType))
	assert.False(t, redis.BoolValue(got.DryRun))
	assert.Equal(t, "prod", redis.StringValue(got.Name))
	assert.Equal(t, "marketplace", redis.StringValue(got.PaymentMethod))
	assert.Equal(t, 7, redis.IntValue(got.PaymentMethodID))
	assert.Equal(t, "7.2", redis.StringValue(got.RedisVersion))
	assert.Equal(t, utils.CmkEnabledString, redis.StringValue(got.PersistentStorageEncryptionType))
	assert.False(t, redis.BoolValue(got.PublicEndpointAccess))

	require.Len(t, got.CloudProviders, 1)
	assert.Equal(t, "AWS", redis.StringValue(got.CloudProviders[0].Provider))
	assert.Equal(t, 1, redis.IntValue(got.CloudProviders[0].CloudAccountID))
	require.Len(t, got.CloudProviders[0].ResourceTags, 2)
	assert.Equal(t, "alpha", redis.StringValue(got.CloudProviders[0].ResourceTags[0].Key))
	assert.Equal(t, "zeta", redis.StringValue(got.CloudProviders[0].ResourceTags[1].Key))

	regions := map[string]string{}
	for _, region := range got.CloudProviders[0].Regions {
		regions[redis.StringValue(region.Region)] = redis.StringValue(region.Networking.DeploymentCIDR)
	}
	assert.Equal(t, map[string]string{"us-east-1": "10.0.0.0/24", "eu-west-1": "10.1.0.0/24"}, regions)

	require.Len(t, got.Databases, 2)
	assert.Equal(t, "creation-plan-db-1", redis.StringValue(got.Databases[0].Name))
	assert.Equal(t, "creation-plan-db-2", redis.StringValue(got.Databases[1].Name))
	assert.Equal(t, "redis", redis.StringValue(got.Databases[0].Protocol))
	assert.Nil(t, got.Databases[0].MemoryLimitInGB)
	assert.Equal(t, 1.5, redis.Float64Value(got.Databases[0].DatasetSizeInGB))
	require.Len(t, got.Databases[0].Modules, 1)
	assert.Equal(t, "RedisJSON", redis.StringValue(got.Databases[0].Modules[0].Name))
	throughputs := map[string][2]int{}
	for _, throughput := range got.Databases[0].LocalThroughputMeasurement {
		throughputs[redis.StringValue(throughput.Region)] = [2]int{
			redis.IntValue(throughput.WriteOperationsPerSecond),
			redis.IntValue(throughput.ReadOperationsPerSecond),
		}
	}
	assert.Equal(t, map[string][2]int{"us-east-1": {1000, 2000}, "eu-west-1": {3000, 4000}}, throughputs)

	missing := blankSubscriptionModel(t)
	_, diags = newActiveActiveCreateSubscription(ctx, missing)
	require.True(t, diags.HasError())
	assert.Equal(t, `the "creation_plan" block is required`, diags[0].Summary())
}

func TestSubscriptionPricingFromAPIKeepsOrderAndZeroValues(t *testing.T) {
	ctx := context.Background()
	list, diags := subscriptionPricingFromAPI(ctx, []*pricing.Pricing{
		{Region: redis.String("eu-west-1"), Type: redis.String("Shards"), Quantity: redis.Int(2)},
		{Region: redis.String("us-east-1")},
	})
	require.False(t, diags.HasError())

	var models []utils.PricingModel
	require.False(t, list.ElementsAs(ctx, &models, false).HasError())
	require.Len(t, models, 2)
	assert.Equal(t, "eu-west-1", models[0].Region.ValueString())
	assert.Equal(t, "us-east-1", models[1].Region.ValueString())
	assert.False(t, models[1].DatabaseName.IsNull())
	assert.Equal(t, "", models[1].DatabaseName.ValueString())
	assert.Equal(t, 0.0, models[1].PricePerUnit.ValueFloat64())
	assert.Equal(t, int64(0), models[1].Quantity.ValueInt64())

	empty, diags := subscriptionPricingFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, empty.IsNull())
	assert.Zero(t, len(empty.Elements()))
}

func TestWriteSubscriptionStatePreservesConfigAndOmittedValues(t *testing.T) {
	ctx := context.Background()
	model := blankSubscriptionModel(t)
	model.ID = types.StringValue("123")
	model.PaymentMethodID = types.StringValue("42")
	model.AwsAccountID = types.StringValue("prior-account")
	model.CustomerManagedKeyDeletionGracePeriod = types.StringValue("immediate")
	model.RedisVersion = types.StringValue("7.2")
	model.CreationPlan = testCreationPlan(t, ctx)
	creationPlan := model.CreationPlan

	diags := writeSubscriptionState(ctx, &model, &subscriptions.Subscription{
		Name:   nil,
		Status: redis.String("active"),
		CloudDetails: []*subscriptions.CloudDetail{{
			Provider: redis.String("AWS"),
		}},
	})
	require.False(t, diags.HasError())
	assert.Equal(t, "", model.Name.ValueString())
	assert.False(t, model.Name.IsNull())
	assert.Equal(t, "42", model.PaymentMethodID.ValueString())
	assert.Equal(t, "prior-account", model.AwsAccountID.ValueString())
	assert.Equal(t, "immediate", model.CustomerManagedKeyDeletionGracePeriod.ValueString())
	assert.Equal(t, "AWS", model.CloudProvider.ValueString())
	assert.True(t, model.PublicEndpointAccess.ValueBool())
	assert.False(t, model.CustomerManagedKeyEnabled.ValueBool())
	assert.True(t, model.ResourceTags.IsNull() || len(model.ResourceTags.Elements()) == 0)
	assert.False(t, model.ResourceTags.IsNull())
	assert.Equal(t, "7.2", model.RedisVersion.ValueString())
	assert.True(t, creationPlan.Equal(model.CreationPlan))

	diags = writeSubscriptionState(ctx, &model, &subscriptions.Subscription{
		Status:          redis.String("active"),
		PaymentMethodID: redis.Int(9),
		PaymentMethod:   redis.String("marketplace"),
		CloudDetails: []*subscriptions.CloudDetail{{
			Provider:     redis.String("GCP"),
			AWSAccountID: redis.String("999"),
		}},
		PublicEndpointAccess:            redis.Bool(false),
		PersistentStorageEncryptionType: redis.String(utils.CmkEnabledString),
		DeletionGracePeriod:             redis.String("15-minutes"),
		CustomerManagedKeyAccessDetails: &subscriptions.CustomerManagedKeyAccessDetails{
			RedisServiceAccount: redis.String("svc"),
			AwsRoleArn:          redis.String("arn:role"),
		},
	})
	require.False(t, diags.HasError())
	assert.Equal(t, "9", model.PaymentMethodID.ValueString())
	assert.Equal(t, "marketplace", model.PaymentMethod.ValueString())
	assert.Equal(t, "999", model.AwsAccountID.ValueString())
	assert.Equal(t, "GCP", model.CloudProvider.ValueString())
	assert.False(t, model.PublicEndpointAccess.ValueBool())
	assert.True(t, model.CustomerManagedKeyEnabled.ValueBool())
	assert.Equal(t, "15-minutes", model.CustomerManagedKeyDeletionGracePeriod.ValueString())
	assert.Equal(t, "svc", model.CustomerManagedKeyRedisServiceAccount.ValueString())
	assert.Equal(t, "arn:role", model.CustomerManagedKeyAwsRoleArn.ValueString())

	emptyCloud := blankSubscriptionModel(t)
	diags = writeSubscriptionState(ctx, &emptyCloud, &subscriptions.Subscription{Status: redis.String("pending")})
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "cloud details is empty")
}

func TestMaintenanceAndCustomerManagedKeyMapping(t *testing.T) {
	ctx := context.Background()
	automatic, diags := maintenanceFromModel(ctx, blankSubscriptionModel(t).MaintenanceWindows)
	require.False(t, diags.HasError())
	assert.Equal(t, "automatic", redis.StringValue(automatic.Mode))
	assert.Empty(t, automatic.Windows)

	windows, diags := customtypes.NewMaintenanceList(ctx, &maintenance.Maintenance{
		Mode: redis.String("manual"),
		Windows: []*maintenance.Window{{
			StartHour:       redis.Int(2),
			DurationInHours: redis.Int(4),
			Days:            []*string{redis.String("Monday"), redis.String("Tuesday")},
		}},
	})
	require.False(t, diags.HasError())
	got, diags := maintenanceFromModel(ctx, windows)
	require.False(t, diags.HasError())
	assert.Equal(t, "manual", redis.StringValue(got.Mode))
	require.Len(t, got.Windows, 1)
	assert.Equal(t, 2, redis.IntValue(got.Windows[0].StartHour))
	assert.Equal(t, []string{"Monday", "Tuesday"}, derefStrings(got.Windows[0].Days))

	_, diags = customerManagedKeysFromModel(ctx, nullCustomerManagedKeyListValue())
	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary(), "customer_managed_key must be set")

	keys, diags := NewCustomerManagedKeyListValue(ctx, []CustomerManagedKeyModel{{
		ResourceName: types.StringValue("key"),
		Region:       types.StringValue("us-east1"),
	}})
	require.False(t, diags.HasError())
	cmks, diags := customerManagedKeysFromModel(ctx, keys)
	require.False(t, diags.HasError())
	require.Len(t, cmks, 1)
	assert.Equal(t, "key", redis.StringValue(cmks[0].ResourceName))
}

func TestCIDRValidator(t *testing.T) {
	t.Run("accepts a network", func(t *testing.T) {
		var response validator.StringResponse
		CIDRValidator().ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue("192.168.0.0/24"),
		}, &response)
		assert.False(t, response.Diagnostics.HasError())
	})
	t.Run("rejects text", func(t *testing.T) {
		var response validator.StringResponse
		CIDRValidator().ValidateString(context.Background(), validator.StringRequest{
			ConfigValue: types.StringValue("nope"),
		}, &response)
		assert.True(t, response.Diagnostics.HasError())
	})
}

func TestClearUnknownComputed(t *testing.T) {
	ctx := context.Background()
	model := blankSubscriptionModel(t)
	model.Name = types.StringUnknown()
	model.PaymentMethod = types.StringUnknown()
	model.PaymentMethodID = types.StringUnknown()
	model.CloudProvider = types.StringUnknown()
	model.AwsAccountID = types.StringUnknown()
	model.RedisVersion = types.StringUnknown()
	model.CreationPlan = unknownCreationPlanListValue()
	model.CustomerManagedKey = unknownCustomerManagedKeyListValue()
	model.CustomerManagedKeyRedisServiceAccount = types.StringUnknown()
	model.CustomerManagedKeyAwsRoleArn = types.StringUnknown()
	model.CustomerManagedKeyDeletionGracePeriod = types.StringUnknown()
	model.CustomerManagedKeyEnabled = types.BoolUnknown()
	model.PublicEndpointAccess = types.BoolUnknown()
	model.Pricing = types.ListUnknown(utils.PricingElemType)
	model.ResourceTags = types.MapUnknown(types.StringType)
	model.MaintenanceWindows = unknownMaintenance(t, ctx)

	clearUnknownComputed(ctx, &model)
	assert.Equal(t, "", model.Name.ValueString())
	assert.True(t, model.RedisVersion.IsNull())
	assert.True(t, model.CreationPlan.IsNull())
	assert.True(t, model.MaintenanceWindows.IsNull())
	assert.True(t, model.Pricing.IsNull())
	assert.True(t, model.CustomerManagedKey.IsNull())
	assert.Equal(t, "immediate", model.CustomerManagedKeyDeletionGracePeriod.ValueString())
	assert.False(t, model.CustomerManagedKeyEnabled.ValueBool())
	assert.True(t, model.PublicEndpointAccess.ValueBool())
	assert.False(t, model.ResourceTags.IsNull())
}

func TestSubscriptionResourceMetadataConfigureAndImport(t *testing.T) {
	ctx := context.Background()
	resourceSchema := subscriptionSchema(t)
	impl := &activeActiveSubscriptionResource{}

	var metadata resource.MetadataResponse
	impl.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "rediscloud"}, &metadata)
	assert.Equal(t, "rediscloud_active_active_subscription", metadata.TypeName)

	var configured resource.ConfigureResponse
	impl.Configure(ctx, resource.ConfigureRequest{ProviderData: &client.ApiClient{}}, &configured)
	require.False(t, configured.Diagnostics.HasError())
	require.NotNil(t, impl.client)

	impl.Configure(ctx, resource.ConfigureRequest{ProviderData: "nope"}, &configured)
	assert.True(t, configured.Diagnostics.HasError())

	var imported resource.ImportStateResponse
	imported.State = tfsdk.State{
		Schema: resourceSchema,
		Raw:    tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), nil),
	}
	impl.ImportState(ctx, resource.ImportStateRequest{ID: "123"}, &imported)
	if imported.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %s", imported.Diagnostics)
	}
	var id types.String
	diags := imported.State.GetAttribute(ctx, path.Root("id"), &id)
	require.False(t, diags.HasError())
	assert.Equal(t, "123", id.ValueString())
}

func TestCustomTypeAndModifierDescriptions(t *testing.T) {
	ctx := context.Background()
	assert.NotEmpty(t, NewCreationPlanListType().String())
	assert.NotEmpty(t, NewCreationPlanRegionSetType().String())
	assert.NotEmpty(t, NewCustomerManagedKeyListType().String())
	assert.NotNil(t, NewCreationPlanListType().ValueType(ctx))
	assert.NotNil(t, NewCreationPlanRegionSetType().ValueType(ctx))
	assert.NotNil(t, NewCustomerManagedKeyListType().ValueType(ctx))

	_, diags := NewCreationPlanListType().ValueFromList(ctx, types.ListNull(creationPlanElemType))
	assert.False(t, diags.HasError())
	_, diags = NewCreationPlanRegionSetType().ValueFromSet(ctx, types.SetNull(creationPlanRegionElemType))
	assert.False(t, diags.HasError())
	_, diags = NewCustomerManagedKeyListType().ValueFromList(ctx, types.ListNull(customerManagedKeyElemType))
	assert.False(t, diags.HasError())

	_, err := NewCreationPlanListType().ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "nope"))
	assert.Error(t, err)
	_, err = NewCreationPlanRegionSetType().ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "nope"))
	assert.Error(t, err)
	_, err = NewCustomerManagedKeyListType().ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "nope"))
	assert.Error(t, err)

	assert.False(t, nullCreationPlanListValue().Equal(types.StringNull()))
	assert.False(t, nullCreationPlanRegionSetValue().Equal(types.StringNull()))
	assert.False(t, nullCustomerManagedKeyListValue().Equal(types.StringNull()))

	assert.NotEmpty(t, useStateOnUpdateString().Description(ctx))
	assert.NotEmpty(t, useStateOnUpdateString().MarkdownDescription(ctx))
	assert.NotEmpty(t, (redisVersionModifier{}).Description(ctx))
	assert.NotEmpty(t, (redisVersionModifier{}).MarkdownDescription(ctx))
	assert.NotEmpty(t, keepStateWhenUnsetList().Description(ctx))
	assert.NotEmpty(t, keepStateWhenUnsetList().MarkdownDescription(ctx))
	assert.NotEmpty(t, CIDRValidator().Description(ctx))
	assert.Equal(t, CIDRValidator().Description(ctx), CIDRValidator().MarkdownDescription(ctx))

	var skipped validator.StringResponse
	CIDRValidator().ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringNull()}, &skipped)
	CIDRValidator().ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringUnknown()}, &skipped)
	assert.False(t, skipped.Diagnostics.HasError())

	var response planmodifier.ListResponse
	keepStateWhenUnsetList().PlanModifyList(ctx, planmodifier.ListRequest{
		State: tfsdk.State{Raw: tftypes.NewValue(tftypes.String, nil)},
	}, &response)
	assert.True(t, response.PlanValue.IsNull())

	assert.False(t, maintenanceConfigured(blankSubscriptionModel(t).MaintenanceWindows))
	windows, windowDiags := customtypes.NewMaintenanceList(ctx, &maintenance.Maintenance{Mode: redis.String("manual")})
	require.False(t, windowDiags.HasError())
	assert.True(t, maintenanceConfigured(windows))

	var cleanup diag.Diagnostics
	appendSDKDiagnostics(&cleanup, sdkdiag.Diagnostics{{
		Severity: sdkdiag.Error,
		Summary:  "cleanup failed",
		Detail:   "detail",
	}})
	require.True(t, cleanup.HasError())
	assert.Equal(t, "cleanup failed", cleanup[0].Summary())

	var configured resource.ConfigureResponse
	(&activeActiveSubscriptionResource{}).Configure(ctx, resource.ConfigureRequest{}, &configured)
	assert.False(t, configured.Diagnostics.HasError())
}

func subscriptionSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	(&activeActiveSubscriptionResource{}).Schema(context.Background(), resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())
	return response.Schema
}

func blankSubscriptionModel(t *testing.T) ActiveActiveSubscriptionResourceModel {
	t.Helper()
	ctx := context.Background()
	maintenance, diags := customtypes.NewMaintenanceList(ctx, nil)
	require.False(t, diags.HasError())
	prices, diags := subscriptionPricingFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	tags, diags := utils.ResourceTagsFromAPI(ctx, nil)
	require.False(t, diags.HasError())
	return ActiveActiveSubscriptionResourceModel{
		CreationPlan:       nullCreationPlanListValue(),
		MaintenanceWindows: maintenance,
		Pricing:            prices,
		CustomerManagedKey: nullCustomerManagedKeyListValue(),
		ResourceTags:       tags,
		Timeouts: timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		})},
	}
}

func subscriptionState(t *testing.T, model ActiveActiveSubscriptionResourceModel) tfsdk.State {
	t.Helper()
	state := tfsdk.State{Schema: subscriptionSchema(t)}
	diags := state.Set(context.Background(), &model)
	require.False(t, diags.HasError(), diags)
	return state
}

func modifySubscriptionPlan(t *testing.T, resourceSchema schema.Schema, planModel ActiveActiveSubscriptionResourceModel, stateModel *ActiveActiveSubscriptionResourceModel) resource.ModifyPlanResponse {
	t.Helper()
	ctx := context.Background()
	plan := tfsdk.Plan{Schema: resourceSchema}
	require.False(t, plan.Set(ctx, &planModel).HasError())
	state := tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), nil)}
	if stateModel != nil {
		state = subscriptionState(t, *stateModel)
	}
	request := resource.ModifyPlanRequest{Plan: plan, State: state}
	response := resource.ModifyPlanResponse{Plan: plan}
	(&activeActiveSubscriptionResource{}).ModifyPlan(ctx, request, &response)
	return response
}

func testCreationPlan(t *testing.T, ctx context.Context) CreationPlanListValue {
	t.Helper()
	regions, diags := NewCreationPlanRegionSetValue(ctx, []CreationPlanRegionModel{
		{
			Region:                   types.StringValue("us-east-1"),
			NetworkingDeploymentCIDR: types.StringValue("10.0.0.0/24"),
			WriteOperationsPerSecond: types.Int64Value(1000),
			ReadOperationsPerSecond:  types.Int64Value(2000),
		},
		{
			Region:                   types.StringValue("eu-west-1"),
			NetworkingDeploymentCIDR: types.StringValue("10.1.0.0/24"),
			WriteOperationsPerSecond: types.Int64Value(3000),
			ReadOperationsPerSecond:  types.Int64Value(4000),
		},
	})
	require.False(t, diags.HasError())
	modules, diags := types.ListValueFrom(ctx, types.StringType, []string{"RedisJSON"})
	require.False(t, diags.HasError())
	plan, diags := NewCreationPlanListValue(ctx, []CreationPlanModel{{
		DatasetSizeInGB: types.Float64Value(1.5),
		Quantity:        types.Int64Value(2),
		Modules:         modules,
		Region:          regions,
	}})
	require.False(t, diags.HasError())
	return plan
}

func unknownMaintenance(t *testing.T, ctx context.Context) customtypes.MaintenanceListValue {
	t.Helper()
	typ := customtypes.NewMaintenanceListType()
	value, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), tftypes.UnknownValue))
	require.NoError(t, err)
	unknown, ok := value.(customtypes.MaintenanceListValue)
	require.True(t, ok)
	require.True(t, unknown.IsUnknown())
	return unknown
}

func derefStrings(values []*string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = redis.StringValue(value)
	}
	return out
}
