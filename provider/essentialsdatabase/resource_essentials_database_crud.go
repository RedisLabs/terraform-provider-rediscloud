package essentialsdatabase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	fixedDatabases "github.com/RedisLabs/rediscloud-go-api/service/fixed/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentialssubscription"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

const (
	createTimeout = 30 * time.Minute
	readTimeout   = 10 * time.Minute
	updateTimeout = 30 * time.Minute
	deleteTimeout = 10 * time.Minute
)

// Create creates the database, then applies the attributes only the update endpoint accepts (enable_default_user, tags).
func (r *essentialsDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EssentialsDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, createTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId := int(plan.SubscriptionID.ValueInt64())

	request, diags := createRequestFromModel(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.SubscriptionMutex.Lock(subId)
	dbId, err := r.client.Client.FixedDatabases.Create(ctx, subId, request)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		resp.Diagnostics.AddError("Unable to Create Essentials Database", err.Error())
		return
	}

	plan.ID = types.StringValue(utils.BuildResourceId(subId, dbId))
	plan.DbID = types.Int64Value(int64(dbId))

	err = waitForEssentialsDatabaseToBeActive(ctx, subId, dbId, r.client, timeout)
	utils.SubscriptionMutex.Unlock(subId)
	if err != nil {
		r.persistCreated(ctx, &plan, resp)
		resp.Diagnostics.AddError("Essentials Database Failed to Become Active", err.Error())
		return
	}

	if diags := r.update(ctx, subId, dbId, &plan, nil, timeout); diags.HasError() {
		r.persistCreated(ctx, &plan, resp)
		resp.Diagnostics.Append(diags...)
		return
	}

	notFound, diags := r.readIntoModel(ctx, subId, dbId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.Diagnostics.AddError("Essentials Database Not Found", "The essentials database was not found immediately after creation.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// persistCreated stores a database that exists but failed to settle, so it is tracked (and tainted) rather than leaked,
// as SDKv2 did by setting the ID before waiting.
func (r *essentialsDatabaseResource) persistCreated(ctx context.Context, plan *EssentialsDatabaseResourceModel, resp *resource.CreateResponse) {
	resolveUnknowns(plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *essentialsDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EssentialsDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, readTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	subId, dbId, err := pro.ToDatabaseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Essentials Database ID", err.Error())
		return
	}
	if subId == 0 {
		subId = int(state.SubscriptionID.ValueInt64())
	}

	notFound, diags := r.readIntoModel(ctx, subId, dbId, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update upgrades the Redis version when requested, then updates the database and its tags.
func (r *essentialsDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state EssentialsDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, updateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, dbId, err := pro.ToDatabaseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Essentials Database ID", err.Error())
		return
	}
	subId := int(plan.SubscriptionID.ValueInt64())

	resp.Diagnostics.Append(r.update(ctx, subId, dbId, &plan, &state, timeout)...)
	if resp.Diagnostics.HasError() {
		return
	}

	notFound, diags := r.readIntoModel(ctx, subId, dbId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the database once it is active.
func (r *essentialsDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EssentialsDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, deleteTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, dbId, err := pro.ToDatabaseId(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid Essentials Database ID", err.Error())
		return
	}
	subId := int(state.SubscriptionID.ValueInt64())

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := waitForEssentialsDatabaseToBeActive(ctx, subId, dbId, r.client, timeout); err != nil {
		resp.Diagnostics.AddError("Essentials Database Failed to Become Active", err.Error())
		return
	}

	if err := r.client.Client.FixedDatabases.Delete(ctx, subId, dbId); err != nil {
		resp.Diagnostics.AddError("Unable to Delete Essentials Database", err.Error())
	}
}

// ImportState imports an existing database using the `subscription_id/db_id` format.
func (r *essentialsDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	subId, dbId, err := pro.ToDatabaseId(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected an import ID in the format subscription_id/db_id, got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), utils.BuildResourceId(subId, dbId))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("subscription_id"), int64(subId))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("db_id"), int64(dbId))...)
}

// update mirrors the SDKv2 update function, which Create also called. prior is nil during Create.
func (r *essentialsDatabaseResource) update(ctx context.Context, subId, dbId int, plan, prior *EssentialsDatabaseResourceModel, timeout time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if prior != nil && redisVersionUpgradeRequested(prior, plan) {
		diags.Append(r.upgradeRedisVersion(ctx, subId, dbId, plan.RedisVersion.ValueString(), timeout)...)
		if diags.HasError() {
			return diags
		}
	}

	request, d := updateRequestFromModel(ctx, plan, prior)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	if err := r.client.Client.FixedDatabases.Update(ctx, subId, dbId, request); err != nil {
		diags.AddError("Unable to Update Essentials Database", err.Error())
		return diags
	}

	if err := waitForEssentialsDatabaseToBeActive(ctx, subId, dbId, r.client, timeout); err != nil {
		diags.AddError("Essentials Database Failed to Become Active", err.Error())
		return diags
	}

	if err := essentialssubscription.WaitForEssentialsSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		diags.AddError("Essentials Subscription Failed to Become Active", err.Error())
		return diags
	}

	// The Tags API is synchronous, so no further wait is needed.
	tagList, d := tagsRequestFromModel(ctx, plan.Tags)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	if err := r.client.Client.Tags.PutFixed(ctx, subId, dbId, tags.AllTags{Tags: &tagList}); err != nil {
		diags.AddError("Unable to Update Essentials Database Tags", err.Error())
	}

	return diags
}

// redisVersionUpgradeRequested treats redis_version as a minimum: no upgrade when the running version already satisfies it.
func redisVersionUpgradeRequested(prior, plan *EssentialsDatabaseResourceModel) bool {
	oldVersion := prior.RedisVersion.ValueString()
	newVersion := plan.RedisVersion.ValueString()
	actual := prior.RedisVersionActual.ValueString()
	return oldVersion != "" && newVersion != "" && oldVersion != newVersion &&
		actual != newVersion && !utils.RedisVersionSatisfied(newVersion, actual)
}

func (r *essentialsDatabaseResource) upgradeRedisVersion(ctx context.Context, subId, dbId int, newVersion string, timeout time.Duration) diag.Diagnostics {
	var diags diag.Diagnostics
	log.Printf("[INFO] Requesting Redis version change to %s...", newVersion)

	upgrade := fixedDatabases.UpgradeRedisVersion{TargetRedisVersion: redis.String(newVersion)}
	if err := r.client.Client.FixedDatabases.UpgradeRedisVersion(ctx, subId, dbId, upgrade); err != nil {
		diags.AddError("Unable to Upgrade Redis Version", fmt.Sprintf("failed to change Redis version to %s: %v", newVersion, err))
		return diags
	}

	log.Printf("[INFO] Redis version change request to %s accepted by API", newVersion)

	if err := waitForEssentialsDatabaseToBeActive(ctx, subId, dbId, r.client, timeout); err != nil {
		diags.AddError("Essentials Database Failed to Become Active", err.Error())
		return diags
	}

	if err := essentialssubscription.WaitForEssentialsSubscriptionToBeActive(ctx, subId, r.client, timeout); err != nil {
		diags.AddError("Essentials Subscription Failed to Become Active", err.Error())
	}

	return diags
}

// readIntoModel maps the API database onto the model and reports whether it no longer exists.
func (r *essentialsDatabaseResource) readIntoModel(ctx context.Context, subId, dbId int, model *EssentialsDatabaseResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	db, err := r.client.Client.FixedDatabases.Get(ctx, subId, dbId)
	if err != nil {
		notFound := &fixedDatabases.NotFound{}
		if errors.As(err, &notFound) {
			return true, diags
		}
		diags.AddError("Unable to Read Essentials Database", err.Error())
		return false, diags
	}

	// enable_payg_features is only null right after an import: every value then comes from the API.
	importing := model.EnablePaygFeatures.IsNull()
	keepPaygValues := !importing && !model.EnablePaygFeatures.ValueBool()
	if importing {
		model.EnablePaygFeatures = types.BoolValue(false)
	}

	model.ID = types.StringValue(utils.BuildResourceId(subId, dbId))
	model.SubscriptionID = types.Int64Value(int64(subId))
	model.DbID = types.Int64Value(int64(redis.IntValue(db.DatabaseId)))
	model.Name = types.StringValue(redis.StringValue(db.Name))
	model.Protocol = types.StringValue(redis.StringValue(db.Protocol))

	if db.RedisVersion != nil {
		model.RedisVersionActual = types.StringValue(redis.StringValue(db.RedisVersion))
		// redis_version is the requested minimum: only fill it when it was never set.
		if model.RedisVersion.IsNull() || model.RedisVersion.IsUnknown() || model.RedisVersion.ValueString() == "" {
			model.RedisVersion = types.StringValue(redis.StringValue(db.RedisVersion))
		}
	}
	if model.RedisVersionActual.IsUnknown() {
		model.RedisVersionActual = types.StringNull()
	}
	if model.RedisVersion.IsUnknown() {
		model.RedisVersion = types.StringNull()
	}

	model.CloudProvider = types.StringValue(redis.StringValue(db.Provider))
	model.Region = types.StringValue(redis.StringValue(db.Region))
	model.RedisVersionCompliance = types.StringValue(redis.StringValue(db.RedisVersionCompliance))
	model.RespVersion = types.StringValue(redis.StringValue(db.RespVersion))
	model.DataPersistence = types.StringValue(redis.StringValue(db.DataPersistence))
	model.DataEviction = types.StringValue(redis.StringValue(db.DataEvictionPolicy))
	model.Replication = types.BoolValue(redis.BoolValue(db.Replication))
	model.ActivatedOn = activatedOnFromAPI(db)
	// The backup path is only returned through the backup settings, when backups are enabled.
	if db.Backup != nil && redis.BoolValue(db.Backup.Enabled) {
		model.PeriodicBackupPath = types.StringValue(redis.StringValue(db.Backup.Destination))
	}
	model.PublicEndpoint = types.StringValue(redis.StringValue(db.PublicEndpoint))
	model.PrivateEndpoint = types.StringValue(redis.StringValue(db.PrivateEndpoint))

	if !keepPaygValues || model.SourceIPs.IsUnknown() {
		model.SourceIPs, diags = sourceIPsIntoModel(ctx, db.Security, model.SourceIPs)
		if diags.HasError() {
			return false, diags
		}
	}

	replica, d := replicaIntoModel(ctx, db.Replica, model.Replica)
	diags.Append(d...)
	model.Replica = replica

	// Client TLS certificates are not returned by the API.
	if model.ClientTLSCertificates.IsUnknown() {
		model.ClientTLSCertificates = types.ListNull(types.StringType)
	}

	model.Password = types.StringValue("")
	model.EnableDefaultUser = types.BoolValue(false)
	enableTLS := false
	if db.Security != nil {
		model.Password = types.StringValue(redis.StringValue(db.Security.Password))
		model.EnableDefaultUser = types.BoolValue(redis.BoolValue(db.Security.EnableDefaultUser))
		enableTLS = redis.BoolValue(db.Security.EnableTls)
	}

	model.Alert, d = AlertsFromAPI(ctx, db.Alerts)
	diags.Append(d...)
	model.Modules, d = ModulesFromAPI(ctx, db.Modules)
	diags.Append(d...)

	keep := func(v attr.Value) bool {
		return keepPaygValues && !v.IsNull() && !v.IsUnknown()
	}
	if !keep(model.MemoryLimitInGB) {
		model.MemoryLimitInGB = types.Float64Value(redis.Float64Value(db.MemoryLimitInGb))
	}
	if !keep(model.SupportOSSClusterAPI) {
		model.SupportOSSClusterAPI = types.BoolValue(redis.BoolValue(db.SupportOSSClusterAPI))
	}
	if !keep(model.ExternalEndpointForOSSClusterAPI) {
		model.ExternalEndpointForOSSClusterAPI = types.BoolValue(redis.BoolValue(db.UseExternalEndpointForOSSClusterAPI))
	}
	if !keep(model.EnableDatabaseClustering) {
		model.EnableDatabaseClustering = types.BoolValue(db.Clustering != nil && redis.BoolValue(db.Clustering.Enabled))
	}
	if !keep(model.RegexRules) {
		model.RegexRules, d = regexRulesFromAPI(ctx, db.Clustering)
		diags.Append(d...)
	}
	if !keep(model.EnableTLS) {
		model.EnableTLS = types.BoolValue(enableTLS)
	}

	apiTags, d := tagsFromAPI(ctx, r.client, subId, dbId)
	diags.Append(d...)
	if diags.HasError() {
		return false, diags
	}
	model.Tags = tagsIntoModel(apiTags, model.Tags)

	return false, diags
}

// sourceIPsIntoModel keeps an empty or allow-all prior value when the API reports no restriction,
// because the API stores both an empty request and ["0.0.0.0/0"] as the allow-all rule.
func sourceIPsIntoModel(ctx context.Context, security *fixedDatabases.Security, prior types.List) (types.List, diag.Diagnostics) {
	sourceIPs := sourceIPsFromAPI(security)
	if len(sourceIPs) > 0 {
		return types.ListValueFrom(ctx, types.StringType, sourceIPs)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorIPs []string
		diags := prior.ElementsAs(ctx, &priorIPs, false)
		if diags.HasError() {
			return prior, diags
		}
		if len(priorIPs) == 0 || (len(priorIPs) == 1 && priorIPs[0] == defaultSourceIP) {
			return prior, nil
		}
	}
	return types.ListNull(types.StringType), nil
}

// replicaIntoModel keeps an unset encryption or server_cert unset when the API returns its zero value.
func replicaIntoModel(ctx context.Context, replica *fixedDatabases.ReplicaOf, prior types.List) (types.List, diag.Diagnostics) {
	if replica == nil {
		return types.ListValueFrom(ctx, replicaElemType, []ReplicaModel{})
	}

	priorByEndpoint := map[string]SyncSourceModel{}
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorReplicas []ReplicaModel
		if diags := prior.ElementsAs(ctx, &priorReplicas, false); diags.HasError() {
			return prior, diags
		}
		for _, priorReplica := range priorReplicas {
			var priorSources []SyncSourceModel
			if diags := priorReplica.SyncSource.ElementsAs(ctx, &priorSources, false); diags.HasError() {
				return prior, diags
			}
			for _, source := range priorSources {
				priorByEndpoint[source.Endpoint.ValueString()] = source
			}
		}
	}

	syncSources := syncSourceModelsFromAPI(replica)
	for i, source := range syncSources {
		priorSource, ok := priorByEndpoint[source.Endpoint.ValueString()]
		if !ok {
			continue
		}
		if priorSource.Encryption.IsNull() && !source.Encryption.ValueBool() {
			syncSources[i].Encryption = types.BoolNull()
		}
		if priorSource.ServerCert.IsNull() && source.ServerCert.ValueString() == "" {
			syncSources[i].ServerCert = types.StringNull()
		}
	}

	syncSourceSet, diags := types.SetValueFrom(ctx, syncSourceElemType, syncSources)
	if diags.HasError() {
		return prior, diags
	}
	return types.ListValueFrom(ctx, replicaElemType, []ReplicaModel{{SyncSource: syncSourceSet}})
}

