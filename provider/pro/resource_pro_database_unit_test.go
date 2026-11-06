package pro

import (
	"context"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

func TestProDatabaseSchemaMatchesModel(t *testing.T) {
	ctx := context.Background()
	proResource := NewProDatabaseResource()
	var response resource.SchemaResponse
	proResource.Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())

	model := ProDatabaseModel{
		ClientTLSCertificates: types.ListNull(types.StringType),
		ReplicaOf:             types.SetNull(types.StringType),
		SourceIPs:             types.SetNull(types.StringType),
		HashingPolicy:         types.ListNull(types.StringType),
		Tags:                  types.MapNull(types.StringType),
		Timeouts: timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		})},
	}
	modelType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(model)}
	assert.Equal(t, modelType, response.Schema.Type())

	alertBlock := response.Schema.Blocks["alert"].(schema.SetNestedBlock)
	assert.True(t, alertBlock.CustomType.(AlertSetType).ElementType().Equal(alertBlock.NestedObject.Type()))
	assert.Empty(t, alertBlock.NestedObject.Blocks)

	modulesAttribute := response.Schema.Attributes["modules"].(schema.SetAttribute)
	assert.True(t, modulesAttribute.CustomType.(ModuleSetType).ElementType().Equal(moduleElementType))
	_, modulesIsBlock := response.Schema.Blocks["modules"]
	assert.False(t, modulesIsBlock)

	backupBlock := response.Schema.Blocks["remote_backup"].(schema.ListNestedBlock)
	assert.True(t, backupBlock.CustomType.(RemoteBackupListType).ElementType().Equal(backupBlock.NestedObject.Type()))
	assert.Empty(t, backupBlock.NestedObject.Blocks)

	protocol := response.Schema.Attributes["protocol"].(schema.StringAttribute)
	assert.False(t, protocol.PlanModifiers == nil)
	subscription := response.Schema.Attributes["subscription_id"].(schema.Int64Attribute)
	assert.NotEmpty(t, subscription.PlanModifiers)
}

func TestAlertAndModuleAndBackupTypes(t *testing.T) {
	ctx := context.Background()

	alerts, diags := NewAlertSet(ctx, []AlertModel{{
		Name:  types.StringValue("dataset-size"),
		Value: types.Int64Value(80),
	}})
	require.False(t, diags.HasError())
	assert.True(t, NewAlertSetType().Equal(NewAlertSetType()))
	assert.False(t, NewAlertSetType().Equal(types.StringType))
	assert.True(t, alerts.Equal(alerts))
	assert.False(t, alerts.Equal(types.StringValue("nope")))

	tfValue, err := alerts.ToTerraformValue(ctx)
	require.NoError(t, err)
	converted, err := NewAlertSetType().ValueFromTerraform(ctx, tfValue)
	require.NoError(t, err)
	roundTrip := converted.(AlertSetValue)
	models, modelDiags := roundTrip.AsModels(ctx)
	require.False(t, modelDiags.HasError())
	require.Len(t, models, 1)
	assert.Equal(t, "dataset-size", models[0].Name.ValueString())
	assert.EqualValues(t, 80, models[0].Value.ValueInt64())

	nullAlerts := nullAlertSet()
	assert.True(t, nullAlerts.IsNull())
	nullTF, err := nullAlerts.ToTerraformValue(ctx)
	require.NoError(t, err)
	assert.True(t, nullTF.IsNull())
	unknownAlerts := AlertSetValue{SetValue: types.SetUnknown(alertElementType)}
	assert.True(t, unknownAlerts.IsUnknown())
	_, err = NewAlertSetType().ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "bad"))
	require.Error(t, err)

	emptyAlerts, diags := NewAlertSet(ctx, nil)
	require.False(t, diags.HasError())
	assert.False(t, emptyAlerts.IsNull())
	assert.Zero(t, len(emptyAlerts.Elements()))

	modules, diags := NewModuleSet(ctx, nil)
	require.False(t, diags.HasError())
	assert.True(t, NewModuleSetType().Equal(modules.Type(ctx)))
	assert.False(t, NewModuleSetType().Equal(types.StringType))
	nullModules := nullModuleSet()
	nullModuleTF, err := nullModules.ToTerraformValue(ctx)
	require.NoError(t, err)
	assert.True(t, nullModuleTF.IsNull())
	_, err = NewModuleSetType().ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "bad"))
	require.Error(t, err)

	backups, diags := NewRemoteBackupList(ctx, []RemoteBackupModel{{
		Interval:    types.StringValue("every-24-hours"),
		TimeUTC:     types.StringValue(""),
		StorageType: types.StringValue("s3"),
		StoragePath: types.StringValue("s3://bucket"),
	}})
	require.False(t, diags.HasError())
	assert.True(t, NewRemoteBackupListType().Equal(backups.Type(ctx)))
	assert.False(t, backups.Equal(types.StringNull()))
	backupModels, backupDiags := backups.AsModels(ctx)
	require.False(t, backupDiags.HasError())
	require.Len(t, backupModels, 1)
	assert.Equal(t, "", backupModels[0].TimeUTC.ValueString())
	nullBackups := nullRemoteBackupList()
	nullBackupTF, err := nullBackups.ToTerraformValue(ctx)
	require.NoError(t, err)
	assert.True(t, nullBackupTF.IsNull())
	_, err = NewRemoteBackupListType().ValueFromTerraform(ctx, tftypes.NewValue(tftypes.String, "bad"))
	require.Error(t, err)
	emptyBackups, diags := emptyRemoteBackupList(ctx)
	require.False(t, diags.HasError())
	assert.Zero(t, len(emptyBackups.Elements()))
}

func TestToDatabaseId(t *testing.T) {
	subID, dbID, err := ToDatabaseId("15/42")
	require.NoError(t, err)
	assert.Equal(t, 15, subID)
	assert.Equal(t, 42, dbID)

	subID, dbID, err = ToDatabaseId("7")
	require.NoError(t, err)
	assert.Equal(t, 0, subID)
	assert.Equal(t, 7, dbID)

	_, _, err = ToDatabaseId("1/2/3")
	require.Error(t, err)
	_, _, err = ToDatabaseId("nope")
	require.Error(t, err)
	_, _, err = ToDatabaseId("1/nope")
	require.Error(t, err)
}

