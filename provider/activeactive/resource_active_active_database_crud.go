package activeactive

import (
	"context"
	"log"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	redisTags "github.com/RedisLabs/rediscloud-go-api/service/tags"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// createDatabase implements the Create operation for the active-active database resource.
func (r *activeActiveDatabaseResource) createDatabase(ctx context.Context, plan *ActiveActiveDatabaseModel, diagnostics *diag.Diagnostics) {
	subId := int(plan.SubscriptionID.ValueInt64())

	// Acquire subscription mutex
	utils.SubscriptionMutex.Lock(subId)

	// Build alerts from plan
	alerts, diags := buildAlertsFromSet(ctx, plan.GlobalAlert)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		utils.SubscriptionMutex.Unlock(subId)
		return
	}

	// Build modules from plan
	modules, diags := listToStringSlice(ctx, plan.GlobalModules)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		utils.SubscriptionMutex.Unlock(subId)
		return
	}

	createModules := make([]*databases.Module, 0, len(modules))
	for _, module := range modules {
		createModules = append(createModules, &databases.Module{Name: redis.String(module)})
	}

	// Build source IPs from plan
	sourceIPs, diags := setToStringSlice(ctx, plan.GlobalSourceIPs)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		utils.SubscriptionMutex.Unlock(subId)
		return
	}

	// Get regions from subscription to set up local throughputs
	regions, err := r.client.Client.Regions.List(ctx, subId)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		diagnostics.AddError("Failed to get subscription regions", err.Error())
		return
	}

	localThroughputs := make([]*databases.LocalThroughput, 0, len(regions.Regions))
	for _, region := range regions.Regions {
		localThroughputs = append(localThroughputs, &databases.LocalThroughput{
			Region:                   redis.String(*region.Region),
			WriteOperationsPerSecond: redis.Int(1000),
			ReadOperationsPerSecond:  redis.Int(1000),
		})
	}

	// Build create request
	createDatabase := databases.CreateActiveActiveDatabase{
		DryRun:                              redis.Bool(false),
		Name:                                redis.String(plan.Name.ValueString()),
		SupportOSSClusterAPI:                redis.Bool(plan.SupportOssClusterAPI.ValueBool()),
		UseExternalEndpointForOSSClusterAPI: redis.Bool(plan.ExternalEndpointForOssClusterAPI.ValueBool()),
		GlobalSourceIP:                      stringPtrSlice(sourceIPs),
		GlobalAlerts:                        alerts,
		GlobalModules:                       createModules,
		LocalThroughputMeasurement:          localThroughputs,
	}

	// Set optional fields
	if !plan.DataEviction.IsNull() && plan.DataEviction.ValueString() != "" {
		createDatabase.DataEvictionPolicy = redis.String(plan.DataEviction.ValueString())
	}

	if !plan.GlobalDataPersistence.IsNull() && plan.GlobalDataPersistence.ValueString() != "" {
		createDatabase.GlobalDataPersistence = redis.String(plan.GlobalDataPersistence.ValueString())
	}

	if !plan.GlobalPassword.IsNull() && plan.GlobalPassword.ValueString() != "" {
		createDatabase.GlobalPassword = redis.String(plan.GlobalPassword.ValueString())
	}

	if !plan.DatasetSizeInGB.IsNull() && plan.DatasetSizeInGB.ValueFloat64() > 0 {
		createDatabase.DatasetSizeInGB = redis.Float64(plan.DatasetSizeInGB.ValueFloat64())
	}

	if !plan.MemoryLimitInGB.IsNull() && plan.MemoryLimitInGB.ValueFloat64() > 0 {
		createDatabase.MemoryLimitInGB = redis.Float64(plan.MemoryLimitInGB.ValueFloat64())
	}

	if !plan.Port.IsNull() && plan.Port.ValueInt64() > 0 {
		createDatabase.PortNumber = redis.Int(int(plan.Port.ValueInt64()))
	}

	if !plan.GlobalRespVersion.IsNull() && plan.GlobalRespVersion.ValueString() != "" {
		createDatabase.RespVersion = redis.String(plan.GlobalRespVersion.ValueString())
	}

	if !plan.RedisVersion.IsNull() && plan.RedisVersion.ValueString() != "" {
		createDatabase.RedisVersion = redis.String(plan.RedisVersion.ValueString())
	}

	// Wait for subscription to be active before creating database
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		diagnostics.AddError("Subscription not active", err.Error())
		return
	}

	// Create the database
	dbId, err := r.client.Client.Database.ActiveActiveCreate(ctx, subId, createDatabase)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		diagnostics.AddError("Failed to create database", err.Error())
		return
	}

	// Set the ID in plan
	plan.ID = types.StringValue(buildResourceId(subId, dbId))
	plan.DbID = types.Int64Value(int64(dbId))

	// Wait for database to be active
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, r.client); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		diagnostics.AddError("Database failed to become active", err.Error())
		return
	}

	// Wait for subscription to be active
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		diagnostics.AddError("Subscription failed to become active", err.Error())
		return
	}

	// Release mutex before update (update will acquire it again)
	utils.SubscriptionMutex.Unlock(subId)

	// Some attributes on a database are not accessible by the create API.
	// Run the update function to apply any additional changes.
	r.updateDatabase(ctx, plan, diagnostics)
	if diagnostics.HasError() {
		return
	}

	// Read back the state to get computed values
	r.readDatabase(ctx, plan, diagnostics)
}