func tagsIntoModel(apiTags, prior types.Map) types.Map {
	if len(apiTags.Elements()) > 0 {
		return apiTags
	}
	if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
		return prior
	}
	return types.MapNull(types.StringType)
}

func createRequestFromModel(ctx context.Context, plan *EssentialsDatabaseResourceModel) (fixedDatabases.CreateFixedDatabase, diag.Diagnostics) {
	var diags diag.Diagnostics

	request := fixedDatabases.CreateFixedDatabase{
		Name:               plan.Name.ValueStringPointer(),
		DataPersistence:    plan.DataPersistence.ValueStringPointer(),
		DataEvictionPolicy: plan.DataEviction.ValueStringPointer(),
		Replication:        plan.Replication.ValueBoolPointer(),
		PeriodicBackupPath: redis.String(plan.PeriodicBackupPath.ValueString()),
	}

	if v := plan.Protocol.ValueString(); v != "" {
		request.Protocol = redis.String(v)
	}
	if v := plan.RedisVersion.ValueString(); v != "" {
		request.RedisVersion = redis.String(v)
	}
	if v := plan.RespVersion.ValueString(); v != "" {
		request.RespVersion = redis.String(v)
	}

	var d diag.Diagnostics
	request.SourceIPs, d = sourceIPsRequest(ctx, plan.SourceIPs)
	diags.Append(d...)
	request.Replica, d = replicaRequest(ctx, plan.Replica)
	diags.Append(d...)
	request.ClientTlsCertificates, d = tlsCertificatesRequest(ctx, plan.ClientTLSCertificates)
	diags.Append(d...)

	if v := plan.Password.ValueString(); v != "" {
		request.Password = redis.String(v)
	}

	request.Alerts, d = alertsRequest(ctx, plan.Alert)
	diags.Append(d...)
	request.Modules, d = modulesRequest(ctx, plan.Modules)
	diags.Append(d...)

	if plan.EnablePaygFeatures.ValueBool() {
		request.MemoryLimitInGB = redis.Float64(plan.MemoryLimitInGB.ValueFloat64())
		request.SupportOSSClusterAPI = redis.Bool(plan.SupportOSSClusterAPI.ValueBool())
		request.UseExternalEndpointForOSSClusterAPI = redis.Bool(plan.ExternalEndpointForOSSClusterAPI.ValueBool())
		request.EnableDatabaseClustering = redis.Bool(plan.EnableDatabaseClustering.ValueBool())
		request.RegexRules, d = stringsRequest(ctx, plan.RegexRules)
		diags.Append(d...)
		request.EnableTls = redis.Bool(plan.EnableTLS.ValueBool())
	}

	return request, diags
}