func TestSourceIPDefaults(t *testing.T) {
	assert.True(t, isDefaultSourceIPStrings(nil))
	assert.True(t, isDefaultSourceIPStrings([]string{"0.0.0.0/0"}))
	assert.True(t, isDefaultSourceIPStrings([]string{"192.168.0.0/16", "10.0.0.0/8", "100.64.0.0/10", "172.16.0.0/12"}))
	assert.False(t, isDefaultSourceIPStrings([]string{"203.0.113.0/24"}))

	ips, needsDefault := resolveReadSourceIPs([]string{"0.0.0.0/0"}, []string{"203.0.113.0/24"})
	assert.False(t, needsDefault)
	assert.Equal(t, []string{"203.0.113.0/24"}, ips)

	_, needsDefault = resolveReadSourceIPs(nil, nil)
	assert.True(t, needsDefault)
	assert.Equal(t, []string{"0.0.0.0/0"}, defaultSourceIPsForAccess(nil))
	assert.Equal(t, defaultPrivateIPRanges, defaultSourceIPsForAccess(redis.Bool(false)))
}

func TestBackupAndHashingAndCertificates(t *testing.T) {
	ctx := context.Background()
	assert.True(t, backupTimesEquivalent("every-12-hours", "02:15", "14:15"))
	assert.False(t, backupTimesEquivalent("every-24-hours", "02:15", "14:15"))
	assert.False(t, backupTimesEquivalent("every-12-hours", "bad", "14:15"))
	assert.Contains(t, remoteBackupIntervalError(), "unexpected value at remote_backup.0.time_utc")

	hashing, diags := HashingPolicyFromAPI(ctx, []*databases.RegexRule{
		{Ordinal: 0, Pattern: ".*\\{(?<tag>.*)\\}.*"},
		{Ordinal: 1, Pattern: "(?<tag>.*)"},
	})
	require.False(t, diags.HasError())
	assert.Zero(t, len(hashing.Elements()))

	custom, diags := HashingPolicyFromAPI(ctx, []*databases.RegexRule{{Ordinal: 0, Pattern: "custom"}})
	require.False(t, diags.HasError())
	var patterns []string
	require.False(t, custom.ElementsAs(ctx, &patterns, false).HasError())
	assert.Equal(t, []string{"custom"}, patterns)

	absent, diags := RemoteBackupFromAPI(ctx, nil, "", nullRemoteBackupList())
	require.False(t, diags.HasError())
	assert.True(t, absent.IsNull())

	prior, diags := NewRemoteBackupList(ctx, []RemoteBackupModel{{
		Interval:    types.StringValue("every-12-hours"),
		TimeUTC:     types.StringValue("14:00"),
		StorageType: types.StringValue("s3"),
		StoragePath: types.StringValue("s3://bucket"),
	}})
	require.False(t, diags.HasError())
	shifted, diags := RemoteBackupFromAPI(ctx, &databases.Backup{
		Enabled:     redis.Bool(true),
		Interval:    redis.String("every-12-hours"),
		TimeUTC:     redis.String("02:00"),
		Destination: redis.String("s3://bucket"),
	}, "", prior)
	require.False(t, diags.HasError())
	shiftedModels, modelDiags := shifted.AsModels(ctx)
	require.False(t, modelDiags.HasError())
	assert.Equal(t, "14:00", shiftedModels[0].TimeUTC.ValueString())
	assert.Equal(t, "s3", shiftedModels[0].StorageType.ValueString())

	disabled, diags := RemoteBackupFromAPI(ctx, &databases.Backup{Enabled: redis.Bool(false)}, "", prior)
	require.False(t, diags.HasError())
	assert.Zero(t, len(disabled.Elements()))

	config, diags := backupConfigForRequest(ctx, nullRemoteBackupList(), types.StringValue(""))
	require.False(t, diags.HasError())
	assert.Nil(t, config)
	config, diags = backupConfigForRequest(ctx, nullRemoteBackupList(), types.StringNull())
	require.False(t, diags.HasError())
	require.NotNil(t, config)
	assert.False(t, redis.BoolValue(config.Active))
	config, diags = backupConfigForRequest(ctx, prior, types.StringValue(""))
	require.False(t, diags.HasError())
	assert.Equal(t, "14:00", redis.StringValue(config.TimeUTC))

	certs, diags := applyCertificateHints(true, types.StringValue(""), types.ListNull(types.StringType))
	require.False(t, diags.HasError())
	var hint []string
	require.False(t, certs.tlsCertificates.ElementsAs(ctx, &hint, false).HasError())
	assert.Equal(t, []string{"Unknown certificate"}, hint)

	cleared, diags := applyCertificateHints(false, types.StringValue("cert"), types.ListNull(types.StringType))
	require.False(t, diags.HasError())
	assert.Equal(t, "", cleared.sslCertificate.ValueString())
	assert.Zero(t, len(cleared.tlsCertificates.Elements()))
}

