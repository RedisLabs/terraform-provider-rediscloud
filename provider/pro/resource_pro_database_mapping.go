package pro

import (
	"context"
	"fmt"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var values []string
	diags := list.ElementsAs(ctx, &values, false)
	return values, diags
}

func stringSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var values []string
	diags := set.ElementsAs(ctx, &values, false)
	return values, diags
}

func stringMap(ctx context.Context, value types.Map) (map[string]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return map[string]string{}, nil
	}
	tags := map[string]string{}
	diags := value.ElementsAs(ctx, &tags, false)
	return tags, diags
}

func stringPointers(values []string) []*string {
	if values == nil {
		return nil
	}
	result := make([]*string, len(values))
	for i, value := range values {
		result[i] = redis.String(value)
	}
	return result
}

func pointerStrings(values []*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, redis.StringValue(value))
	}
	return result
}

func knownString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	return value.ValueString(), true
}

func floatKnownNonZero(value types.Float64) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueFloat64() != 0
}

func listFromStrings(ctx context.Context, values []string) (types.List, diag.Diagnostics) {
	if values == nil {
		values = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

func setFromStrings(ctx context.Context, values []string) (types.Set, diag.Diagnostics) {
	if values == nil {
		values = []string{}
	}
	return types.SetValueFrom(ctx, types.StringType, values)
}

func mapFromStrings(ctx context.Context, values map[string]string) (types.Map, diag.Diagnostics) {
	if values == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, values)
}

// AlertsFromAPI maps database alerts onto the alert set.
func AlertsFromAPI(ctx context.Context, alerts []*databases.Alert) (AlertSetValue, diag.Diagnostics) {
	models := make([]AlertModel, 0, len(alerts))
	for _, alert := range alerts {
		models = append(models, AlertModel{
			Name:  types.StringValue(redis.StringValue(alert.Name)),
			Value: types.Int64Value(int64(redis.IntValue(alert.Value))),
		})
	}
	return NewAlertSet(ctx, models)
}

// ModulesFromAPI maps database modules onto the modules set.
func ModulesFromAPI(ctx context.Context, modules []*databases.Module) (ModuleSetValue, diag.Diagnostics) {
	models := make([]ModuleModel, 0, len(modules))
	for _, module := range modules {
		models = append(models, ModuleModel{Name: types.StringValue(redis.StringValue(module.Name))})
	}
	return NewModuleSet(ctx, models)
}

// HashingPolicyFromAPI maps clustering regex rules. The provider's default
// pair is stored as an empty list, matching the SDKv2 helper.
func HashingPolicyFromAPI(ctx context.Context, rules []*databases.RegexRule) (types.List, diag.Diagnostics) {
	patterns := make([]string, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if rule.Ordinal >= 0 && rule.Ordinal < len(patterns) {
			patterns[rule.Ordinal] = rule.Pattern
		}
	}
	if len(patterns) == 2 && patterns[0] == ".*\\{(?<tag>.*)\\}.*" && patterns[1] == "(?<tag>.*)" {
		patterns = []string{}
	}
	return listFromStrings(ctx, patterns)
}

// RemoteBackupFromAPI maps the backup response. storage_type is not returned
// by the API, so the prior state value is kept. An absent backup stays null
// when the prior block was null, and becomes an empty list when a block is
// being cleared. time_utc keeps an empty string when the API omits it.
func RemoteBackupFromAPI(ctx context.Context, backup *databases.Backup, periodicBackupPath string, prior RemoteBackupListValue) (RemoteBackupListValue, diag.Diagnostics) {
	if periodicBackupPath != "" || backup == nil || !redis.BoolValue(backup.Enabled) {
		if prior.IsNull() || prior.IsUnknown() {
			return nullRemoteBackupList(), nil
		}
		return emptyRemoteBackupList(ctx)
	}

	storageType := backupStorageType(ctx, prior)
	//TODO(TF3.0) make time_utc nullable
	timeUTC := types.StringValue(redis.StringValue(backup.TimeUTC))
	if !prior.IsNull() && !prior.IsUnknown() {
		priorModels, diags := prior.AsModels(ctx)
		if diags.HasError() {
			return nullRemoteBackupList(), diags
		}
		if len(priorModels) == 1 {
			priorTime, ok := knownString(priorModels[0].TimeUTC)
			if ok && backupTimesEquivalent(redis.StringValue(backup.Interval), priorTime, timeUTC.ValueString()) {
				timeUTC = priorModels[0].TimeUTC
			}
		}
	}

	return NewRemoteBackupList(ctx, []RemoteBackupModel{{
		Interval:    types.StringValue(redis.StringValue(backup.Interval)),
		TimeUTC:     timeUTC,
		StorageType: types.StringValue(storageType),
		StoragePath: types.StringValue(redis.StringValue(backup.Destination)),
	}})
}

func backupStorageType(ctx context.Context, prior RemoteBackupListValue) string {
	if prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) == 0 {
		return ""
	}
	models, diags := prior.AsModels(ctx)
	if diags.HasError() || len(models) == 0 || models[0].StorageType.IsNull() || models[0].StorageType.IsUnknown() {
		return ""
	}
	return models[0].StorageType.ValueString()
}