// updateRequestFromModel builds the update request. While Pay-As-You-Go features are disabled, SDKv2 suppressed the
// source_ips diff and therefore sent the prior value, so the prior value is sent here too.
func updateRequestFromModel(ctx context.Context, plan, prior *EssentialsDatabaseResourceModel) (fixedDatabases.UpdateFixedDatabase, diag.Diagnostics) {
	var diags diag.Diagnostics
	payg := plan.EnablePaygFeatures.ValueBool()

	request := fixedDatabases.UpdateFixedDatabase{
		Name:               plan.Name.ValueStringPointer(),
		DataPersistence:    plan.DataPersistence.ValueStringPointer(),
		DataEvictionPolicy: plan.DataEviction.ValueStringPointer(),
		Replication:        plan.Replication.ValueBoolPointer(),
		PeriodicBackupPath: redis.String(plan.PeriodicBackupPath.ValueString()),
		EnableDefaultUser:  plan.EnableDefaultUser.ValueBoolPointer(),
	}

	if v := plan.RespVersion.ValueString(); v != "" {
		request.RespVersion = redis.String(v)
	}

	sourceIPs := plan.SourceIPs
	if prior != nil && !payg {
		sourceIPs = prior.SourceIPs
	}
	var d diag.Diagnostics
	request.SourceIPs, d = sourceIPsRequest(ctx, sourceIPs)
	diags.Append(d...)
	request.Replica, d = replicaRequest(ctx, plan.Replica)
	diags.Append(d...)
	request.ClientTlsCertificates, d = tlsCertificatesRequest(ctx, plan.ClientTLSCertificates)
	diags.Append(d...)

	// The password cannot be updated while the default user is disabled.
	if v := plan.Password.ValueString(); v != "" && plan.EnableDefaultUser.ValueBool() {
		request.Password = redis.String(v)
	}

	request.Alerts, d = alertsRequest(ctx, plan.Alert)
	diags.Append(d...)

	if payg {
		request.MemoryLimitInGB = redis.Float64(plan.MemoryLimitInGB.ValueFloat64())
		request.SupportOSSClusterAPI = redis.Bool(plan.SupportOSSClusterAPI.ValueBool())
		request.UseExternalEndpointForOSSClusterAPI = redis.Bool(plan.ExternalEndpointForOSSClusterAPI.ValueBool())
		request.EnableDatabaseClustering = redis.Bool(plan.EnableDatabaseClustering.ValueBool())
		regexRules := plan.RegexRules
		if regexRules.IsUnknown() && prior != nil {
			regexRules = prior.RegexRules
		}
		request.RegexRules, d = stringsRequest(ctx, regexRules)
		diags.Append(d...)
		request.EnableTls = redis.Bool(plan.EnableTLS.ValueBool())
	}

	return request, diags
}

