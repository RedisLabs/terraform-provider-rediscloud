package activeactive

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// subscriptionPricingFromAPI copies pricing into state the way the SDKv2 resource did.
// Absent strings and price_per_unit become zero values, not null, and the API order
// is kept. The data source sorts; sorting here would reorder subscriptions that
// already have pricing in state.
//
// TODO(TF3.0) make pricing strings and price_per_unit nullable and share PricingFromAPI.
func subscriptionPricingFromAPI(ctx context.Context, prices []*pricing.Pricing) (types.List, diag.Diagnostics) {
	elems := make([]utils.PricingModel, 0, len(prices))
	for _, p := range prices {
		elems = append(elems, utils.PricingModel{
			DatabaseName:        types.StringValue(redis.StringValue(p.DatabaseName)),
			Type:                types.StringValue(redis.StringValue(p.Type)),
			TypeDetails:         types.StringValue(redis.StringValue(p.TypeDetails)),
			Quantity:            types.Int64Value(int64(redis.IntValue(p.Quantity))),
			QuantityMeasurement: types.StringValue(redis.StringValue(p.QuantityMeasurement)),
			PricePerUnit:        types.Float64Value(redis.Float64Value(p.PricePerUnit)),
			PriceCurrency:       types.StringValue(redis.StringValue(p.PriceCurrency)),
			PricePeriod:         types.StringValue(redis.StringValue(p.PricePeriod)),
			Region:              types.StringValue(redis.StringValue(p.Region)),
		})
	}
	return types.ListValueFrom(ctx, utils.PricingElemType, elems)
}

func paymentMethodIDPointer(value types.String) (*int, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(value.ValueString())
	if err != nil {
		return nil, err
	}
	return redis.Int(id), nil
}

func resourceTagsToAPI(ctx context.Context, tags types.Map) ([]*subscriptions.ResourceTag, diag.Diagnostics) {
	result := make([]*subscriptions.ResourceTag, 0)
	if tags.IsNull() || tags.IsUnknown() {
		return result, nil
	}

	var values map[string]string
	diags := tags.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, diags
	}
	for _, key := range slices.Sorted(maps.Keys(values)) {
		result = append(result, &subscriptions.ResourceTag{
			Key:   redis.String(key),
			Value: redis.String(values[key]),
		})
	}
	return result, nil
}

func stringPointers(values []string) []*string {
	out := make([]*string, len(values))
	for i, value := range values {
		out[i] = redis.String(value)
	}
	return out
}

func boolOr(value types.Bool, fallback bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return fallback
	}
	return value.ValueBool()
}

func knownString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	return value.ValueString(), true
}

func positiveFloat(value types.Float64) (float64, bool) {
	if value.IsNull() || value.IsUnknown() || value.ValueFloat64() <= 0 {
		return 0, false
	}
	return value.ValueFloat64(), true
}

// newActiveActiveCreateSubscription builds the create call from a planned model.
func newActiveActiveCreateSubscription(ctx context.Context, plan ActiveActiveSubscriptionResourceModel) (subscriptions.CreateSubscription, diag.Diagnostics) {
	var diags diag.Diagnostics
	if plan.CreationPlan.IsNull() || plan.CreationPlan.IsUnknown() || len(plan.CreationPlan.Elements()) == 0 {
		diags.AddError(`the "creation_plan" block is required`, `the "creation_plan" block is required`)
		return subscriptions.CreateSubscription{}, diags
	}

	plans, planDiags := plan.CreationPlan.AsModels(ctx)
	diags.Append(planDiags...)
	if diags.HasError() {
		return subscriptions.CreateSubscription{}, diags
	}

	providers, providerDiags := createCloudProviders(ctx, plan.CloudProvider.ValueString(), plans[0], plan.ResourceTags)
	diags.Append(providerDiags...)
	if diags.HasError() {
		return subscriptions.CreateSubscription{}, diags
	}

	databases, databaseDiags := createPlanDatabases(ctx, plans[0])
	diags.Append(databaseDiags...)
	if diags.HasError() {
		return subscriptions.CreateSubscription{}, diags
	}

	paymentMethodID, err := paymentMethodIDPointer(plan.PaymentMethodID)
	if err != nil {
		diags.AddError(err.Error(), err.Error())
		return subscriptions.CreateSubscription{}, diags
	}

	request := subscriptions.CreateSubscription{
		DeploymentType:       redis.String(subscriptions.SubscriptionDeploymentTypeActiveActive),
		Name:                 redis.String(plan.Name.ValueString()),
		DryRun:               redis.Bool(false),
		PaymentMethodID:      paymentMethodID,
		PaymentMethod:        redis.String(plan.PaymentMethod.ValueString()),
		CloudProviders:       providers,
		Databases:            databases,
		PublicEndpointAccess: redis.Bool(boolOr(plan.PublicEndpointAccess, true)),
	}
	if boolOr(plan.CustomerManagedKeyEnabled, false) {
		request.PersistentStorageEncryptionType = redis.String(utils.CmkEnabledString)
	}
	if version, ok := knownString(plan.RedisVersion); ok && version != "" {
		request.RedisVersion = redis.String(version)
	}
	return request, diags
}