// readDatabase implements the Read operation for the active-active database resource.
// Returns true if the resource was removed (not found).
func (r *activeActiveDatabaseResource) readDatabase(ctx context.Context, state *ActiveActiveDatabaseModel, diagnostics *diag.Diagnostics) bool {
	subId, dbId, err := parseResourceId(state.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid resource ID", err.Error())
		return false
	}

	// Handle case where subscription_id is 0 (not importing)
	if subId == 0 {
		subId = int(state.SubscriptionID.ValueInt64())
	}

	// Get the database from API
	db, err := r.client.Client.Database.GetActiveActive(ctx, subId, dbId)
	if err != nil {
		if _, ok := err.(*databases.NotFound); ok {
			log.Printf("[DEBUG] Database %d not found, removing from state", dbId)
			return true
		}
		diagnostics.AddError("Failed to read database", err.Error())
		return false
	}

	// Set basic fields
	state.SubscriptionID = types.Int64Value(int64(subId))
	state.DbID = types.Int64Value(int64(redis.IntValue(db.ID)))
	state.Name = types.StringValue(redis.StringValue(db.Name))
	state.DataEviction = types.StringValue(redis.StringValue(db.DataEvictionPolicy))
	state.SupportOssClusterAPI = types.BoolValue(redis.BoolValue(db.SupportOSSClusterAPI))
	state.ExternalEndpointForOssClusterAPI = types.BoolValue(redis.BoolValue(db.UseExternalEndpointForOSSClusterAPI))
	state.RedisVersion = types.StringValue(redis.StringValue(db.RedisVersion))

	// Set global_data_persistence if present
	if db.GlobalDataPersistence != nil {
		state.GlobalDataPersistence = types.StringValue(redis.StringValue(db.GlobalDataPersistence))
	}

	// Set global_password if present
	if db.GlobalPassword != nil {
		state.GlobalPassword = types.StringValue(redis.StringValue(db.GlobalPassword))
	}

	// Set global_enable_default_user if present
	if db.GlobalEnableDefaultUser != nil {
		state.GlobalEnableDefaultUser = types.BoolValue(redis.BoolValue(db.GlobalEnableDefaultUser))
	}

	// Handle global_source_ips - preserve user's custom value or compute defaults
	currentSourceIPs, diags := setToStringSlice(ctx, state.GlobalSourceIPs)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return false
	}

	var globalSourceIPs []string
	if !isDefaultGlobalSourceIPs(currentSourceIPs) {
		// User has a custom value - preserve it
		globalSourceIPs = currentSourceIPs
	} else {
		// No custom value - compute default based on subscription's public_endpoint_access
		if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client); err != nil {
			diagnostics.AddError("Failed to wait for subscription", err.Error())
			return false
		}
		subscription, err := r.client.Client.Subscription.Get(ctx, subId)
		if err != nil {
			diagnostics.AddError("Failed to get subscription", err.Error())
			return false
		}

		if subscription.PublicEndpointAccess != nil && !*subscription.PublicEndpointAccess {
			globalSourceIPs = defaultPrivateIPRanges
		} else {
			globalSourceIPs = []string{"0.0.0.0/0"}
		}
	}

	sourceIPSet, diags := stringSliceToSet(ctx, globalSourceIPs)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return false
	}
	state.GlobalSourceIPs = sourceIPSet

	// Set enable_tls from first region database
	if len(db.CrdbDatabases) > 0 && db.CrdbDatabases[0].Security != nil {
		state.EnableTLS = types.BoolValue(redis.BoolValue(db.CrdbDatabases[0].Security.EnableTls))
	}

	// Handle memory/dataset size - only set one based on what's in config
	// Must explicitly set unused field to Null (not Unknown) after apply
	if len(db.CrdbDatabases) > 0 {
		memorySetInConfig := utils.IsConfigured(state.MemoryLimitInGB) && state.MemoryLimitInGB.ValueFloat64() > 0
		datasetSetInConfig := utils.IsConfigured(state.DatasetSizeInGB) && state.DatasetSizeInGB.ValueFloat64() > 0

		if memorySetInConfig {
			if db.CrdbDatabases[0].MemoryLimitInGB != nil {
				state.MemoryLimitInGB = types.Float64Value(redis.Float64Value(db.CrdbDatabases[0].MemoryLimitInGB))
			}
			state.DatasetSizeInGB = types.Float64Null()
		} else if datasetSetInConfig || (!memorySetInConfig && !datasetSetInConfig) {
			if db.CrdbDatabases[0].DatasetSizeInGB != nil {
				state.DatasetSizeInGB = types.Float64Value(redis.Float64Value(db.CrdbDatabases[0].DatasetSizeInGB))
			}
			state.MemoryLimitInGB = types.Float64Null()
		}
	}

	// Build public and private endpoint maps
	publicEndpoints := make(map[string]string)
	privateEndpoints := make(map[string]string)
	for _, regionDb := range db.CrdbDatabases {
		region := redis.StringValue(regionDb.Region)
		publicEndpoints[region] = redis.StringValue(regionDb.PublicEndpoint)
		privateEndpoints[region] = redis.StringValue(regionDb.PrivateEndpoint)
	}

	publicEndpointMap, diags := stringMapToMap(ctx, publicEndpoints)
	diagnostics.Append(diags...)
	state.PublicEndpoint = publicEndpointMap

	privateEndpointMap, diags := stringMapToMap(ctx, privateEndpoints)
	diagnostics.Append(diags...)
	state.PrivateEndpoint = privateEndpointMap

	// Set modules
	modulesList, diags := stringSliceToList(ctx, flattenModulesToNames(db.Modules))
	diagnostics.Append(diags...)
	state.GlobalModules = modulesList

	// Build override_region from API response, but only for regions that are in state
	overrideRegionConfigs, diags := r.buildOverrideRegionFromAPI(ctx, db, state)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return false
	}
	state.OverrideRegion = overrideRegionConfigs

	// Read tags using the tags service
	tagResponse, err := r.client.Client.Tags.Get(ctx, subId, dbId)
	if err != nil {
		// Tags might not be available, log but don't fail
		log.Printf("[DEBUG] Failed to get tags for database %d: %v", dbId, err)
	} else if tagResponse != nil && tagResponse.Tags != nil {
		tagMap := make(map[string]string)
		for _, tag := range *tagResponse.Tags {
			tagMap[redis.StringValue(tag.Key)] = redis.StringValue(tag.Value)
		}
		tagsValue, diags := stringMapToMap(ctx, tagMap)
		diagnostics.Append(diags...)
		state.Tags = tagsValue
	}

	return false
}