func stringsRequest(ctx context.Context, list types.List) ([]*string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var values []string
	diags := list.ElementsAs(ctx, &values, false)
	return redis.StringSlice(values...), diags
}

func sourceIPsRequest(ctx context.Context, list types.List) ([]*string, diag.Diagnostics) {
	sourceIPs, diags := stringsRequest(ctx, list)
	if len(sourceIPs) == 0 {
		return []*string{redis.String(defaultSourceIP)}, diags
	}
	return sourceIPs, diags
}

func tlsCertificatesRequest(ctx context.Context, list types.List) ([]*fixedDatabases.DatabaseCertificate, diag.Diagnostics) {
	certs, diags := stringsRequest(ctx, list)
	if len(certs) == 0 {
		return nil, diags
	}
	certificates := make([]*fixedDatabases.DatabaseCertificate, 0, len(certs))
	for _, cert := range certs {
		certificates = append(certificates, &fixedDatabases.DatabaseCertificate{PublicCertificatePEMString: cert})
	}
	return certificates, diags
}

func replicaRequest(ctx context.Context, list types.List) (*fixedDatabases.ReplicaOf, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var replicas []ReplicaModel
	diags := list.ElementsAs(ctx, &replicas, false)
	if diags.HasError() || len(replicas) != 1 {
		return nil, diags
	}

	var sources []SyncSourceModel
	diags.Append(replicas[0].SyncSource.ElementsAs(ctx, &sources, false)...)
	syncSources := make([]*fixedDatabases.SyncSource, 0, len(sources))
	for _, source := range sources {
		syncSources = append(syncSources, &fixedDatabases.SyncSource{
			Endpoint:   redis.String(source.Endpoint.ValueString()),
			Encryption: redis.Bool(source.Encryption.ValueBool()),
			ServerCert: redis.String(source.ServerCert.ValueString()),
		})
	}
	return &fixedDatabases.ReplicaOf{SyncSources: syncSources}, diags
}