func createCloudProviders(ctx context.Context, providerName string, plan CreationPlanModel, resourceTags types.Map) ([]*subscriptions.CreateCloudProvider, diag.Diagnostics) {
	regions, diags := createRegions(ctx, plan)
	if diags.HasError() {
		return nil, diags
	}
	tags, tagDiags := resourceTagsToAPI(ctx, resourceTags)
	diags.Append(tagDiags...)
	if diags.HasError() {
		return nil, diags
	}

	// Active-Active subscriptions are created on Redis-managed cloud resources.
	return []*subscriptions.CreateCloudProvider{{
		Provider:       redis.String(providerName),
		CloudAccountID: redis.Int(1),
		Regions:        regions,
		ResourceTags:   tags,
	}}, diags
}

func createRegions(ctx context.Context, plan CreationPlanModel) ([]*subscriptions.CreateRegion, diag.Diagnostics) {
	regions := make([]*subscriptions.CreateRegion, 0)
	if plan.Region.IsNull() || plan.Region.IsUnknown() {
		return regions, nil
	}
	models, diags := plan.Region.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}
	for _, region := range models {
		createRegion := &subscriptions.CreateRegion{
			Region: redis.String(region.Region.ValueString()),
		}
		if cidr, ok := knownString(region.NetworkingDeploymentCIDR); ok && cidr != "" {
			createRegion.Networking = &subscriptions.CreateNetworking{
				DeploymentCIDR: redis.String(cidr),
			}
		}
		regions = append(regions, createRegion)
	}
	return regions, nil
}