// TagsFromAPI maps the tags response. No tags stay null so an omitted block
// does not drift against an empty map.
func TagsFromAPI(ctx context.Context, tags map[string]string, prior types.Map) (types.Map, diag.Diagnostics) {
	if len(tags) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior, nil
		}
		return types.MapNull(types.StringType), nil
	}
	return mapFromStrings(ctx, tags)
}

func alertsForRequest(ctx context.Context, alerts AlertSetValue) ([]*databases.Alert, diag.Diagnostics) {
	if alerts.IsNull() || alerts.IsUnknown() {
		return []*databases.Alert{}, nil
	}
	models, diags := alerts.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]*databases.Alert, 0, len(models))
	for _, model := range models {
		result = append(result, &databases.Alert{
			Name:  redis.String(model.Name.ValueString()),
			Value: redis.Int(int(model.Value.ValueInt64())),
		})
	}
	return result, nil
}

func modulesForRequest(ctx context.Context, modules ModuleSetValue) ([]*databases.Module, diag.Diagnostics) {
	if modules.IsNull() || modules.IsUnknown() {
		return []*databases.Module{}, nil
	}
	models, diags := modules.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}
	result := make([]*databases.Module, 0, len(models))
	for _, model := range models {
		result = append(result, &databases.Module{Name: redis.String(model.Name.ValueString())})
	}
	return result, nil
}

// backupConfigForRequest builds the create/update backup payload.
// An absent remote_backup block omits the field when periodic_backup_path is
// present as a string, matching BuildBackupPlan. A null periodic path with no
// block disables the backup.
func backupConfigForRequest(ctx context.Context, backups RemoteBackupListValue, periodicBackupPath types.String) (*databases.DatabaseBackupConfig, diag.Diagnostics) {
	if backups.IsUnknown() {
		return nil, nil
	}
	if backups.IsNull() || len(backups.Elements()) == 0 {
		if periodicBackupPath.IsNull() {
			return &databases.DatabaseBackupConfig{Active: redis.Bool(false)}, nil
		}
		return nil, nil
	}
	models, diags := backups.AsModels(ctx)
	if diags.HasError() {
		return nil, diags
	}
	if len(models) == 0 {
		if periodicBackupPath.IsNull() {
			return &databases.DatabaseBackupConfig{Active: redis.Bool(false)}, nil
		}
		return nil, nil
	}
	backup := models[0]
	config := &databases.DatabaseBackupConfig{
		Active:      redis.Bool(true),
		Interval:    redis.String(backup.Interval.ValueString()),
		StorageType: redis.String(backup.StorageType.ValueString()),
		StoragePath: redis.String(backup.StoragePath.ValueString()),
	}
	if timeUTC, ok := knownString(backup.TimeUTC); ok && timeUTC != "" {
		config.TimeUTC = redis.String(timeUTC)
	}
	return config, nil
}