func TestAdjustProDatabasePlan(t *testing.T) {
	ctx := context.Background()
	stateModules, diags := NewModuleSet(ctx, []ModuleModel{{Name: types.StringValue("RedisBloom")}})
	require.False(t, diags.HasError())
	configModules, diags := NewModuleSet(ctx, []ModuleModel{{Name: types.StringValue("RedisJSON")}})
	require.False(t, diags.HasError())

	state := ProDatabaseModel{
		RedisVersion:           types.StringValue("8.4"),
		RedisVersionActual:     types.StringValue("8.6"),
		Modules:                stateModules,
		QueryPerformanceFactor: types.StringValue("Standard"),
		Password:               types.StringValue("secret"),
		RemoteBackup:           nullRemoteBackupList(),
	}
	config := state
	config.RedisVersion = types.StringValue("8.2")
	config.Modules = configModules
	config.QueryPerformanceFactor = types.StringValue("standard")
	plan := config

	adjusted := adjustProDatabasePlan(ctx, config, plan, state, true)
	require.False(t, adjusted.Diagnostics.HasError())
	assert.Equal(t, "8.4", adjusted.Plan.RedisVersion.ValueString())
	assert.True(t, adjusted.Plan.Modules.Equal(stateModules))
	assert.Empty(t, adjusted.RequiresReplace)
	assert.Equal(t, "Standard", adjusted.Plan.QueryPerformanceFactor.ValueString())

	state.RedisVersion = types.StringValue("7.2")
	state.RedisVersionActual = types.StringValue("7.2")
	config.RedisVersion = types.StringValue("7.4")
	plan = config
	plan.Modules = configModules
	adjusted = adjustProDatabasePlan(ctx, config, plan, state, true)
	require.False(t, adjusted.Diagnostics.HasError())
	assert.Equal(t, "7.4", adjusted.Plan.RedisVersion.ValueString())
	require.Len(t, adjusted.RequiresReplace, 1)
	assert.True(t, adjusted.RequiresReplace[0].Equal(path.Root("modules")))

	config.EnablePasswordless = types.BoolValue(true)
	config.Password = types.StringValue("secret")
	plan = config
	adjusted = adjustProDatabasePlan(ctx, config, plan, state, true)
	assert.True(t, adjusted.Diagnostics.HasError())

	config.Password = types.StringNull()
	config.EnablePasswordless = types.BoolValue(false)
	config.RemoteBackup, diags = NewRemoteBackupList(ctx, []RemoteBackupModel{{
		Interval:    types.StringValue("every-6-hours"),
		TimeUTC:     types.StringValue("16:00"),
		StorageType: types.StringValue("s3"),
		StoragePath: types.StringValue("s3://bucket"),
	}})
	require.False(t, diags.HasError())
	plan = config
	adjusted = adjustProDatabasePlan(ctx, config, plan, state, true)
	assert.True(t, adjusted.Diagnostics.HasError())
	assert.Contains(t, adjusted.Diagnostics.Errors()[0].Detail(), "remote_backup.0.time_utc")

	assert.True(t, shouldUpgradeRedisVersion(types.StringValue("7.2"), types.StringValue("7.4"), types.StringValue("7.2")))
	assert.False(t, shouldUpgradeRedisVersion(types.StringValue("7.2"), types.StringValue("7.4"), types.StringValue("7.4")))
	assert.False(t, shouldUpgradeRedisVersion(types.StringValue(""), types.StringValue("7.4"), types.StringValue("7.2")))
	assert.Contains(t, observedStatusDetail(redis.String("active"), assert.AnError), "last observed status active:")
}

func TestApplyProDatabaseReadPreservesContracts(t *testing.T) {
	ctx := context.Background()
	modules, diags := NewModuleSet(ctx, []ModuleModel{{Name: types.StringValue("RedisBloom")}})
	require.False(t, diags.HasError())
	prior := ProDatabaseModel{
		RedisVersion:                     types.StringValue("8.2"),
		MemoryLimitInGB:                  types.Float64Null(),
		DatasetSizeInGB:                  types.Float64Value(1),
		Password:                         types.StringValue("configured"),
		AverageItemSizeInBytes:           types.Int64Value(0),
		ExternalEndpointForOSSClusterAPI: types.BoolValue(false),
		Modules:                          modules,
		Alert:                            nullAlertSet(),
		SourceIPs:                        types.SetNull(types.StringType),
		ClientSSLCertificate:             types.StringValue(""),
		ClientTLSCertificates:            types.ListNull(types.StringType),
		PeriodicBackupPath:               types.StringValue(""),
		RemoteBackup:                     nullRemoteBackupList(),
		Tags:                             types.MapNull(types.StringType),
		ReplicaOf:                        types.SetNull(types.StringType),
		QueryPerformanceFactor:           types.StringNull(),
		RamPercentage:                    types.Int64Null(),
	}
	db := &databases.Database{
		ID:                    redis.Int(9),
		Name:                  redis.String("example"),
		Protocol:              redis.String("redis"),
		RedisVersion:          redis.String("8.6"),
		RespVersion:           redis.String("resp3"),
		DataPersistence:       redis.String("none"),
		DataEvictionPolicy:    redis.String("volatile-lru"),
		Replication:           redis.Bool(true),
		SupportOSSClusterAPI:  redis.Bool(false),
		ThroughputMeasurement: &databases.Throughput{By: redis.String("operations-per-second"), Value: redis.Int(1000)},
		DatasetSizeInGB:       redis.Float64(1),
		MemoryLimitInGB:       redis.Float64(2),
		PublicEndpoint:        redis.String(""),
		PrivateEndpoint:       redis.String("redis://private"),
		Modules:               []*databases.Module{{Name: redis.String("RediSearch")}},
		Security: &databases.Security{
			Password:          redis.String(""),
			SourceIPs:         []*string{redis.String("0.0.0.0/0")},
			EnableTls:         redis.Bool(false),
			EnableDefaultUser: redis.Bool(true),
		},
		Clustering: &databases.Clustering{RegexRules: []*databases.RegexRule{
			{Ordinal: 0, Pattern: ".*\\{(?<tag>.*)\\}.*"},
			{Ordinal: 1, Pattern: "(?<tag>.*)"},
		}},
	}

	updated, diags := applyProDatabaseRead(ctx, prior, db, redis.Bool(true), map[string]string{"market": "emea"}, true)
	require.False(t, diags.HasError())
	assert.Equal(t, "8.2", updated.RedisVersion.ValueString())
	assert.Equal(t, "8.6", updated.RedisVersionActual.ValueString())
	readModules, moduleDiags := updated.Modules.AsModels(ctx)
	require.False(t, moduleDiags.HasError())
	require.Len(t, readModules, 1)
	assert.Equal(t, "RediSearch", readModules[0].Name.ValueString())
	assert.Equal(t, "", updated.Password.ValueString())
	assert.True(t, updated.EnablePasswordless.ValueBool())
	assert.True(t, updated.MemoryLimitInGB.IsNull())
	assert.Equal(t, 1.0, updated.DatasetSizeInGB.ValueFloat64())
	assert.Equal(t, "", updated.PublicEndpoint.ValueString())
	assert.Equal(t, int64(0), updated.AverageItemSizeInBytes.ValueInt64())
	assert.True(t, updated.ClientTLSCertificates.IsNull())
	assert.Zero(t, len(updated.HashingPolicy.Elements()))
	var source []string
	require.False(t, updated.SourceIPs.ElementsAs(ctx, &source, false).HasError())
	assert.Equal(t, []string{"0.0.0.0/0"}, source)
	assert.Equal(t, "emea", mapValue(t, updated.Tags, "market"))

	prior.RedisVersion = types.StringNull()
	prior.Modules = nullModuleSet()
	prior.DatasetSizeInGB = types.Float64Null()
	db.RedisVersion = redis.String("7.4")
	db.Modules = nil
	db.Security.Password = redis.String("generated")
	updated, diags = applyProDatabaseRead(ctx, prior, db, nil, nil, false)
	require.False(t, diags.HasError())
	assert.Equal(t, "7.4", updated.RedisVersion.ValueString())
	assert.Equal(t, "generated", updated.Password.ValueString())
	assert.False(t, updated.EnablePasswordless.ValueBool())
	assert.False(t, updated.Modules.IsNull())
}