func createPlanDatabases(ctx context.Context, plan CreationPlanModel) ([]*subscriptions.CreateDatabase, diag.Diagnostics) {
	var diags diag.Diagnostics
	throughputs, throughputDiags := localThroughputs(ctx, plan)
	diags.Append(throughputDiags...)
	if diags.HasError() {
		return nil, diags
	}

	modules := make([]*subscriptions.CreateModules, 0)
	if !plan.Modules.IsNull() && !plan.Modules.IsUnknown() {
		var names []string
		diags.Append(plan.Modules.ElementsAs(ctx, &names, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, name := range names {
			modules = append(modules, &subscriptions.CreateModules{Name: redis.String(name)})
		}
	}

	quantity := int(plan.Quantity.ValueInt64())
	databases := make([]*subscriptions.CreateDatabase, 0, quantity)
	for i := 1; i <= quantity; i++ {
		database := subscriptions.CreateDatabase{
			Name:                       redis.String(fmt.Sprintf("creation-plan-db-%d", i)),
			Protocol:                   redis.String("redis"),
			LocalThroughputMeasurement: throughputs,
			Quantity:                   redis.Int(1),
			Modules:                    modules,
		}
		if size, ok := positiveFloat(plan.DatasetSizeInGB); ok {
			database.DatasetSizeInGB = redis.Float64(size)
		}
		if size, ok := positiveFloat(plan.MemoryLimitInGB); ok {
			database.MemoryLimitInGB = redis.Float64(size)
		}
		databases = append(databases, &database)
	}
	return databases, diags
}

func localThroughputs(ctx context.Context, plan CreationPlanModel) ([]*subscriptions.CreateLocalThroughput, diag.Diagnostics) {
	throughputs := make([]*subscriptions.CreateLocalThroughput, 0)
	if plan.Region.IsNull() || plan.Region.IsUnknown() {
		return throughputs, nil
	}
	regions, diags := plan.Region.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}
	for _, region := range regions {
		throughputs = append(throughputs, &subscriptions.CreateLocalThroughput{
			Region:                   redis.String(region.Region.ValueString()),
			WriteOperationsPerSecond: redis.Int(int(region.WriteOperationsPerSecond.ValueInt64())),
			ReadOperationsPerSecond:  redis.Int(int(region.ReadOperationsPerSecond.ValueInt64())),
		})
	}
	return throughputs, nil
}

func customerManagedKeysFromModel(ctx context.Context, keys CustomerManagedKeyListValue) ([]subscriptions.CustomerManagedKey, diag.Diagnostics) {
	var diags diag.Diagnostics
	if keys.IsNull() || keys.IsUnknown() || len(keys.Elements()) == 0 {
		diags.AddError(
			"customer_managed_key must be set when subscription is in encryption key pending state",
			"customer_managed_key must be set when subscription is in encryption key pending state",
		)
		return nil, diags
	}
	models, modelDiags := keys.AsModels(ctx)
	diags.Append(modelDiags...)
	if diags.HasError() {
		return nil, diags
	}
	cmks := make([]subscriptions.CustomerManagedKey, 0, len(models))
	for _, model := range models {
		cmks = append(cmks, subscriptions.CustomerManagedKey{
			ResourceName: redis.String(model.ResourceName.ValueString()),
			Region:       redis.String(model.Region.ValueString()),
		})
	}
	return cmks, diags
}

func maintenanceFromModel(ctx context.Context, windows customtypes.MaintenanceListValue) (maintenance.Maintenance, diag.Diagnostics) {
	if windows.IsNull() || windows.IsUnknown() || len(windows.Elements()) == 0 {
		return maintenance.Maintenance{Mode: redis.String("automatic")}, nil
	}

	models, diags := windows.AsModels(ctx)
	if diags.HasError() {
		return maintenance.Maintenance{}, diags
	}
	windowModels := make([]customtypes.MaintenanceWindowModel, 0)
	if !models[0].Window.IsNull() && !models[0].Window.IsUnknown() {
		var windowDiags diag.Diagnostics
		windowModels, windowDiags = models[0].Window.AsModels(ctx)
		diags.Append(windowDiags...)
		if diags.HasError() {
			return maintenance.Maintenance{}, diags
		}
	}

	apiWindows := make([]*maintenance.Window, 0, len(windowModels))
	for _, window := range windowModels {
		var days []string
		if !window.Days.IsNull() && !window.Days.IsUnknown() {
			diags.Append(window.Days.ElementsAs(ctx, &days, false)...)
			if diags.HasError() {
				return maintenance.Maintenance{}, diags
			}
		}
		apiWindows = append(apiWindows, &maintenance.Window{
			StartHour:       redis.Int(int(window.StartHour.ValueInt64())),
			DurationInHours: redis.Int(int(window.DurationInHours.ValueInt64())),
			Days:            stringPointers(days),
		})
	}
	return maintenance.Maintenance{
		Mode:    redis.String(models[0].Mode.ValueString()),
		Windows: apiWindows,
	}, diags
}

// writeSubscriptionState copies API subscription fields onto the model.
// Configuration-only values (creation plan, customer-managed keys, redis version)
// are left untouched. Absent computed strings stay empty when state does not
// already hold a value, matching the SDKv2 read.
func writeSubscriptionState(ctx context.Context, model *ActiveActiveSubscriptionResourceModel, subscription *subscriptions.Subscription) diag.Diagnostics {
	var diags diag.Diagnostics

	//TODO(TF3.0) make name nullable
	model.Name = types.StringValue(redis.StringValue(subscription.Name))

	if subscription.PaymentMethodID != nil && redis.IntValue(subscription.PaymentMethodID) != 0 {
		model.PaymentMethodID = types.StringValue(strconv.Itoa(redis.IntValue(subscription.PaymentMethodID)))
	} else if model.PaymentMethodID.IsNull() || model.PaymentMethodID.IsUnknown() {
		//TODO(TF3.0) make payment_method_id nullable
		model.PaymentMethodID = types.StringValue("")
	}
	//TODO(TF3.0) make payment_method nullable
	model.PaymentMethod = types.StringValue(redis.StringValue(subscription.PaymentMethod))

	if len(subscription.CloudDetails) == 0 || subscription.CloudDetails[0] == nil {
		message := fmt.Sprintf("cloud details is empty. Subscription status: %s", redis.StringValue(subscription.Status))
		diags.AddError(message, message)
		return diags
	}
	detail := subscription.CloudDetails[0]
	//TODO(TF3.0) make cloud_provider nullable
	model.CloudProvider = types.StringValue(redis.StringValue(detail.Provider))
	if detail.AWSAccountID != nil {
		model.AwsAccountID = types.StringValue(redis.StringValue(detail.AWSAccountID))
	} else if model.AwsAccountID.IsNull() || model.AwsAccountID.IsUnknown() {
		//TODO(TF3.0) make aws_account_id nullable
		model.AwsAccountID = types.StringValue("")
	}

	tags, tagDiags := utils.ResourceTagsFromAPI(ctx, detail.ResourceTags)
	diags.Append(tagDiags...)
	if diags.HasError() {
		return diags
	}
	model.ResourceTags = tags

	var serviceAccount, roleArn *string
	if subscription.CustomerManagedKeyAccessDetails != nil {
		serviceAccount = subscription.CustomerManagedKeyAccessDetails.RedisServiceAccount
		roleArn = subscription.CustomerManagedKeyAccessDetails.AwsRoleArn
	}
	if serviceAccount != nil {
		model.CustomerManagedKeyRedisServiceAccount = types.StringValue(redis.StringValue(serviceAccount))
	} else if model.CustomerManagedKeyRedisServiceAccount.IsNull() || model.CustomerManagedKeyRedisServiceAccount.IsUnknown() {
		//TODO(TF3.0) make customer_managed_key_redis_service_account nullable
		model.CustomerManagedKeyRedisServiceAccount = types.StringValue("")
	}
	if roleArn != nil {
		model.CustomerManagedKeyAwsRoleArn = types.StringValue(redis.StringValue(roleArn))
	} else if model.CustomerManagedKeyAwsRoleArn.IsNull() || model.CustomerManagedKeyAwsRoleArn.IsUnknown() {
		//TODO(TF3.0) make customer_managed_key_aws_role_arn nullable
		model.CustomerManagedKeyAwsRoleArn = types.StringValue("")
	}

	publicEndpoint := true
	if subscription.PublicEndpointAccess != nil {
		publicEndpoint = redis.BoolValue(subscription.PublicEndpointAccess)
	}
	model.PublicEndpointAccess = types.BoolValue(publicEndpoint)

	model.CustomerManagedKeyEnabled = types.BoolValue(subscription.PersistentStorageEncryptionType != nil &&
		redis.StringValue(subscription.PersistentStorageEncryptionType) == utils.CmkEnabledString)

	if subscription.DeletionGracePeriod != nil {
		model.CustomerManagedKeyDeletionGracePeriod = types.StringValue(redis.StringValue(subscription.DeletionGracePeriod))
	} else if model.CustomerManagedKeyDeletionGracePeriod.IsNull() || model.CustomerManagedKeyDeletionGracePeriod.IsUnknown() {
		//TODO(TF3.0) make customer_managed_key_deletion_grace_period nullable
		model.CustomerManagedKeyDeletionGracePeriod = types.StringValue("")
	}

	return diags
}

func writeActiveCollections(ctx context.Context, model *ActiveActiveSubscriptionResourceModel, windows *maintenance.Maintenance, prices []*pricing.Pricing) diag.Diagnostics {
	var diags diag.Diagnostics
	maintenanceWindows, maintenanceDiags := customtypes.NewMaintenanceList(ctx, windows)
	diags.Append(maintenanceDiags...)
	if diags.HasError() {
		return diags
	}
	model.MaintenanceWindows = maintenanceWindows

	pricingList, pricingDiags := subscriptionPricingFromAPI(ctx, prices)
	diags.Append(pricingDiags...)
	if diags.HasError() {
		return diags
	}
	model.Pricing = pricingList
	return diags
}

func mapsEffectivelyEqual(left, right types.Map) bool {
	if left.Equal(right) {
		return true
	}
	return mapEmpty(left) && mapEmpty(right)
}

func mapEmpty(value types.Map) bool {
	return value.IsNull() || (!value.IsUnknown() && len(value.Elements()) == 0)
}