// updateDatabase implements the Update operation for the active-active database resource.
func (r *activeActiveDatabaseResource) updateDatabase(ctx context.Context, plan *ActiveActiveDatabaseModel, diagnostics *diag.Diagnostics) {
	subId, dbId, err := parseResourceId(plan.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	// Acquire subscription mutex
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// Build alerts from plan
	alerts, diags := buildAlertsFromSet(ctx, plan.GlobalAlert)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	// Build source IPs from plan
	globalSourceIPs, diags := setToStringSlice(ctx, plan.GlobalSourceIPs)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	// Build regions from override_region
	regions, diags := r.buildRegionsFromPlan(ctx, plan)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	// Build update request
	update := databases.UpdateActiveActiveDatabase{
		GlobalAlerts:   &alerts,
		GlobalSourceIP: stringPtrSlice(globalSourceIPs),
		Regions:        regions,
	}

	// Set size fields (one must be set)
	if !plan.DatasetSizeInGB.IsNull() && plan.DatasetSizeInGB.ValueFloat64() > 0 {
		update.DatasetSizeInGB = redis.Float64(plan.DatasetSizeInGB.ValueFloat64())
	}

	if !plan.MemoryLimitInGB.IsNull() && plan.MemoryLimitInGB.ValueFloat64() > 0 {
		update.MemoryLimitInGB = redis.Float64(plan.MemoryLimitInGB.ValueFloat64())
	}

	// Handle global_source_ips defaults based on subscription's public_endpoint_access setting
	if isDefaultGlobalSourceIPs(globalSourceIPs) {
		subscription, err := r.client.Client.Subscription.Get(ctx, subId)
		if err != nil {
			diagnostics.AddError("Failed to get subscription", err.Error())
			return
		}

		if subscription.PublicEndpointAccess != nil && !*subscription.PublicEndpointAccess {
			update.GlobalSourceIP = stringPtrSlice(defaultPrivateIPRanges)
		} else {
			update.GlobalSourceIP = []*string{redis.String("0.0.0.0/0")}
		}
	}

	// Set global password
	if !plan.GlobalPassword.IsNull() && plan.GlobalPassword.ValueString() != "" {
		update.GlobalPassword = redis.String(plan.GlobalPassword.ValueString())
	}

	// Set global data persistence
	if !plan.GlobalDataPersistence.IsNull() && plan.GlobalDataPersistence.ValueString() != "" {
		update.GlobalDataPersistence = redis.String(plan.GlobalDataPersistence.ValueString())
	}

	// BUG FIX: Use direct value access instead of deprecated GetOkExists
	// Since the field has Default: true, we can safely use the value directly
	update.GlobalEnableDefaultUser = redis.Bool(plan.GlobalEnableDefaultUser.ValueBool())

	// Set OSS cluster API fields
	update.SupportOSSClusterAPI = redis.Bool(plan.SupportOssClusterAPI.ValueBool())
	update.UseExternalEndpointForOSSClusterAPI = redis.Bool(plan.ExternalEndpointForOssClusterAPI.ValueBool())

	// Set data eviction
	if !plan.DataEviction.IsNull() && plan.DataEviction.ValueString() != "" {
		update.DataEvictionPolicy = redis.String(plan.DataEviction.ValueString())
	}

	// Handle TLS configuration
	enableTLS := plan.EnableTLS.ValueBool()
	clientSSLCertificate := plan.ClientSSLCertificate.ValueString()
	clientTLSCertificates, diags := listToStringSlice(ctx, plan.ClientTLSCertificates)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	if enableTLS {
		update.EnableTls = redis.Bool(true)
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
			update.ClientTLSCertificates = &[]*string{}
		}
		if len(clientTLSCertificates) > 0 {
			tlsCerts := stringPtrSlice(clientTLSCertificates)
			update.ClientTLSCertificates = &tlsCerts
		}
	} else {
		if clientSSLCertificate != "" {
			// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else if len(clientTLSCertificates) > 0 {
			diagnostics.AddError("Invalid TLS configuration", "TLS certificates may not be provided while enable_tls is false")
			return
		} else {
			update.EnableTls = redis.Bool(false)
		}
	}

	// Execute update
	if err := r.client.Client.Database.ActiveActiveUpdate(ctx, subId, dbId, update); err != nil {
		diagnostics.AddError("Failed to update database", err.Error())
		return
	}

	// Wait for database to be active
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, r.client); err != nil {
		diagnostics.AddError("Database failed to become active after update", err.Error())
		return
	}

	// Wait for subscription to be active
	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, r.client); err != nil {
		diagnostics.AddError("Subscription failed to become active after update", err.Error())
		return
	}

	// Update tags using the tags service
	if !plan.Tags.IsNull() {
		tags, diags := mapToStringMap(ctx, plan.Tags)
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}

		var tagList []*redisTags.Tag
		for k, v := range tags {
			tagList = append(tagList, &redisTags.Tag{
				Key:   redis.String(k),
				Value: redis.String(v),
			})
		}

		if err := r.client.Client.Tags.Put(ctx, subId, dbId, redisTags.AllTags{Tags: &tagList}); err != nil {
			diagnostics.AddError("Failed to update tags", err.Error())
			return
		}
	}
}