func moduleCount(ctx context.Context, modules ModuleSetValue) (int, diag.Diagnostics) {
	if modules.IsNull() || modules.IsUnknown() {
		return 0, nil
	}
	models, diags := modules.AsModels(ctx)
	return len(models), diags
}

type proDatabasePlanAdjustment struct {
	Plan            ProDatabaseModel
	RequiresReplace path.Paths
	Diagnostics     diag.Diagnostics
}

// adjustProDatabasePlan applies the SDKv2 CustomizeDiff and DiffSuppressFunc
// rules that have no schema-validator equivalent.
func adjustProDatabasePlan(ctx context.Context, config, plan, state ProDatabaseModel, hasState bool) proDatabasePlanAdjustment {
	var result proDatabasePlanAdjustment
	result.Plan = plan

	// Use the planned value. Config omits the schema default, so a null
	// enable_passwordless is false there and true on the plan when state or
	// the default says passwordless. The conflict check and the password
	// clear have to see the same value.
	if password, ok := knownString(config.Password); ok && result.Plan.EnablePasswordless.ValueBool() {
		if err := ShouldRejectPasswordlessWithPassword(true, true, password); err != nil {
			result.Diagnostics.AddError("Conflicting attributes", err.Error())
		}
	}
	if result.Plan.EnablePasswordless.ValueBool() {
		result.Plan.Password = types.StringValue("")
	} else if hasState && config.Password.IsNull() {
		result.Plan.Password = state.Password
	}

	if periodic, ok := knownString(config.PeriodicBackupPath); ok && periodic != "" &&
		!config.RemoteBackup.IsNull() && !config.RemoteBackup.IsUnknown() && len(config.RemoteBackup.Elements()) > 0 {
		result.Diagnostics.AddError(
			"Conflicting attributes",
			`"periodic_backup_path" cannot be specified when "remote_backup" is specified`,
		)
	}

	if !config.RemoteBackup.IsNull() && !config.RemoteBackup.IsUnknown() {
		backups, diags := config.RemoteBackup.AsModels(ctx)
		result.Diagnostics.Append(diags...)
		if !diags.HasError() {
			for _, backup := range backups {
				timeUTC, ok := knownString(backup.TimeUTC)
				if ok && timeUTC != "" && backup.Interval.ValueString() != databases.BackupIntervalEvery12Hours && backup.Interval.ValueString() != databases.BackupIntervalEvery24Hours {
					result.Diagnostics.AddError("Invalid remote_backup", remoteBackupIntervalError())
				}
			}
		}
	}

	if hasState && !plan.RemoteBackup.IsNull() && !plan.RemoteBackup.IsUnknown() && !state.RemoteBackup.IsNull() && !state.RemoteBackup.IsUnknown() {
		plannedBackups, plannedDiags := plan.RemoteBackup.AsModels(ctx)
		stateBackups, stateDiags := state.RemoteBackup.AsModels(ctx)
		result.Diagnostics.Append(plannedDiags...)
		result.Diagnostics.Append(stateDiags...)
		if !plannedDiags.HasError() && !stateDiags.HasError() && len(plannedBackups) == 1 && len(stateBackups) == 1 {
			plannedTime, plannedOK := knownString(plannedBackups[0].TimeUTC)
			stateTime, stateOK := knownString(stateBackups[0].TimeUTC)
			interval := plannedBackups[0].Interval.ValueString()
			if plannedOK && stateOK && backupTimesEquivalent(interval, stateTime, plannedTime) {
				plannedBackups[0].TimeUTC = stateBackups[0].TimeUTC
				updated, diags := NewRemoteBackupList(ctx, plannedBackups)
				result.Diagnostics.Append(diags...)
				if !diags.HasError() {
					result.Plan.RemoteBackup = updated
				}
			}
		}
	}

	result.Plan.RedisVersion = adjustRedisVersionPlan(config, result.Plan, state, hasState)

	if stateQPF, stateOK := knownString(state.QueryPerformanceFactor); hasState && stateOK {
		if configQPF, configOK := knownString(config.QueryPerformanceFactor); configOK && strings.EqualFold(configQPF, stateQPF) {
			result.Plan.QueryPerformanceFactor = state.QueryPerformanceFactor
		}
	}

	version := configuredRedisVersion(config, state, hasState)
	if version != "" && hasState {
		count, diags := moduleCount(ctx, config.Modules)
		result.Diagnostics.Append(diags...)
		if ShouldWarnRedis8Modules(version, count > 0) {
			logRedis8ModulesWarning(version)
		}
	}
	if hasState && !result.Plan.Modules.IsUnknown() && !state.Modules.IsUnknown() {
		if ShouldSuppressModuleDiffsForRedis8(version) {
			result.Plan.Modules = state.Modules
		} else if !result.Plan.Modules.Equal(state.Modules) {
			result.RequiresReplace = append(result.RequiresReplace, path.Root("modules"))
		}
	}

	return result
}