func TestProDatabaseCRUDWithFakeAPI(t *testing.T) {
	ctx := context.Background()
	fake := &fakeProDatabaseAPI{nextID: 42, publicAccess: redis.Bool(true)}
	fake.db = &databases.Database{
		ID:                    redis.Int(42),
		Name:                  redis.String("example"),
		Protocol:              redis.String("redis"),
		RedisVersion:          redis.String("7.4"),
		RespVersion:           redis.String("resp3"),
		DataPersistence:       redis.String("none"),
		DataEvictionPolicy:    redis.String("volatile-lru"),
		Replication:           redis.Bool(true),
		ThroughputMeasurement: &databases.Throughput{By: redis.String("operations-per-second"), Value: redis.Int(1000)},
		DatasetSizeInGB:       redis.Float64(1),
		Security: &databases.Security{
			Password:          redis.String("generated"),
			SourceIPs:         []*string{redis.String("0.0.0.0/0")},
			EnableTls:         redis.Bool(false),
			EnableDefaultUser: redis.Bool(true),
		},
		Status: redis.String("active"),
	}
	resourceUnderTest := &proDatabaseResource{
		api:                       fake,
		waitForSubscriptionActive: func(context.Context, int, time.Duration) error { return nil },
		waitForDatabaseActive:     func(context.Context, int, int, time.Duration) error { return nil },
	}
	plan := testDatabasePlan()
	var diags diag.Diagnostics
	resourceUnderTest.createDatabase(ctx, &plan, &diags, time.Second)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "10/42", plan.ID.ValueString())
	assert.Equal(t, int64(42), plan.DbID.ValueInt64())
	assert.Equal(t, "generated", plan.Password.ValueString())
	require.NotNil(t, fake.updated.SourceIP)
	assert.Equal(t, "0.0.0.0/0", redis.StringValue(fake.updated.SourceIP[0]))
	assert.Equal(t, "example", redis.StringValue(fake.created.Name))
	assert.Empty(t, *fake.updated.Alerts)

	fake.publicAccess = redis.Bool(false)
	plan = testDatabasePlan()
	plan.ID = types.StringValue("10/42")
	plan.DbID = types.Int64Value(42)
	prior := plan
	prior.RedisVersion = types.StringValue("7.2")
	prior.RedisVersionActual = types.StringValue("7.2")
	plan.RedisVersion = types.StringValue("7.4")
	diags = nil
	resourceUnderTest.updateDatabase(ctx, &plan, &prior, &diags, time.Second)
	require.False(t, diags.HasError(), diags)
	assert.Equal(t, "7.4", fake.upgraded)
	require.NotEmpty(t, fake.updated.SourceIP)
	assert.Equal(t, "10.0.0.0/8", redis.StringValue(fake.updated.SourceIP[0]))

	prior.EnablePasswordless = types.BoolValue(true)
	plan.EnablePasswordless = types.BoolValue(false)
	plan.Password = types.StringValue("")
	diags = nil
	resourceUnderTest.updateDatabase(ctx, &plan, &prior, &diags, time.Second)
	assert.True(t, diags.HasError())
	assert.Contains(t, diags.Errors()[0].Detail(), "when disabling passwordless mode")

	missing := &fakeProDatabaseAPI{getErr: &databases.NotFound{}}
	reader := &proDatabaseResource{api: missing, waitForSubscriptionActive: resourceUnderTest.waitForSubscriptionActive, waitForDatabaseActive: resourceUnderTest.waitForDatabaseActive}
	state := testDatabasePlan()
	state.ID = types.StringValue("10/42")
	diags = nil
	removed := reader.readDatabase(ctx, &state, &diags, time.Second)
	require.False(t, diags.HasError())
	assert.True(t, removed)

	deleter := &proDatabaseResource{api: missing, waitForSubscriptionActive: resourceUnderTest.waitForSubscriptionActive, waitForDatabaseActive: resourceUnderTest.waitForDatabaseActive}
	diags = nil
	deleter.deleteDatabase(ctx, &state, &diags, time.Second)
	assert.True(t, diags.HasError())

	failWait := &fakeProDatabaseAPI{nextID: 7, db: &databases.Database{Status: redis.String("pending")}}
	failing := &proDatabaseResource{
		api:                       failWait,
		waitForSubscriptionActive: func(context.Context, int, time.Duration) error { return nil },
		waitForDatabaseActive:     func(context.Context, int, int, time.Duration) error { return assert.AnError },
	}
	failedPlan := testDatabasePlan()
	diags = nil
	failing.createDatabase(ctx, &failedPlan, &diags, time.Second)
	assert.True(t, diags.HasError())
	assert.Equal(t, "10/7", failedPlan.ID.ValueString())
	assert.Contains(t, diags.Errors()[0].Detail(), "last observed status")
}