// deleteDatabase implements the Delete operation for the active-active database resource.
func (r *activeActiveDatabaseResource) deleteDatabase(ctx context.Context, state *ActiveActiveDatabaseModel, diagnostics *diag.Diagnostics) {
	subId, dbId, err := parseResourceId(state.ID.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid resource ID", err.Error())
		return
	}

	// Acquire subscription mutex
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// Wait for database to be active before deletion
	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, r.client); err != nil {
		diagnostics.AddError("Database not active", err.Error())
		return
	}

	// BUG FIX: Actually return the error instead of swallowing it
	if err := r.client.Client.Database.Delete(ctx, subId, dbId); err != nil {
		diagnostics.AddError("Failed to delete database", err.Error())
		return
	}

	// Wait for deletion to complete
	if err := waitForDatabaseToBeDeleted(ctx, subId, dbId, r.client); err != nil {
		diagnostics.AddError("Database deletion failed", err.Error())
		return
	}
}

// buildRegionsFromPlan builds the regions list for the update request from the plan.
func (r *activeActiveDatabaseResource) buildRegionsFromPlan(ctx context.Context, plan *ActiveActiveDatabaseModel) ([]*databases.LocalRegionProperties, diag.Diagnostics) {
	var allDiags diag.Diagnostics

	if plan.OverrideRegion.IsNull() || plan.OverrideRegion.IsUnknown() {
		return nil, nil
	}

	var overrideRegions []OverrideRegionModel
	diags := plan.OverrideRegion.ElementsAs(ctx, &overrideRegions, false)
	allDiags.Append(diags...)
	if allDiags.HasError() {
		return nil, allDiags
	}

	// Get global values for fallback
	globalSourceIPs, diags := setToStringSlice(ctx, plan.GlobalSourceIPs)
	allDiags.Append(diags...)

	globalAlerts, diags := buildAlertsFromSet(ctx, plan.GlobalAlert)
	allDiags.Append(diags...)

	if allDiags.HasError() {
		return nil, allDiags
	}

	regions := make([]*databases.LocalRegionProperties, 0, len(overrideRegions))
	for _, region := range overrideRegions {
		regionProps := &databases.LocalRegionProperties{
			Region: redis.String(region.Name.ValueString()),
		}

		// Set enable_default_user
		regionProps.EnableDefaultUser = redis.Bool(region.EnableDefaultUser.ValueBool())

		// Build override alerts or use global alerts
		overrideAlerts, diags := buildAlertsFromSet(ctx, region.OverrideGlobalAlert)
		allDiags.Append(diags...)
		if len(overrideAlerts) > 0 {
			regionProps.Alerts = &overrideAlerts
		} else if len(globalAlerts) > 0 {
			regionProps.Alerts = &globalAlerts
		}

		// Build override source IPs or use global source IPs
		overrideSourceIPs, diags := setToStringSlice(ctx, region.OverrideGlobalSourceIPs)
		allDiags.Append(diags...)
		if len(overrideSourceIPs) > 0 {
			regionProps.SourceIP = stringPtrSlice(overrideSourceIPs)
		} else if len(globalSourceIPs) > 0 {
			regionProps.SourceIP = stringPtrSlice(globalSourceIPs)
		}

		// Set data persistence
		if !region.OverrideGlobalDataPersistence.IsNull() && region.OverrideGlobalDataPersistence.ValueString() != "" {
			regionProps.DataPersistence = redis.String(region.OverrideGlobalDataPersistence.ValueString())
		} else if !plan.GlobalDataPersistence.IsNull() && plan.GlobalDataPersistence.ValueString() != "" {
			regionProps.DataPersistence = redis.String(plan.GlobalDataPersistence.ValueString())
		}

		// Set password
		if !region.OverrideGlobalPassword.IsNull() && region.OverrideGlobalPassword.ValueString() != "" {
			regionProps.Password = redis.String(region.OverrideGlobalPassword.ValueString())
		} else if !plan.GlobalPassword.IsNull() && plan.GlobalPassword.ValueString() != "" {
			regionProps.Password = redis.String(plan.GlobalPassword.ValueString())
		}

		// Build backup plan
		backupConfig, diags := buildBackupPlan(ctx, region.RemoteBackup)
		allDiags.Append(diags...)
		if backupConfig != nil {
			regionProps.RemoteBackup = backupConfig
		}

		regions = append(regions, regionProps)
	}

	return regions, allDiags
}