func alertsRequest(ctx context.Context, set types.Set) (*[]*databases.Alert, diag.Diagnostics) {
	alerts := make([]*databases.Alert, 0)
	if set.IsNull() || set.IsUnknown() {
		return &alerts, nil
	}
	var models []AlertModel
	diags := set.ElementsAs(ctx, &models, false)
	for _, alert := range models {
		alerts = append(alerts, &databases.Alert{
			Name:  redis.String(alert.Name.ValueString()),
			Value: redis.Int(int(alert.Value.ValueInt64())),
		})
	}
	return &alerts, diags
}

func modulesRequest(ctx context.Context, set types.Set) (*[]*databases.Module, diag.Diagnostics) {
	modules := make([]*databases.Module, 0)
	if set.IsNull() || set.IsUnknown() {
		return &modules, nil
	}
	var models []ModuleModel
	diags := set.ElementsAs(ctx, &models, false)
	for _, module := range models {
		modules = append(modules, &databases.Module{Name: redis.String(module.Name.ValueString())})
	}
	return &modules, diags
}

func tagsRequestFromModel(ctx context.Context, tagMap types.Map) ([]*tags.Tag, diag.Diagnostics) {
	tagList := make([]*tags.Tag, 0)
	if tagMap.IsNull() || tagMap.IsUnknown() {
		return tagList, nil
	}
	values := map[string]string{}
	diags := tagMap.ElementsAs(ctx, &values, false)
	for k, v := range values {
		tagList = append(tagList, &tags.Tag{Key: redis.String(k), Value: redis.String(v)})
	}
	return tagList, diags
}

