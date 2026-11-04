package pro

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *proDatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var model ProDatabaseDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(model.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Subscription ID", err.Error())
		return
	}

	list := d.client.Client.Database.List(ctx, subId)
	dbs, err := filterProDatabases(list, proDatabaseFilters(model))
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Pro Databases", err.Error())
		return
	}

	if len(dbs) == 0 {
		resp.Diagnostics.AddError(
			"No Pro Databases Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(dbs) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Pro Databases Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	// Some attributes are only returned when retrieving a single database.
	dbId := redis.IntValue(dbs[0].ID)
	db, err := d.client.Client.Database.Get(ctx, subId, dbId)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Pro Database", err.Error())
		return
	}

	model.ID = types.StringValue(fmt.Sprintf("%d/%d", subId, dbId))
	model.DbID = types.Int64Value(int64(dbId))
	model.Name = types.StringValue(redis.StringValue(db.Name))
	model.Protocol = types.StringValue(redis.StringValue(db.Protocol))
	model.Region = types.StringValue(redis.StringValue(db.Region))
	model.MemoryLimitInGB = types.Float64Value(redis.Float64Value(db.MemoryLimitInGB))
	model.DatasetSizeInGB = types.Float64Value(redis.Float64Value(db.DatasetSizeInGB))
	model.SupportOSSClusterAPI = types.BoolValue(redis.BoolValue(db.SupportOSSClusterAPI))
	model.RespVersion = types.StringValue(redis.StringValue(db.RespVersion))
	model.DataPersistence = types.StringValue(redis.StringValue(db.DataPersistence))
	model.DataEviction = types.StringValue(redis.StringValue(db.DataEvictionPolicy))
	model.Replication = types.BoolValue(redis.BoolValue(db.Replication))
	model.PublicEndpoint = types.StringValue(redis.StringValue(db.PublicEndpoint))
	model.PrivateEndpoint = types.StringValue(redis.StringValue(db.PrivateEndpoint))
	model.QueryPerformanceFactor = types.StringPointerValue(db.QueryPerformanceFactor)
	model.RedisVersion = types.StringPointerValue(db.RedisVersion)

	model.ThroughputMeasurementBy = types.StringNull()
	model.ThroughputMeasurementValue = types.Int64Null()
	if db.ThroughputMeasurement != nil {
		model.ThroughputMeasurementBy = types.StringValue(redis.StringValue(db.ThroughputMeasurement.By))
		model.ThroughputMeasurementValue = types.Int64Value(int64(redis.IntValue(db.ThroughputMeasurement.Value)))
	}

	// An explicitly empty (non-nil) password on a redis-protocol database means
	// passwordless mode. A nil password means "not applicable".
	passwordless := false
	model.Password = types.StringNull()
	model.EnableDefaultUser = types.BoolNull()
	model.SourceIPs = types.SetNull(types.StringType)
	if db.Security != nil {
		model.Password = types.StringValue(redis.StringValue(db.Security.Password))
		passwordless = redis.StringValue(db.Protocol) == "redis" && db.Security.Password != nil && *db.Security.Password == ""
		model.EnableDefaultUser = types.BoolValue(redis.BoolValue(db.Security.EnableDefaultUser))

		sourceIPs, diags := types.SetValueFrom(ctx, types.StringType, redis.StringSliceValue(db.Security.SourceIPs...))
		resp.Diagnostics.Append(diags...)
		model.SourceIPs = sourceIPs
	}
	model.EnablePasswordless = types.BoolValue(passwordless)

	model.ReplicaOf = types.SetNull(types.StringType)
	if db.ReplicaOf != nil {
		replicaOf, diags := types.SetValueFrom(ctx, types.StringType, redis.StringSliceValue(db.ReplicaOf.Endpoints...))
		resp.Diagnostics.Append(diags...)
		model.ReplicaOf = replicaOf
	}

	model.HashingPolicy = types.ListNull(types.StringType)
	if db.Clustering != nil {
		hashingPolicy, diags := types.ListValueFrom(ctx, types.StringType, FlattenRegexRules(db.Clustering.RegexRules))
		resp.Diagnostics.Append(diags...)
		model.HashingPolicy = hashingPolicy
	}

	alerts, diags := DatabaseAlertsFromAPI(ctx, db.Alerts)
	resp.Diagnostics.Append(diags...)
	model.Alert = alerts

	modules, diags := DatabaseModulesFromAPI(ctx, db.Modules)
	resp.Diagnostics.Append(diags...)
	model.Module = modules

	autoMinorVersionUpgrade := true
	if db.AutoMinorVersionUpgrade != nil {
		autoMinorVersionUpgrade = redis.BoolValue(db.AutoMinorVersionUpgrade)
	}
	model.AutoMinorVersionUpgrade = types.BoolValue(autoMinorVersionUpgrade)

	// Lookup errors are forgiven: a latest status is sometimes unavailable.
	latestBackupStatus, err := d.client.Client.LatestBackup.Get(ctx, subId, dbId)
	if err != nil {
		latestBackupStatus = nil
	}
	backupStatus, diags := LatestBackupStatusFromAPI(ctx, latestBackupStatus)
	resp.Diagnostics.Append(diags...)
	model.LatestBackupStatus = backupStatus

	latestImportStatus, err := d.client.Client.LatestImport.Get(ctx, subId, dbId)
	if err != nil {
		latestImportStatus = nil
	}
	importStatus, diags := LatestImportStatusFromAPI(ctx, latestImportStatus)
	resp.Diagnostics.Append(diags...)
	model.LatestImportStatus = importStatus

	tagResponse, err := d.client.Client.Tags.Get(ctx, subId, dbId)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Pro Database Tags", err.Error())
		return
	}
	dbTags, diags := DatabaseTagsFromAPI(ctx, tagResponse)
	resp.Diagnostics.Append(diags...)
	model.Tags = dbTags

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// proDatabaseFilters keeps pro databases only, since Active-Active databases come
// from the same endpoint. Zero or empty filters are ignored, as SDKv2 GetOk did.
func proDatabaseFilters(model ProDatabaseDataSourceModel) []func(db *databases.Database) bool {
	filters := []func(db *databases.Database) bool{
		func(db *databases.Database) bool {
			return !redis.BoolValue(db.ActiveActiveRedis)
		},
	}

	if v := model.DbID.ValueInt64(); v != 0 {
		filters = append(filters, func(db *databases.Database) bool {
			return int64(redis.IntValue(db.ID)) == v
		})
	}
	if v := model.Name.ValueString(); v != "" {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Name) == v
		})
	}
	if v := model.Protocol.ValueString(); v != "" {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Protocol) == v
		})
	}
	if v := model.Region.ValueString(); v != "" {
		filters = append(filters, func(db *databases.Database) bool {
			return redis.StringValue(db.Region) == v
		})
	}
	return filters
}

func filterProDatabases(list *databases.ListDatabase, filters []func(db *databases.Database) bool) ([]*databases.Database, error) {
	var filtered []*databases.Database
	for list.Next() {
		if filterProDatabase(list.Value(), filters) {
			filtered = append(filtered, list.Value())
		}
	}
	if list.Err() != nil {
		return nil, list.Err()
	}

	return filtered, nil
}

func filterProDatabase(db *databases.Database, filters []func(db *databases.Database) bool) bool {
	for _, filter := range filters {
		if !filter(db) {
			return false
		}
	}
	return true
}
