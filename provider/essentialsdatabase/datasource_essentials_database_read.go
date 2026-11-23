package essentialsdatabase

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// Read refreshes the Terraform state with the latest data.
func (d *essentialsDatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	if d.client == nil {
		resp.Diagnostics.AddError(
			"Provider Not Configured",
			"The provider client is not configured. This is an internal error - please report this to the provider developers.",
		)
		return
	}

	var model EssentialsDatabaseDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId := int(model.SubscriptionID.ValueInt64())

	var filters []databaseFilter

	// SDKv2 GetOk treated db_id = 0 and name = "" as unset; keep that so existing configs behave the same.
	if utils.IsConfigured(model.DbID) && model.DbID.ValueInt64() != 0 {
		dbId := int(model.DbID.ValueInt64())
		filters = append(filters, func(db *fixedDatabases.FixedDatabase) bool {
			return redis.IntValue(db.DatabaseId) == dbId
		})
	}

	if utils.IsConfigured(model.Name) && model.Name.ValueString() != "" {
		name := model.Name.ValueString()
		filters = append(filters, func(db *fixedDatabases.FixedDatabase) bool {
			return redis.StringValue(db.Name) == name
		})
	}

	dbs, err := filterFixedDatabases(d.client.Client.FixedDatabases.List(ctx, subId), filters)
	if err != nil {
		resp.Diagnostics.AddError("Unable to List Essentials Databases", err.Error())
		return
	}

	if len(dbs) == 0 {
		resp.Diagnostics.AddError(
			"No Essentials Databases Found",
			"Your query returned no results. Please change your search criteria and try again.",
		)
		return
	}

	if len(dbs) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Essentials Databases Found",
			"Your query returned more than one result. Please change try a more specific search criteria and try again.",
		)
		return
	}

	// Some attributes are only returned when retrieving a single database.
	db, err := d.client.Client.FixedDatabases.Get(ctx, subId, redis.IntValue(dbs[0].DatabaseId))
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Essentials Database", err.Error())
		return
	}

	resp.Diagnostics.Append(d.databaseIntoModel(ctx, subId, db, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (d *essentialsDatabaseDataSource) databaseIntoModel(ctx context.Context, subId int, db *fixedDatabases.FixedDatabase, model *EssentialsDatabaseDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	dbId := redis.IntValue(db.DatabaseId)

	model.ID = types.StringValue(utils.BuildResourceId(subId, dbId))
	model.DbID = types.Int64Value(int64(dbId))
	model.Name = types.StringValue(redis.StringValue(db.Name))
	model.Protocol = types.StringValue(redis.StringValue(db.Protocol))
	model.CloudProvider = types.StringValue(redis.StringValue(db.Provider))
	model.Region = types.StringValue(redis.StringValue(db.Region))
	model.RedisVersion = types.StringValue(redis.StringValue(db.RedisVersion))
	model.RedisVersionCompliance = types.StringValue(redis.StringValue(db.RedisVersionCompliance))
	model.RespVersion = types.StringValue(redis.StringValue(db.RespVersion))
	model.DataPersistence = types.StringValue(redis.StringValue(db.DataPersistence))
	model.DataEviction = types.StringValue(redis.StringValue(db.DataEvictionPolicy))
	model.Replication = types.BoolValue(redis.BoolValue(db.Replication))
	model.ActivatedOn = activatedOnFromAPI(db)
	model.PeriodicBackupPath = types.StringNull()
	if db.Backup != nil && redis.BoolValue(db.Backup.Enabled) {
		model.PeriodicBackupPath = types.StringValue(redis.StringValue(db.Backup.Destination))
	}
	model.PublicEndpoint = types.StringValue(redis.StringValue(db.PublicEndpoint))
	model.PrivateEndpoint = types.StringValue(redis.StringValue(db.PrivateEndpoint))

	sourceIPs, d2 := types.SetValueFrom(ctx, types.StringType, sourceIPsFromAPI(db.Security))
	diags.Append(d2...)
	model.SourceIPs = sourceIPs

	replicas, d2 := replicaModelsFromAPI(ctx, db.Replica)
	diags.Append(d2...)
	model.Replica, d2 = types.SetValueFrom(ctx, replicaElemType, replicas)
	diags.Append(d2...)

	// Client TLS certificates are not returned by the API.
	model.ClientTLSCertificates = types.ListNull(types.StringType)

	model.Password = types.StringValue("")
	model.EnableDefaultUser = types.BoolValue(false)
	model.EnableTLS = types.BoolValue(false)
	if db.Security != nil {
		model.Password = types.StringValue(redis.StringValue(db.Security.Password))
		model.EnableDefaultUser = types.BoolValue(redis.BoolValue(db.Security.EnableDefaultUser))
		model.EnableTLS = types.BoolValue(redis.BoolValue(db.Security.EnableTls))
	}

	model.Alert, d2 = AlertsFromAPI(ctx, db.Alerts)
	diags.Append(d2...)
	model.Modules, d2 = ModulesFromAPI(ctx, db.Modules)
	diags.Append(d2...)

	// Errors are forgiven: the latest status is not always available.
	latestBackup, err := d.client.Client.LatestBackup.GetFixed(ctx, subId, dbId)
	if err != nil {
		latestBackup = nil
	}
	model.LatestBackupStatus, d2 = latestBackupStatusFromAPI(ctx, latestBackup)
	diags.Append(d2...)

	latestImport, err := d.client.Client.LatestImport.GetFixed(ctx, subId, dbId)
	if err != nil {
		latestImport = nil
	}
	model.LatestImportStatus, d2 = latestImportStatusFromAPI(ctx, latestImport)
	diags.Append(d2...)

	model.MemoryLimitInGB = types.Float64Value(redis.Float64Value(db.MemoryLimitInGb))
	model.SupportOSSClusterAPI = types.BoolValue(redis.BoolValue(db.SupportOSSClusterAPI))
	model.ExternalEndpointForOSSClusterAPI = types.BoolValue(redis.BoolValue(db.UseExternalEndpointForOSSClusterAPI))
	model.EnableDatabaseClustering = types.BoolValue(db.Clustering != nil && redis.BoolValue(db.Clustering.Enabled))
	model.RegexRules, d2 = regexRulesFromAPI(ctx, db.Clustering)
	diags.Append(d2...)

	model.Tags, d2 = tagsFromAPI(ctx, d.client, subId, dbId)
	diags.Append(d2...)

	return diags
}

func activatedOnFromAPI(db *fixedDatabases.FixedDatabase) types.String {
	if db.ActivatedOn == nil {
		return types.StringValue("")
	}
	return types.StringValue(db.ActivatedOn.String())
}