func resolveUnknowns(model *EssentialsDatabaseResourceModel) {
	for _, s := range []*types.String{
		&model.Protocol, &model.RedisVersion, &model.RedisVersionActual, &model.CloudProvider, &model.Region,
		&model.RedisVersionCompliance, &model.RespVersion, &model.ActivatedOn, &model.PublicEndpoint,
		&model.PrivateEndpoint, &model.Password,
	} {
		if s.IsUnknown() {
			*s = types.StringNull()
		}
	}
	for _, b := range []*types.Bool{
		&model.SupportOSSClusterAPI, &model.ExternalEndpointForOSSClusterAPI, &model.EnableDatabaseClustering, &model.EnableTLS,
	} {
		if b.IsUnknown() {
			*b = types.BoolNull()
		}
	}
	if model.MemoryLimitInGB.IsUnknown() {
		model.MemoryLimitInGB = types.Float64Null()
	}
	for _, l := range []*types.List{&model.SourceIPs, &model.ClientTLSCertificates, &model.RegexRules} {
		if l.IsUnknown() {
			*l = types.ListNull(types.StringType)
		}
	}
	if model.Modules.IsUnknown() {
		model.Modules = types.SetNull(moduleElemType)
	}
	if model.Tags.IsUnknown() {
		model.Tags = types.MapNull(types.StringType)
	}
}

func waitForEssentialsDatabaseToBeActive(ctx context.Context, subId, id int, api *client.ApiClient, timeout time.Duration) error {
	wait := &retry.StateChangeConf{
		Delay: api.WaitDelay(30 * time.Second),
		Pending: []string{
			databases.StatusDraft,
			databases.StatusPending,
			databases.StatusActiveChangePending,
			databases.StatusRCPActiveChangeDraft,
			databases.StatusActiveChangeDraft,
			databases.StatusRCPDraft,
			databases.StatusRCPChangePending,
			databases.StatusProxyPolicyChangePending,
			databases.StatusProxyPolicyChangeDraft,
			databases.StatusDynamicEndpointsCreationPending,
			databases.StatusActiveUpgradePending,
		},
		Target:       []string{databases.StatusActive},
		Timeout:      timeout,
		PollInterval: api.WaitPollInterval(30 * time.Second),

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for fixed database %d to be active", id)

			database, err := api.Client.FixedDatabases.Get(ctx, subId, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(database.Status), redis.StringValue(database.Status), nil
		},
	}
	_, err := wait.WaitForStateContext(ctx)
	return err
}