func buildCreateDatabase(ctx context.Context, plan ProDatabaseModel) (databases.CreateDatabase, diag.Diagnostics) {
	var diags diag.Diagnostics
	modules, moduleDiags := modulesForRequest(ctx, plan.Modules)
	diags.Append(moduleDiags...)
	alerts, alertDiags := alertsForRequest(ctx, plan.Alert)
	diags.Append(alertDiags...)
	backup, backupDiags := backupConfigForRequest(ctx, plan.RemoteBackup, plan.PeriodicBackupPath)
	diags.Append(backupDiags...)
	if diags.HasError() {
		return databases.CreateDatabase{}, diags
	}

	create := databases.CreateDatabase{
		Name:                 redis.String(plan.Name.ValueString()),
		Protocol:             redis.String(plan.Protocol.ValueString()),
		SupportOSSClusterAPI: redis.Bool(plan.SupportOSSClusterAPI.ValueBool()),
		DataPersistence:      redis.String(plan.DataPersistence.ValueString()),
		DataEvictionPolicy:   redis.String(plan.DataEviction.ValueString()),
		Replication:          redis.Bool(plan.Replication.ValueBool()),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(plan.ThroughputMeasurementBy.ValueString()),
			Value: redis.Int(int(plan.ThroughputMeasurementValue.ValueInt64())),
		},
		Modules:                 modules,
		Alerts:                  alerts,
		RemoteBackup:            backup,
		AutoMinorVersionUpgrade: redis.Bool(plan.AutoMinorVersionUpgrade.ValueBool()),
	}

	if value, ok := knownString(plan.QueryPerformanceFactor); ok && value != "" {
		create.QueryPerformanceFactor = redis.String(value)
	}
	if value, ok := knownString(plan.RedisVersion); ok && value != "" {
		create.RedisVersion = redis.String(value)
	}
	if value, ok := knownString(plan.Password); ok && value != "" {
		create.Password = redis.String(value)
	}
	if plan.EnablePasswordless.ValueBool() {
		create.Password = redis.String("")
	}
	if !plan.AverageItemSizeInBytes.IsNull() && !plan.AverageItemSizeInBytes.IsUnknown() && plan.AverageItemSizeInBytes.ValueInt64() > 0 {
		create.AverageItemSizeInBytes = redis.Int(int(plan.AverageItemSizeInBytes.ValueInt64()))
	}
	if !plan.RamPercentage.IsNull() && !plan.RamPercentage.IsUnknown() && plan.RamPercentage.ValueInt64() > 0 {
		create.RamPercentage = redis.Int(int(plan.RamPercentage.ValueInt64()))
	}
	if floatKnownNonZero(plan.DatasetSizeInGB) {
		create.DatasetSizeInGB = redis.Float64(plan.DatasetSizeInGB.ValueFloat64())
	}
	if floatKnownNonZero(plan.MemoryLimitInGB) {
		create.MemoryLimitInGB = redis.Float64(plan.MemoryLimitInGB.ValueFloat64())
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		create.PortNumber = redis.Int(int(plan.Port.ValueInt64()))
	}
	if value, ok := knownString(plan.RespVersion); ok && value != "" {
		create.RespVersion = redis.String(value)
	}
	return create, diags
}