func testDatabasePlan() ProDatabaseModel {
	return ProDatabaseModel{
		SubscriptionID:                   types.Int64Value(10),
		Name:                             types.StringValue("example"),
		Protocol:                         types.StringValue("redis"),
		DatasetSizeInGB:                  types.Float64Value(1),
		MemoryLimitInGB:                  types.Float64Null(),
		SupportOSSClusterAPI:             types.BoolValue(false),
		DataPersistence:                  types.StringValue("none"),
		DataEviction:                     types.StringValue("volatile-lru"),
		Replication:                      types.BoolValue(true),
		ThroughputMeasurementBy:          types.StringValue("operations-per-second"),
		ThroughputMeasurementValue:       types.Int64Value(1000),
		AverageItemSizeInBytes:           types.Int64Value(0),
		Password:                         types.StringNull(),
		EnablePasswordless:               types.BoolValue(false),
		ExternalEndpointForOSSClusterAPI: types.BoolValue(false),
		EnableTLS:                        types.BoolValue(false),
		EnableDefaultUser:                types.BoolValue(true),
		AutoMinorVersionUpgrade:          types.BoolValue(true),
		ClientSSLCertificate:             types.StringValue(""),
		ClientTLSCertificates:            types.ListNull(types.StringType),
		PeriodicBackupPath:               types.StringValue(""),
		ReplicaOf:                        types.SetNull(types.StringType),
		Alert:                            nullAlertSet(),
		Modules:                          nullModuleSet(),
		SourceIPs:                        types.SetNull(types.StringType),
		HashingPolicy:                    types.ListNull(types.StringType),
		RemoteBackup:                     nullRemoteBackupList(),
		Tags:                             types.MapNull(types.StringType),
		RedisVersion:                     types.StringValue("7.4"),
		Port:                             types.Int64Null(),
		RamPercentage:                    types.Int64Null(),
		QueryPerformanceFactor:           types.StringNull(),
		RespVersion:                      types.StringNull(),
	}
}

func mapValue(t *testing.T, value types.Map, key string) string {
	t.Helper()
	var tags map[string]string
	require.False(t, value.ElementsAs(context.Background(), &tags, false).HasError())
	return tags[key]
}

type fakeProDatabaseAPI struct {
	nextID       int
	db           *databases.Database
	publicAccess *bool
	created      databases.CreateDatabase
	updated      databases.UpdateDatabase
	upgraded     string
	deleted      bool
	tags         map[string]string
	getErr       error
	updateErr    error
	createErr    error
	accessErr    error
	tagsErr      error
	putTagsErr   error
	upgradeErr   error
}

func (f *fakeProDatabaseAPI) Create(_ context.Context, _ int, create databases.CreateDatabase) (int, error) {
	f.created = create
	if f.createErr != nil {
		return 0, f.createErr
	}
	if f.db != nil {
		f.db.ID = redis.Int(f.nextID)
	}
	return f.nextID, nil
}

func (f *fakeProDatabaseAPI) Get(context.Context, int, int) (*databases.Database, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.db, nil
}

func (f *fakeProDatabaseAPI) Update(_ context.Context, _ int, _ int, update databases.UpdateDatabase) error {
	f.updated = update
	return f.updateErr
}

func (f *fakeProDatabaseAPI) Delete(context.Context, int, int) error {
	f.deleted = true
	return f.getErr
}

func (f *fakeProDatabaseAPI) UpgradeRedisVersion(_ context.Context, _ int, _ int, upgrade databases.UpgradeRedisVersion) error {
	f.upgraded = redis.StringValue(upgrade.TargetRedisVersion)
	return f.upgradeErr
}

func (f *fakeProDatabaseAPI) SubscriptionPublicEndpointAccess(context.Context, int) (*bool, error) {
	if f.accessErr != nil {
		return nil, f.accessErr
	}
	return f.publicAccess, nil
}

func (f *fakeProDatabaseAPI) GetTags(context.Context, int, int) (map[string]string, error) {
	if f.tagsErr != nil {
		return nil, f.tagsErr
	}
	if f.tags == nil {
		return map[string]string{}, nil
	}
	return f.tags, nil
}

func (f *fakeProDatabaseAPI) PutTags(_ context.Context, _ int, _ int, tags map[string]string) error {
	f.tags = tags
	return f.putTagsErr
}