// buildOverrideRegionFromAPI builds the override_region set from the API response.
func (r *activeActiveDatabaseResource) buildOverrideRegionFromAPI(ctx context.Context, db *databases.ActiveActiveDatabase, state *ActiveActiveDatabaseModel) (types.Set, diag.Diagnostics) {
	var allDiags diag.Diagnostics

	alertAttrTypes := getAlertAttrTypes()
	remoteBackupAttrTypes := getRemoteBackupAttrTypes()
	overrideRegionAttrTypes := map[string]attr.Type{
		"name":                             types.StringType,
		"override_global_alert":            types.SetType{ElemType: types.ObjectType{AttrTypes: alertAttrTypes}},
		"override_global_password":         types.StringType,
		"override_global_source_ips":       types.SetType{ElemType: types.StringType},
		"override_global_data_persistence": types.StringType,
		"enable_default_user":              types.BoolType,
		"remote_backup":                    types.ListType{ElemType: types.ObjectType{AttrTypes: remoteBackupAttrTypes}},
	}

	// If no override_region in state, return null set
	if state.OverrideRegion.IsNull() || len(state.OverrideRegion.Elements()) == 0 {
		return types.SetNull(types.ObjectType{AttrTypes: overrideRegionAttrTypes}), nil
	}

	// Get the regions from state to know which ones to include
	var stateRegions []OverrideRegionModel
	diags := state.OverrideRegion.ElementsAs(ctx, &stateRegions, false)
	allDiags.Append(diags...)
	if allDiags.HasError() {
		return types.SetNull(types.ObjectType{AttrTypes: overrideRegionAttrTypes}), allDiags
	}

	stateRegionNames := make(map[string]*OverrideRegionModel)
	for i, r := range stateRegions {
		stateRegionNames[r.Name.ValueString()] = &stateRegions[i]
	}

	// Build override region configs from API response
	regionConfigs := make([]attr.Value, 0)
	for _, regionDb := range db.CrdbDatabases {
		regionName := redis.StringValue(regionDb.Region)

		// Only include regions that are in the state
		stateRegion, exists := stateRegionNames[regionName]
		if !exists {
			continue
		}

		// Build the region config
		regionConfig := map[string]attr.Value{
			"name": types.StringValue(regionName),
		}

		// Handle override_global_source_ips
		// Only set if the state had source IPs configured
		if stateRegion != nil && !stateRegion.OverrideGlobalSourceIPs.IsNull() && len(stateRegion.OverrideGlobalSourceIPs.Elements()) > 0 {
			sourceIPs := stringSliceValue(regionDb.Security.SourceIPs)
			// Filter out default source IPs
			if !isDefaultSourceIPsForRegion(sourceIPs) {
				sourceIPSet, diags := stringSliceToSet(ctx, sourceIPs)
				allDiags.Append(diags...)
				regionConfig["override_global_source_ips"] = sourceIPSet
			} else {
				regionConfig["override_global_source_ips"] = types.SetNull(types.StringType)
			}
		} else {
			regionConfig["override_global_source_ips"] = types.SetNull(types.StringType)
		}

		// Handle override_global_data_persistence
		if stateRegion != nil && !stateRegion.OverrideGlobalDataPersistence.IsNull() && stateRegion.OverrideGlobalDataPersistence.ValueString() != "" {
			if regionDb.DataPersistence != nil {
				regionConfig["override_global_data_persistence"] = types.StringValue(redis.StringValue(regionDb.DataPersistence))
			} else {
				regionConfig["override_global_data_persistence"] = types.StringNull()
			}
		} else {
			regionConfig["override_global_data_persistence"] = types.StringNull()
		}

		// Handle override_global_password
		// BUG FIX: Add nil check for regionDb.Security.Password
		if stateRegion != nil && !stateRegion.OverrideGlobalPassword.IsNull() && stateRegion.OverrideGlobalPassword.ValueString() != "" {
			if regionDb.Security != nil && regionDb.Security.Password != nil {
				globalPassword := state.GlobalPassword.ValueString()
				if *regionDb.Security.Password == globalPassword {
					regionConfig["override_global_password"] = types.StringValue("")
				} else {
					regionConfig["override_global_password"] = types.StringValue(redis.StringValue(regionDb.Security.Password))
				}
			} else {
				regionConfig["override_global_password"] = types.StringNull()
			}
		} else {
			regionConfig["override_global_password"] = types.StringNull()
		}

		// Handle override_global_alert
		if stateRegion != nil && !stateRegion.OverrideGlobalAlert.IsNull() && len(stateRegion.OverrideGlobalAlert.Elements()) > 0 {
			alertSet, diags := flattenAlertsToSet(ctx, regionDb.Alerts)
			allDiags.Append(diags...)
			regionConfig["override_global_alert"] = alertSet
		} else {
			regionConfig["override_global_alert"] = types.SetNull(types.ObjectType{AttrTypes: alertAttrTypes})
		}

		// Handle enable_default_user
		if regionDb.Security != nil && regionDb.Security.EnableDefaultUser != nil {
			regionConfig["enable_default_user"] = types.BoolValue(redis.BoolValue(regionDb.Security.EnableDefaultUser))
		} else {
			regionConfig["enable_default_user"] = types.BoolValue(true) // Default value
		}

		// Handle remote_backup
		// Get storage_type from state since API doesn't return it
		stateStorageType := ""
		if stateRegion != nil && !stateRegion.RemoteBackup.IsNull() && len(stateRegion.RemoteBackup.Elements()) > 0 {
			var stateBackups []RemoteBackupModel
			diags := stateRegion.RemoteBackup.ElementsAs(ctx, &stateBackups, false)
			allDiags.Append(diags...)
			if len(stateBackups) > 0 {
				stateStorageType = stateBackups[0].StorageType.ValueString()
			}
		}
		backupList, diags := flattenBackupPlan(ctx, regionDb.Backup, stateStorageType)
		allDiags.Append(diags...)
		regionConfig["remote_backup"] = backupList

		regionObj, diags := types.ObjectValue(overrideRegionAttrTypes, regionConfig)
		allDiags.Append(diags...)
		if allDiags.HasError() {
			continue
		}
		regionConfigs = append(regionConfigs, regionObj)
	}

	if len(regionConfigs) == 0 {
		return types.SetNull(types.ObjectType{AttrTypes: overrideRegionAttrTypes}), allDiags
	}

	return types.SetValue(types.ObjectType{AttrTypes: overrideRegionAttrTypes}, regionConfigs)
}

// isDefaultSourceIPsForRegion checks if the source IPs are default values.
func isDefaultSourceIPsForRegion(sourceIPs []string) bool {
	// Check for default public access
	if len(sourceIPs) == 1 && sourceIPs[0] == "0.0.0.0/0" {
		return true
	}

	// Check for RFC1918 private ranges
	if len(sourceIPs) == len(defaultPrivateIPRanges) {
		privateIPSet := make(map[string]bool)
		for _, ip := range defaultPrivateIPRanges {
			privateIPSet[ip] = true
		}
		for _, ip := range sourceIPs {
			if !privateIPSet[ip] {
				return false
			}
		}
		return true
	}

	return false
}