func buildUpdateDatabase(ctx context.Context, plan ProDatabaseModel, prior *ProDatabaseModel, publicEndpointAccess *bool) (databases.UpdateDatabase, diag.Diagnostics) {
	var diags diag.Diagnostics
	if prior != nil && prior.EnablePasswordless.ValueBool() && !plan.EnablePasswordless.ValueBool() {
		if password, ok := knownString(plan.Password); !ok || password == "" {
			diags.AddError("Invalid password", "when disabling passwordless mode, you must provide a 'password'")
			return databases.UpdateDatabase{}, diags
		}
	}

	alerts, alertDiags := alertsForRequest(ctx, plan.Alert)
	diags.Append(alertDiags...)
	backup, backupDiags := backupConfigForRequest(ctx, plan.RemoteBackup, plan.PeriodicBackupPath)
	diags.Append(backupDiags...)
	sourceIPs, sourceDiags := stringSet(ctx, plan.SourceIPs)
	diags.Append(sourceDiags...)
	replicaOf, replicaDiags := stringSet(ctx, plan.ReplicaOf)
	diags.Append(replicaDiags...)
	hashingPolicy, hashingDiags := stringList(ctx, plan.HashingPolicy)
	diags.Append(hashingDiags...)
	tlsCertificates, tlsDiags := stringList(ctx, plan.ClientTLSCertificates)
	diags.Append(tlsDiags...)
	if diags.HasError() {
		return databases.UpdateDatabase{}, diags
	}

	if plan.SourceIPs.IsNull() || plan.SourceIPs.IsUnknown() || isDefaultSourceIPStrings(sourceIPs) {
		sourceIPs = defaultSourceIPsForAccess(publicEndpointAccess)
	}

	update := databases.UpdateDatabase{
		Name:                 redis.String(plan.Name.ValueString()),
		SupportOSSClusterAPI: redis.Bool(plan.SupportOSSClusterAPI.ValueBool()),
		Replication:          redis.Bool(plan.Replication.ValueBool()),
		ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
			By:    redis.String(plan.ThroughputMeasurementBy.ValueString()),
			Value: redis.Int(int(plan.ThroughputMeasurementValue.ValueInt64())),
		},
		DataPersistence:                     redis.String(plan.DataPersistence.ValueString()),
		DataEvictionPolicy:                  redis.String(plan.DataEviction.ValueString()),
		SourceIP:                            stringPointers(sourceIPs),
		Alerts:                              &alerts,
		RemoteBackup:                        backup,
		EnableDefaultUser:                   redis.Bool(plan.EnableDefaultUser.ValueBool()),
		AutoMinorVersionUpgrade:             redis.Bool(plan.AutoMinorVersionUpgrade.ValueBool()),
		ReplicaOf:                           stringPointers(replicaOf),
		UseExternalEndpointForOSSClusterAPI: redis.Bool(plan.ExternalEndpointForOSSClusterAPI.ValueBool()),
	}
	if update.ReplicaOf == nil {
		update.ReplicaOf = []*string{}
	}
	if floatKnownNonZero(plan.DatasetSizeInGB) {
		update.DatasetSizeInGB = redis.Float64(plan.DatasetSizeInGB.ValueFloat64())
	} else if floatKnownNonZero(plan.MemoryLimitInGB) {
		update.MemoryLimitInGB = redis.Float64(plan.MemoryLimitInGB.ValueFloat64())
	}
	if value, ok := knownString(plan.QueryPerformanceFactor); ok && value != "" {
		update.QueryPerformanceFactor = redis.String(value)
	}
	if plan.EnablePasswordless.ValueBool() {
		update.Password = redis.String("")
	} else if password, ok := knownString(plan.Password); ok && password != "" {
		update.Password = redis.String(password)
	}
	if !plan.RamPercentage.IsNull() && !plan.RamPercentage.IsUnknown() && plan.RamPercentage.ValueInt64() > 0 {
		update.RamPercentage = redis.Int(int(plan.RamPercentage.ValueInt64()))
	}
	if len(hashingPolicy) != 0 {
		update.RegexRules = stringPointers(hashingPolicy)
	}
	if path, ok := knownString(plan.PeriodicBackupPath); ok && path != "" {
		update.PeriodicBackupPath = redis.String(path)
	}
	if value, ok := knownString(plan.RespVersion); ok && value != "" {
		update.RespVersion = redis.String(value)
	}

	sslCertificate, _ := knownString(plan.ClientSSLCertificate)
	if plan.EnableTLS.ValueBool() {
		update.EnableTls = redis.Bool(true)
		if sslCertificate != "" {
			update.ClientSSLCertificate = redis.String(sslCertificate)
		}
		certificates := stringPointers(tlsCertificates)
		update.ClientTLSCertificates = &certificates
	} else if sslCertificate != "" {
		update.ClientSSLCertificate = redis.String(sslCertificate)
	} else if len(tlsCertificates) > 0 {
		diags.AddError("Invalid TLS configuration", "TLS certificates may not be provided while enable_tls is false")
		return databases.UpdateDatabase{}, diags
	} else {
		update.EnableTls = redis.Bool(false)
	}

	return update, diags
}