func TestProDatabaseResourceMethods(t *testing.T) {
	ctx := context.Background()
	schemaResponse := databaseSchema(t)
	fake := &fakeProDatabaseAPI{nextID: 42, publicAccess: redis.Bool(true), db: readyDatabase(42)}
	resourceUnderTest := readyResource(fake)

	var metadata resource.MetadataResponse
	resourceUnderTest.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "rediscloud"}, &metadata)
	assert.Equal(t, "rediscloud_subscription_database", metadata.TypeName)

	var emptyConfigure resource.ConfigureResponse
	resourceUnderTest.Configure(ctx, resource.ConfigureRequest{}, &emptyConfigure)
	assert.False(t, emptyConfigure.Diagnostics.HasError())

	var badConfigure resource.ConfigureResponse
	(&proDatabaseResource{}).Configure(ctx, resource.ConfigureRequest{ProviderData: "nope"}, &badConfigure)
	assert.True(t, badConfigure.Diagnostics.HasError())

	configured := &proDatabaseResource{}
	var goodConfigure resource.ConfigureResponse
	configured.Configure(ctx, resource.ConfigureRequest{ProviderData: &client.ApiClient{}}, &goodConfigure)
	require.False(t, goodConfigure.Diagnostics.HasError())
	assert.NotNil(t, configured.api)

	createResponse := resource.CreateResponse{State: tfsdk.State{Schema: schemaResponse}}
	resourceUnderTest.Create(ctx, resource.CreateRequest{Plan: mustPlan(t, schemaResponse, testDatabasePlan())}, &createResponse)
	require.False(t, createResponse.Diagnostics.HasError(), createResponse.Diagnostics)
	var created ProDatabaseModel
	require.False(t, createResponse.State.Get(ctx, &created).HasError())
	assert.Equal(t, "10/42", created.ID.ValueString())

	readResponse := resource.ReadResponse{State: createResponse.State}
	resourceUnderTest.Read(ctx, resource.ReadRequest{State: createResponse.State}, &readResponse)
	require.False(t, readResponse.Diagnostics.HasError(), readResponse.Diagnostics)

	updateResponse := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResponse}}
	resourceUnderTest.Update(ctx, resource.UpdateRequest{
		Plan:  mustPlan(t, schemaResponse, created),
		State: createResponse.State,
	}, &updateResponse)
	require.False(t, updateResponse.Diagnostics.HasError(), updateResponse.Diagnostics)

	plan := mustPlan(t, schemaResponse, testDatabasePlan())
	modify := resource.ModifyPlanResponse{Plan: plan}
	resourceUnderTest.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Plan:   plan,
		Config: tfsdk.Config{Schema: schemaResponse, Raw: plan.Raw},
	}, &modify)
	require.False(t, modify.Diagnostics.HasError(), modify.Diagnostics)

	nullPlan := tfsdk.Plan{Schema: schemaResponse, Raw: tftypes.NewValue(schemaResponse.Type().TerraformType(ctx), nil)}
	nullModify := resource.ModifyPlanResponse{Plan: nullPlan}
	resourceUnderTest.ModifyPlan(ctx, resource.ModifyPlanRequest{Plan: nullPlan}, &nullModify)
	assert.False(t, nullModify.Diagnostics.HasError())

	imported := resource.ImportStateResponse{State: emptyState(schemaResponse)}
	resourceUnderTest.ImportState(ctx, resource.ImportStateRequest{ID: "10/42"}, &imported)
	require.False(t, imported.Diagnostics.HasError(), imported.Diagnostics)
	var importedModel ProDatabaseModel
	require.False(t, imported.State.Get(ctx, &importedModel).HasError())
	assert.Equal(t, "10/42", importedModel.ID.ValueString())
	assert.Equal(t, int64(10), importedModel.SubscriptionID.ValueInt64())
	assert.Equal(t, int64(42), importedModel.DbID.ValueInt64())
	rejected := resource.ImportStateResponse{State: emptyState(schemaResponse)}
	resourceUnderTest.ImportState(ctx, resource.ImportStateRequest{ID: "nope"}, &rejected)
	assert.True(t, rejected.Diagnostics.HasError())

	deleted := resource.DeleteResponse{}
	resourceUnderTest.Delete(ctx, resource.DeleteRequest{State: createResponse.State}, &deleted)
	require.False(t, deleted.Diagnostics.HasError(), deleted.Diagnostics)
	assert.True(t, fake.deleted)

	missing := readyResource(&fakeProDatabaseAPI{getErr: &databases.NotFound{}})
	removed := resource.ReadResponse{State: createResponse.State}
	missing.Read(ctx, resource.ReadRequest{State: createResponse.State}, &removed)
	require.False(t, removed.Diagnostics.HasError(), removed.Diagnostics)
	assert.True(t, removed.State.Raw.IsNull())
}

func TestProDatabaseCRUDErrors(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	plan := testDatabasePlan()
	(&proDatabaseResource{}).createDatabase(ctx, &plan, &diags, time.Second)
	assert.True(t, diags.HasError())

	diags = nil
	plan.SubscriptionID = types.Int64Null()
	readyResource(&fakeProDatabaseAPI{}).createDatabase(ctx, &plan, &diags, time.Second)
	assert.True(t, diags.HasError())

	diags = nil
	plan = testDatabasePlan()
	creating := readyResource(&fakeProDatabaseAPI{createErr: assert.AnError, db: readyDatabase(1)})
	creating.createDatabase(ctx, &plan, &diags, time.Second)
	assert.True(t, diags.HasError())
	assert.True(t, plan.ID.IsNull() || plan.ID.IsUnknown())

	diags = nil
	waiting := &proDatabaseResource{api: &fakeProDatabaseAPI{}, waitForSubscriptionActive: func(context.Context, int, time.Duration) error {
		return assert.AnError
	}}
	waiting.createDatabase(ctx, &plan, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Subscription not active")

	state := testDatabasePlan()
	state.ID = types.StringValue("not-an-id")
	diags = nil
	readyResource(&fakeProDatabaseAPI{}).readDatabase(ctx, &state, &diags, time.Second)
	assert.True(t, diags.HasError())

	state.ID = types.StringValue("10/42")
	diags = nil
	readyResource(&fakeProDatabaseAPI{getErr: assert.AnError}).readDatabase(ctx, &state, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Failed to read database")

	diags = nil
	readyResource(&fakeProDatabaseAPI{db: readyDatabase(42), accessErr: assert.AnError}).readDatabase(ctx, &state, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Failed to read subscription")

	diags = nil
	readyResource(&fakeProDatabaseAPI{db: readyDatabase(42), tagsErr: assert.AnError, publicAccess: redis.Bool(true)}).readDatabase(ctx, &state, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Failed to read database tags")

	plan = testDatabasePlan()
	plan.ID = types.StringValue("bad")
	diags = nil
	readyResource(&fakeProDatabaseAPI{}).updateDatabase(ctx, &plan, nil, &diags, time.Second)
	assert.True(t, diags.HasError())

	plan.ID = types.StringValue("10/42")
	prior := plan
	prior.RedisVersion = types.StringValue("7.2")
	prior.RedisVersionActual = types.StringValue("7.2")
	plan.RedisVersion = types.StringValue("7.4")
	diags = nil
	readyResource(&fakeProDatabaseAPI{upgradeErr: assert.AnError, db: readyDatabase(42), publicAccess: redis.Bool(true)}).updateDatabase(ctx, &plan, &prior, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Failed to change Redis version")

	diags = nil
	readyResource(&fakeProDatabaseAPI{updateErr: assert.AnError, db: readyDatabase(42), publicAccess: redis.Bool(true)}).updateDatabase(ctx, &plan, nil, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Failed to update database")

	diags = nil
	readyResource(&fakeProDatabaseAPI{putTagsErr: assert.AnError, db: readyDatabase(42), publicAccess: redis.Bool(true)}).updateDatabase(ctx, &plan, nil, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Failed to write database tags")

	state.SubscriptionID = types.Int64Null()
	diags = nil
	readyResource(&fakeProDatabaseAPI{}).deleteDatabase(ctx, &state, &diags, time.Second)
	assert.Contains(t, diags.Errors()[0].Summary(), "Missing subscription_id")

	state = testDatabasePlan()
	state.ID = types.StringValue("10/42")
	diags = nil
	deleter := readyResource(&fakeProDatabaseAPI{db: readyDatabase(42)})
	deleter.deleteDatabase(ctx, &state, &diags, time.Second)
	require.False(t, diags.HasError(), diags)

	assert.Equal(t, assert.AnError.Error(), (&proDatabaseResource{api: &fakeProDatabaseAPI{}}).statusDetail(ctx, 1, 1, assert.AnError))
	assert.Equal(t, "", configuredRedisVersion(ProDatabaseModel{}, ProDatabaseModel{}, false))
	assert.False(t, ShouldSuppressModuleDiffsForRedis8(""))
	assert.False(t, ShouldSuppressModuleDiffsForRedis8("latest"))
	assert.False(t, ShouldWarnRedis8Modules("latest", true))
}

func TestProDatabaseValidatorsAndRequests(t *testing.T) {
	ctx := context.Background()
	assert.Contains(t, timeValidator{}.Description(ctx), "HH:MM")
	assert.Equal(t, timeValidator{}.Description(ctx), timeValidator{}.MarkdownDescription(ctx))

	valid := validator.StringResponse{}
	timeValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue("16:05")}, &valid)
	assert.False(t, valid.Diagnostics.HasError())
	timeValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringNull()}, &valid)
	invalid := validator.StringResponse{}
	timeValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("time_utc"), ConfigValue: types.StringValue("25:99")}, &invalid)
	assert.True(t, invalid.Diagnostics.HasError())

	assert.Contains(t, cidrValidator{}.Description(ctx), "CIDR")
	assert.Equal(t, cidrValidator{}.Description(ctx), cidrValidator{}.MarkdownDescription(ctx))
	cidrOK := validator.StringResponse{}
	cidrValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue("203.0.113.0/24")}, &cidrOK)
	assert.False(t, cidrOK.Diagnostics.HasError())
	cidrBad := validator.StringResponse{}
	cidrValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue("not-a-cidr")}, &cidrBad)
	assert.True(t, cidrBad.Diagnostics.HasError())
	cidrValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringUnknown()}, &cidrOK)

	assert.Contains(t, redisURLValidator{}.Description(ctx), "redis://")
	assert.Equal(t, redisURLValidator{}.Description(ctx), redisURLValidator{}.MarkdownDescription(ctx))
	urlOK := validator.StringResponse{}
	redisURLValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue("redis://host:6379")}, &urlOK)
	assert.False(t, urlOK.Diagnostics.HasError())
	urlBad := validator.StringResponse{}
	redisURLValidator{}.ValidateString(ctx, validator.StringRequest{ConfigValue: types.StringValue("https://example.com")}, &urlBad)
	assert.True(t, urlBad.Diagnostics.HasError())

	emptyName, nameDiags := NewModuleSet(ctx, []ModuleModel{{Name: types.StringValue("")}})
	require.False(t, nameDiags.HasError())
	moduleResponse := validator.SetResponse{}
	moduleSetValidator{}.ValidateSet(ctx, validator.SetRequest{Path: path.Root("modules"), ConfigValue: emptyName.SetValue}, &moduleResponse)
	assert.True(t, moduleResponse.Diagnostics.HasError())
	assert.Equal(t, moduleSetValidator{}.Description(ctx), moduleSetValidator{}.MarkdownDescription(ctx))

	assert.Contains(t, lowercaseTagsValidator{}.Description(ctx), "lower case")
	assert.Equal(t, lowercaseTagsValidator{}.Description(ctx), lowercaseTagsValidator{}.MarkdownDescription(ctx))
	tags, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"Market": "EMEA"})
	require.False(t, diags.HasError())
	tagResponse := validator.MapResponse{}
	lowercaseTagsValidator{}.ValidateMap(ctx, validator.MapRequest{Path: path.Root("tags"), ConfigValue: tags}, &tagResponse)
	assert.True(t, tagResponse.Diagnostics.HasError())
	assert.Contains(t, tagResponse.Diagnostics.Errors()[0].Detail(), "tag keys and values must be lower case")
	lowercaseTagsValidator{}.ValidateMap(ctx, validator.MapRequest{ConfigValue: types.MapNull(types.StringType)}, &validator.MapResponse{})

	assert.Equal(t, "rediscloud.pro.AlertSetType", NewAlertSetType().String())
	assert.Equal(t, "rediscloud.pro.ModuleSetType", NewModuleSetType().String())
	assert.Equal(t, "rediscloud.pro.RemoteBackupListType", NewRemoteBackupListType().String())
	assert.True(t, NewAlertSetType().ValueType(ctx).Type(ctx).Equal(NewAlertSetType()))
	assert.True(t, NewModuleSetType().ValueType(ctx).Type(ctx).Equal(NewModuleSetType()))
	assert.True(t, NewRemoteBackupListType().ValueType(ctx).Type(ctx).Equal(NewRemoteBackupListType()))
	converted, convertDiags := NewAlertSetType().ValueFromSet(ctx, types.SetNull(alertElementType))
	require.False(t, convertDiags.HasError())
	assert.True(t, converted.(AlertSetValue).IsNull())
	moduleConverted, moduleDiags := NewModuleSetType().ValueFromSet(ctx, types.SetNull(moduleElementType))
	require.False(t, moduleDiags.HasError())
	assert.True(t, moduleConverted.(ModuleSetValue).IsNull())
	backupConverted, backupDiags := NewRemoteBackupListType().ValueFromList(ctx, types.ListNull(remoteBackupElementType))
	require.False(t, backupDiags.HasError())
	assert.True(t, backupConverted.(RemoteBackupListValue).IsNull())
	unknownModules := ModuleSetValue{SetValue: types.SetUnknown(moduleElementType)}
	unknownTF, err := unknownModules.ToTerraformValue(ctx)
	require.NoError(t, err)
	assert.False(t, unknownTF.IsKnown())
	unknownBackups := RemoteBackupListValue{ListValue: types.ListUnknown(remoteBackupElementType)}
	unknownBackupTF, err := unknownBackups.ToTerraformValue(ctx)
	require.NoError(t, err)
	assert.False(t, unknownBackupTF.IsKnown())

	alerts, diags := NewAlertSet(ctx, []AlertModel{{Name: types.StringValue("dataset-size"), Value: types.Int64Value(80)}})
	require.False(t, diags.HasError())
	modules, diags := NewModuleSet(ctx, []ModuleModel{{Name: types.StringValue("RedisBloom")}})
	require.False(t, diags.HasError())
	hashes, diags := types.ListValueFrom(ctx, types.StringType, []string{"custom"})
	require.False(t, diags.HasError())
	replicas, diags := types.SetValueFrom(ctx, types.StringType, []string{"redis://host:6379"})
	require.False(t, diags.HasError())
	certificates, diags := types.ListValueFrom(ctx, types.StringType, []string{"cert"})
	require.False(t, diags.HasError())
	plan := testDatabasePlan()
	plan.Alert = alerts
	plan.Modules = modules
	plan.Password = types.StringValue("secret")
	plan.Port = types.Int64Value(10000)
	plan.RespVersion = types.StringValue("resp3")
	plan.QueryPerformanceFactor = types.StringValue("2x")
	plan.AverageItemSizeInBytes = types.Int64Value(100)
	plan.MemoryLimitInGB = types.Float64Value(2)
	plan.DatasetSizeInGB = types.Float64Null()
	created, createDiags := buildCreateDatabase(ctx, plan)
	require.False(t, createDiags.HasError())
	assert.Equal(t, "secret", redis.StringValue(created.Password))
	assert.Equal(t, 10000, redis.IntValue(created.PortNumber))
	assert.Equal(t, 2.0, redis.Float64Value(created.MemoryLimitInGB))
	require.Len(t, created.Modules, 1)

	plan.EnableTLS = types.BoolValue(true)
	plan.ClientTLSCertificates = certificates
	plan.ClientSSLCertificate = types.StringValue("ssl")
	plan.HashingPolicy = hashes
	plan.ReplicaOf = replicas
	plan.PeriodicBackupPath = types.StringValue("s3://bucket")
	plan.RamPercentage = types.Int64Value(25)
	updated, updateDiags := buildUpdateDatabase(ctx, plan, nil, redis.Bool(true))
	require.False(t, updateDiags.HasError())
	assert.True(t, redis.BoolValue(updated.EnableTls))
	assert.Equal(t, "ssl", redis.StringValue(updated.ClientSSLCertificate))
	assert.Equal(t, "s3://bucket", redis.StringValue(updated.PeriodicBackupPath))
	require.NotNil(t, updated.RegexRules)

	plan.EnableTLS = types.BoolValue(false)
	plan.ClientSSLCertificate = types.StringValue("")
	_, rejectedDiags := buildUpdateDatabase(ctx, plan, nil, redis.Bool(true))
	assert.True(t, rejectedDiags.HasError())
	assert.Contains(t, rejectedDiags.Errors()[0].Detail(), "TLS certificates may not be provided while enable_tls is false")
}