func ensureKnownString(value types.String, fallback string) types.String {
	if value.IsUnknown() {
		return types.StringValue(fallback)
	}
	return value
}

func ensureKnownBool(value types.Bool, fallback bool) types.Bool {
	if value.IsNull() || value.IsUnknown() {
		return types.BoolValue(fallback)
	}
	return value
}

func ensureKnownInt(value types.Int64, fallback int64) types.Int64 {
	if value.IsUnknown() {
		return types.Int64Value(fallback)
	}
	return value
}

func ensureKnownFloat(value types.Float64) types.Float64 {
	if value.IsUnknown() {
		return types.Float64Null()
	}
	return value
}

// applyProDatabaseRead copies an API database onto the model using the SDKv2
// read rules. publicEndpointAccess is consulted only when source IP defaults
// must be recomputed; pass nil when the caller has not loaded the subscription.
func applyProDatabaseRead(ctx context.Context, model ProDatabaseModel, db *databases.Database, publicEndpointAccess *bool, tags map[string]string, resolveSourceDefaults bool) (ProDatabaseModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if db == nil {
		diags.AddError("Missing database", "The Redis Cloud API returned no database.")
		return model, diags
	}
	if db.ThroughputMeasurement == nil {
		diags.AddError("Missing throughput", "The Redis Cloud API returned a database without throughput measurement.")
		return model, diags
	}

	model.DbID = types.Int64Value(int64(redis.IntValue(db.ID)))
	model.Name = types.StringValue(redis.StringValue(db.Name))
	model.Protocol = types.StringValue(redis.StringValue(db.Protocol))
	model.SupportOSSClusterAPI = types.BoolValue(redis.BoolValue(db.SupportOSSClusterAPI))
	//TODO(TF3.0) make resp_version nullable
	model.RespVersion = types.StringValue(redis.StringValue(db.RespVersion))
	model.DataPersistence = types.StringValue(redis.StringValue(db.DataPersistence))
	model.DataEviction = types.StringValue(redis.StringValue(db.DataEvictionPolicy))
	model.Replication = types.BoolValue(redis.BoolValue(db.Replication))
	model.ThroughputMeasurementBy = types.StringValue(redis.StringValue(db.ThroughputMeasurement.By))
	model.ThroughputMeasurementValue = types.Int64Value(int64(redis.IntValue(db.ThroughputMeasurement.Value)))
	//TODO(TF3.0) make public_endpoint nullable
	model.PublicEndpoint = types.StringValue(redis.StringValue(db.PublicEndpoint))
	//TODO(TF3.0) make private_endpoint nullable
	model.PrivateEndpoint = types.StringValue(redis.StringValue(db.PrivateEndpoint))

	if db.QueryPerformanceFactor != nil {
		model.QueryPerformanceFactor = types.StringValue(redis.StringValue(db.QueryPerformanceFactor))
	} else if model.QueryPerformanceFactor.IsUnknown() {
		model.QueryPerformanceFactor = types.StringNull()
	}
	if db.RedisVersion != nil {
		model.RedisVersionActual = types.StringValue(redis.StringValue(db.RedisVersion))
		if model.RedisVersion.IsNull() || model.RedisVersion.IsUnknown() || model.RedisVersion.ValueString() == "" {
			model.RedisVersion = types.StringValue(redis.StringValue(db.RedisVersion))
		}
	} else if model.RedisVersionActual.IsUnknown() {
		model.RedisVersionActual = types.StringNull()
	}
	if model.RedisVersion.IsUnknown() {
		model.RedisVersion = types.StringNull()
	}

	version := redis.StringValue(db.RedisVersion)
	modules, moduleDiags := ModulesFromAPI(ctx, db.Modules)
	diags.Append(moduleDiags...)
	if !moduleDiags.HasError() {
		if ShouldSuppressModuleDiffsForRedis8(version) {
			if !model.Modules.IsNull() && !model.Modules.IsUnknown() && len(model.Modules.Elements()) > 0 {
				model.Modules = modules
			} else if model.Modules.IsUnknown() {
				model.Modules = nullModuleSet()
			}
		} else {
			model.Modules = modules
		}
	}

	alerts, alertDiags := AlertsFromAPI(ctx, db.Alerts)
	diags.Append(alertDiags...)
	if !alertDiags.HasError() {
		if len(alerts.Elements()) == 0 && (model.Alert.IsNull() || model.Alert.IsUnknown()) {
			model.Alert = nullAlertSet()
		} else {
			model.Alert = alerts
		}
	}

	model.AverageItemSizeInBytes = ensureKnownInt(model.AverageItemSizeInBytes, 0)
	if db.RamPercentage != nil {
		model.RamPercentage = types.Int64Value(int64(redis.IntValue(db.RamPercentage)))
	} else if model.RamPercentage.IsUnknown() {
		model.RamPercentage = types.Int64Null()
	}
	model.ExternalEndpointForOSSClusterAPI = ensureKnownBool(model.ExternalEndpointForOSSClusterAPI, false)

	memoryInState := floatKnownNonZero(model.MemoryLimitInGB)
	datasetInState := floatKnownNonZero(model.DatasetSizeInGB)
	if !memoryInState {
		model.DatasetSizeInGB = types.Float64Value(redis.Float64Value(db.DatasetSizeInGB))
	}
	if !datasetInState {
		model.MemoryLimitInGB = types.Float64Value(redis.Float64Value(db.MemoryLimitInGB))
	}
	model.DatasetSizeInGB = ensureKnownFloat(model.DatasetSizeInGB)
	model.MemoryLimitInGB = ensureKnownFloat(model.MemoryLimitInGB)

	password := ""
	if !model.Password.IsNull() && !model.Password.IsUnknown() {
		password = model.Password.ValueString()
	}
	passwordless := false
	if db.Security != nil && redis.StringValue(db.Protocol) == "redis" {
		password = redis.StringValue(db.Security.Password)
		passwordless = db.Security.Password != nil && *db.Security.Password == ""
	}
	model.Password = types.StringValue(password)
	model.EnablePasswordless = types.BoolValue(passwordless)

	var apiSourceIPs []string
	if db.Security != nil {
		apiSourceIPs = pointerStrings(db.Security.SourceIPs)
	}
	stateSourceIPs, sourceDiags := stringSet(ctx, model.SourceIPs)
	diags.Append(sourceDiags...)
	resolved, needsDefault := resolveReadSourceIPs(apiSourceIPs, stateSourceIPs)
	if needsDefault {
		if resolveSourceDefaults {
			resolved = defaultSourceIPsForAccess(publicEndpointAccess)
		} else if model.SourceIPs.IsNull() || model.SourceIPs.IsUnknown() {
			resolved = []string{}
		} else {
			resolved = stateSourceIPs
		}
	}
	sourceSet, setDiags := setFromStrings(ctx, resolved)
	diags.Append(setDiags...)
	model.SourceIPs = sourceSet

	var rules []*databases.RegexRule
	if db.Clustering != nil {
		rules = db.Clustering.RegexRules
	}
	hashing, hashingDiags := HashingPolicyFromAPI(ctx, rules)
	diags.Append(hashingDiags...)
	model.HashingPolicy = hashing

	enableTLS := false
	enableDefaultUser := true
	tlsAuth := false
	if db.Security != nil {
		enableTLS = redis.BoolValue(db.Security.EnableTls)
		if db.Security.EnableDefaultUser != nil {
			enableDefaultUser = redis.BoolValue(db.Security.EnableDefaultUser)
		}
		tlsAuth = redis.BoolValue(db.Security.TLSClientAuthentication)
	}
	model.EnableTLS = types.BoolValue(enableTLS)
	model.EnableDefaultUser = types.BoolValue(enableDefaultUser)

	autoMinor := true
	if db.AutoMinorVersionUpgrade != nil {
		autoMinor = redis.BoolValue(db.AutoMinorVersionUpgrade)
	}
	model.AutoMinorVersionUpgrade = types.BoolValue(autoMinor)

	priorTLSNull := model.ClientTLSCertificates.IsNull() || model.ClientTLSCertificates.IsUnknown()
	certificates, certDiags := applyCertificateHints(tlsAuth, model.ClientSSLCertificate, model.ClientTLSCertificates)
	diags.Append(certDiags...)
	model.ClientSSLCertificate = ensureKnownString(certificates.sslCertificate, "")
	// An omitted certificate list stays null. Writing an empty list would drift
	// against config on every plan; the SDKv2 empty list compared equal to unset.
	if !tlsAuth && priorTLSNull && (certificates.tlsCertificates.IsNull() || len(certificates.tlsCertificates.Elements()) == 0) {
		model.ClientTLSCertificates = types.ListNull(types.StringType)
	} else if certificates.tlsCertificates.IsNull() || certificates.tlsCertificates.IsUnknown() {
		empty, emptyDiags := listFromStrings(ctx, []string{})
		diags.Append(emptyDiags...)
		model.ClientTLSCertificates = empty
	} else {
		model.ClientTLSCertificates = certificates.tlsCertificates
	}

	periodic, _ := knownString(model.PeriodicBackupPath)
	remoteBackup, backupDiags := RemoteBackupFromAPI(ctx, db.Backup, periodic, model.RemoteBackup)
	diags.Append(backupDiags...)
	model.RemoteBackup = remoteBackup

	mappedTags, tagDiags := TagsFromAPI(ctx, tags, model.Tags)
	diags.Append(tagDiags...)
	model.Tags = mappedTags

	if model.ReplicaOf.IsUnknown() {
		model.ReplicaOf = types.SetNull(types.StringType)
	}
	if model.Port.IsUnknown() {
		model.Port = types.Int64Null()
	}
	model.ClientSSLCertificate = ensureKnownString(model.ClientSSLCertificate, "")
	model.PeriodicBackupPath = ensureKnownString(model.PeriodicBackupPath, "")
	return model, diags
}

func observedStatusDetail(status *string, err error) string {
	if status == nil || *status == "" {
		return err.Error()
	}
	return fmt.Sprintf("last observed status %s: %s", *status, err.Error())
}