func databaseSchema(t *testing.T) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	NewProDatabaseResource().Schema(context.Background(), resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError())
	return response.Schema
}

func timedModel(model ProDatabaseModel) ProDatabaseModel {
	model.Timeouts = timeouts.Value{Object: types.ObjectNull(map[string]attr.Type{
		"create": types.StringType,
		"read":   types.StringType,
		"update": types.StringType,
		"delete": types.StringType,
	})}
	return model
}

func mustPlan(t *testing.T, resourceSchema schema.Schema, model ProDatabaseModel) tfsdk.Plan {
	t.Helper()
	model = timedModel(model)
	plan := tfsdk.Plan{Schema: resourceSchema}
	diags := plan.Set(context.Background(), &model)
	require.False(t, diags.HasError(), diags)
	return plan
}

func emptyState(resourceSchema schema.Schema) tfsdk.State {
	ctx := context.Background()
	terraformType := resourceSchema.Type().TerraformType(ctx).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, attributeType := range terraformType.AttributeTypes {
		values[name] = tftypes.NewValue(attributeType, nil)
	}
	return tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(terraformType, values)}
}

func readyResource(api proDatabaseAPI) *proDatabaseResource {
	return &proDatabaseResource{
		api:                       api,
		waitForSubscriptionActive: func(context.Context, int, time.Duration) error { return nil },
		waitForDatabaseActive:     func(context.Context, int, int, time.Duration) error { return nil },
	}
}

func readyDatabase(id int) *databases.Database {
	return &databases.Database{
		ID:                    redis.Int(id),
		Name:                  redis.String("example"),
		Protocol:              redis.String("redis"),
		RedisVersion:          redis.String("7.4"),
		RespVersion:           redis.String("resp3"),
		DataPersistence:       redis.String("none"),
		DataEvictionPolicy:    redis.String("volatile-lru"),
		Replication:           redis.Bool(true),
		ThroughputMeasurement: &databases.Throughput{By: redis.String("operations-per-second"), Value: redis.Int(1000)},
		DatasetSizeInGB:       redis.Float64(1),
		Security: &databases.Security{
			Password:          redis.String("generated"),
			SourceIPs:         []*string{redis.String("0.0.0.0/0")},
			EnableTls:         redis.Bool(false),
			EnableDefaultUser: redis.Bool(true),
		},
		Status: redis.String("active"),
	}
}
