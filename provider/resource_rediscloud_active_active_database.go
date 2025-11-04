package provider

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudActiveActiveDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within an active-active subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActiveDatabaseCreate,
		ReadContext:   resourceRedisCloudActiveActiveDatabaseRead,
		UpdateContext: resourceRedisCloudActiveActiveDatabaseUpdate,
		DeleteContext: resourceRedisCloudActiveActiveDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				subId, dbId, err := pro.ToDatabaseId(d.Id())
				if err != nil {
					return nil, err
				}
				if err := d.Set("subscription_id", subId); err != nil {
					return nil, err
				}
				if err := d.Set("db_id", dbId); err != nil {
					return nil, err
				}
				d.SetId(utils.BuildResourceId(subId, dbId))
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
			var keys []string
			for _, key := range diff.GetChangedKeysPrefix("override_region") {
				if strings.HasSuffix(key, "time_utc") {
					keys = append(keys, strings.TrimSuffix(key, ".0.time_utc"))
				}
			}

			for _, key := range keys {
				if err := pro.RemoteBackupIntervalSetCorrectly(key)(ctx, diff, i); err != nil {
					return err
				}
			}

			return nil
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "Identifier of the subscription",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"db_id": {
				Description: "Identifier of the database created",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"name": {
				Description:      "A meaningful name to identify the database",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 40)),
			},
			"memory_limit_in_gb": {
				Description:  "(Deprecated) Maximum memory usage for this specific database",
				Type:         schema.TypeFloat,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"memory_limit_in_gb", "dataset_size_in_gb"},
			},
			"dataset_size_in_gb": {
				Description:  "Maximum amount of data in the dataset for this specific database in GB",
				Type:         schema.TypeFloat,
				Optional:     true,
				Computed:     true,
				ExactlyOneOf: []string{"memory_limit_in_gb", "dataset_size_in_gb"},
			},
			"redis_version": {
				Description: "Defines the Redis database version. If omitted, the Redis version will be set to the default version",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"support_oss_cluster_api": {
				Description: "Support Redis open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"external_endpoint_for_oss_cluster_api": {
				Description: "Should use the external endpoint for open-source (OSS) Cluster API",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
			"enable_tls": {
				Description: "Use TLS for authentication.",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"client_ssl_certificate": {
				Description: "SSL certificate to authenticate user connections.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"client_tls_certificates": {
				Description: "TLS certificates to authenticate user connections",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ConflictsWith: []string{"client_ssl_certificate"},
			},
			"data_eviction": {
				Description: "Data eviction items policy",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "volatile-lru",
			},
			"global_data_persistence": {
				Description: "Rate of database data persistence (in persistent storage)",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"global_password": {
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Computed:    true,
			},
			"global_alert": {
				Description: "Set of alerts to enable on the database",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description:      "Alert name",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
						},
						"value": {
							Description: "Alert value",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"global_modules": {
				Description: "List of modules to enable on the database. This information is only used when creating a new database and any changes will be ignored after this.",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
						// We don't want to ignore the block if the resource is about to be created.
						return false
					}
					return true
				},
			},
			"global_source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
				},
			},
			"global_enable_default_user": {
				Description: "When 'true', enables connecting to the database with the 'default' user across all regions. Default: 'true'",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"override_region": {
				Description: "Region-specific configuration parameters to override the global configuration",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Region name",
							Type:        schema.TypeString,
							Required:    true,
						},
						"override_global_alert": {
							Description: "Set of alerts to enable on the database",
							Type:        schema.TypeSet,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Description:      "Alert name",
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
									},
									"value": {
										Description: "Alert value",
										Type:        schema.TypeInt,
										Required:    true,
									},
								},
							},
						},
						"override_global_password": {
							Description: "Password used to access the database. If left empty, the password will be generated automatically",
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
						},
						"override_global_source_ips": {
							Description: "Set of CIDR addresses to allow access to the database",
							Type:        schema.TypeSet,
							Optional:    true,
							MinItems:    1,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validation.ToDiagFunc(validation.IsCIDR),
							},
						},
						"override_global_data_persistence": {
							Description: "Rate of database data persistence (in persistent storage)",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"enable_default_user": {
							Description: "When 'true', enables connecting to the database with the 'default' user. If not set, the global setting will be used.",
							Type:        schema.TypeBool,
							Optional:    true,
						},
						"remote_backup": {
							Description: "An object that specifies the backup options for the database in this region",
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"interval": {
										Description:      "Defines the frequency of the automatic backup",
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.BackupIntervals(), false)),
									},
									"time_utc": {
										Description:      "Defines the hour automatic backups are made - only applicable when interval is `every-12-hours` or `every-24-hours`",
										Type:             schema.TypeString,
										Optional:         true,
										ValidateDiagFunc: utils.IsTime(),
									},
									"storage_type": {
										Description:      "Defines the provider of the storage location",
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(databases.BackupStorageTypes(), false)),
									},
									"storage_path": {
										Description: "Defines a URI representing the backup storage location",
										Type:        schema.TypeString,
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
			"public_endpoint": {
				Description: "Region public and private endpoints to access the database",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"private_endpoint": {
				Description: "Region public and private endpoints to access the database",
				Type:        schema.TypeMap,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"port": {
				Description:      "TCP port on which the database is available",
				Type:             schema.TypeInt,
				ValidateDiagFunc: validation.ToDiagFunc(validation.IntBetween(10000, 19999)),
				Optional:         true,
				ForceNew:         true,
			},
			"global_resp_version": {
				Description: "The initial RESP version for all databases provisioned under this AA database. This information is only used when creating a new database and any changes will be ignored after this.",
				Type:        schema.TypeString,
				// The block is ignored in the UPDATE operation or after IMPORTing the resource.
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if d.Id() == "" {
						// We don't want to ignore the block if the resource is about to be created.
						return false
					}
					return true
				},
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.StringMatch(regexp.MustCompile("^(resp2|resp3)$"), "must be 'resp2' or 'resp3'")),
			},
			"tags": {
				Description: "Tags for database management",
				Type:        schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:         true,
				ValidateDiagFunc: pro.ValidateTagsfunc,
			},
		},
	}
}

func resourceRedisCloudActiveActiveDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subId := d.Get("subscription_id").(int)
	utils.SubscriptionMutex.Lock(subId)

	name := d.Get("name").(string)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	useExternalEndpointForOSSClusterAPI := d.Get("external_endpoint_for_oss_cluster_api").(bool)
	globalSourceIp := utils.SetToStringSlice(d.Get("global_source_ips").(*schema.Set))

	createAlerts := make([]*databases.Alert, 0)
	alerts := d.Get("global_alert").(*schema.Set)
	for _, alert := range alerts.List() {
		alertMap := alert.(map[string]interface{})

		alertName := alertMap["name"].(string)
		alertValue := alertMap["value"].(int)

		createAlert := &databases.Alert{
			Name:  redis.String(alertName),
			Value: redis.Int(alertValue),
		}

		createAlerts = append(createAlerts, createAlert)
	}

	createModules := make([]*databases.Module, 0)
	planModules := utils.InterfaceToStringSlice(d.Get("global_modules").([]interface{}))
	for _, module := range planModules {
		createModule := &databases.Module{
			Name: module,
		}
		createModules = append(createModules, createModule)
	}

	// Get regions from /subscriptions/{subscriptionId}/regions, this will use the Regions API
	regions, err := api.Client.Regions.List(ctx, subId)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	localThroughputs := make([]*databases.LocalThroughput, 0)
	for _, region := range regions.Regions {
		createLocalThroughput := &databases.LocalThroughput{
			Region:                   redis.String(*region.Region),
			WriteOperationsPerSecond: redis.Int(1000),
			ReadOperationsPerSecond:  redis.Int(1000),
		}

		localThroughputs = append(localThroughputs, createLocalThroughput)
	}

	createDatabase := databases.CreateActiveActiveDatabase{
		DryRun:                              redis.Bool(false),
		Name:                                redis.String(name),
		SupportOSSClusterAPI:                redis.Bool(supportOSSClusterAPI),
		UseExternalEndpointForOSSClusterAPI: redis.Bool(useExternalEndpointForOSSClusterAPI),
		GlobalSourceIP:                      globalSourceIp,
		GlobalAlerts:                        createAlerts,
		GlobalModules:                       createModules,
		LocalThroughputMeasurement:          localThroughputs,
	}

	utils.SetStringIfNotEmpty(d, "data_eviction", func(s *string) {
		createDatabase.DataEvictionPolicy = s
	})

	utils.SetStringIfNotEmpty(d, "global_data_persistence", func(s *string) {
		createDatabase.GlobalDataPersistence = s
	})

	utils.SetStringIfNotEmpty(d, "global_password", func(s *string) {
		createDatabase.GlobalPassword = s
	})

	utils.SetFloat64(d, "dataset_size_in_gb", func(f *float64) {
		createDatabase.DatasetSizeInGB = f
	})

	utils.SetFloat64(d, "memory_limit_in_gb", func(f *float64) {
		createDatabase.MemoryLimitInGB = f
	})

	utils.SetIntIfPositive(d, "port", func(i *int) {
		createDatabase.PortNumber = i
	})

	utils.SetStringIfNotEmpty(d, "global_resp_version", func(s *string) {
		createDatabase.RespVersion = s
	})

	utils.SetStringIfNotEmpty(d, "redis_version", func(s *string) {
		createDatabase.RedisVersion = s
	})

	utils.SetBool(d, "auto_minor_version_upgrade", func(b *bool) {
		createDatabase.AutoMinorVersionUpgrade = b
	})

	// Confirm Subscription Active status before creating database
	err = utils.WaitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	dbId, err := api.Client.Database.ActiveActiveCreate(ctx, subId, createDatabase)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	d.SetId(utils.BuildResourceId(subId, dbId))

	// Confirm Database Active status
	err = utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api)
	if err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		utils.SubscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	utils.SubscriptionMutex.Unlock(subId)
	return resourceRedisCloudActiveActiveDatabaseUpdate(ctx, d, meta)
}

func resourceRedisCloudActiveActiveDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	var diags diag.Diagnostics

	subId, dbId, err := pro.ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// We are not import this resource, so we can read the subscription_id defined in this resource.
	if subId == 0 {
		subId = d.Get("subscription_id").(int)
	}

	db, err := api.Client.Database.GetActiveActive(ctx, subId, dbId)
	if err != nil {
		if _, ok := err.(*databases.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("db_id", redis.IntValue(db.ID)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(db.Name)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("data_eviction", redis.StringValue(db.DataEvictionPolicy)); err != nil {
		return diag.FromErr(err)
	}

	if _, ok := d.GetOk("global_data_persistence"); ok {
		if db.GlobalDataPersistence != nil {
			if err := d.Set("global_data_persistence", redis.StringValue(db.GlobalDataPersistence)); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	// Read global_password from API response
	if db.GlobalPassword != nil {
		if err := d.Set("global_password", redis.StringValue(db.GlobalPassword)); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("external_endpoint_for_oss_cluster_api", redis.BoolValue(db.UseExternalEndpointForOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("enable_tls", redis.BoolValue(db.CrdbDatabases[0].Security.EnableTls)); err != nil {
		return diag.FromErr(err)
	}

	// To prevent both fields being included in API requests, only one of these two fields should be set in the state
	// Only add `dataset_size_in_gb` to the state if `memory_limit_in_gb` is not already in the state
	if _, inState := d.GetOk("memory_limit_in_gb"); !inState {
		if err := d.Set("dataset_size_in_gb", redis.Float64(*db.CrdbDatabases[0].DatasetSizeInGB)); err != nil {
			return diag.FromErr(err)
		}
	}

	// Likewise, only add `memory_limit_in_gb` to the state if `dataset_size_in_gb` is not already in the state
	if _, inState := d.GetOk("dataset_size_in_gb"); !inState {
		if err := d.Set("memory_limit_in_gb", redis.Float64(*db.CrdbDatabases[0].MemoryLimitInGB)); err != nil {
			return diag.FromErr(err)
		}
	}

	var regionDbConfigs []map[string]interface{}
	publicEndpointConfig := make(map[string]interface{})
	privateEndpointConfig := make(map[string]interface{})

	// Build API region lookup map
	apiRegions := make(map[string]*databases.CrdbDatabase)
	for _, regionDb := range db.CrdbDatabases {
		region := redis.StringValue(regionDb.Region)
		apiRegions[region] = regionDb
		// Set the endpoints for all regions
		publicEndpointConfig[region] = redis.StringValue(regionDb.PublicEndpoint)
		privateEndpointConfig[region] = redis.StringValue(regionDb.PrivateEndpoint)
	}

	// Iterate through STATE override_region blocks (not API regions) to preserve Set ordering/hashing
	stateOverrideRegions := d.Get("override_region").(*schema.Set).List()
	tflog.Debug(ctx, "Read: Starting to process regions from STATE", map[string]interface{}{
		"regionCount": len(stateOverrideRegions),
	})

	for _, stateRegion := range stateOverrideRegions {
		stateRegionMap := stateRegion.(map[string]interface{})
		region := stateRegionMap["name"].(string)

		// Debug: log what keys are in stateRegionMap
		mapKeys := make([]string, 0, len(stateRegionMap))
		for k := range stateRegionMap {
			mapKeys = append(mapKeys, k)
		}
		tflog.Debug(ctx, "Read: StateRegionMap keys", map[string]interface{}{
			"region": region,
			"keys":   mapKeys,
		})

		// Look up corresponding API data
		regionDb, exists := apiRegions[region]
		if !exists {
			tflog.Warn(ctx, "Read: Region in state not found in API response", map[string]interface{}{
				"region": region,
			})
			continue
		}

		regionDbConfig := buildRegionConfigFromAPIAndState(ctx, d, db, region, regionDb, stateRegionMap)
		regionDbConfigs = append(regionDbConfigs, regionDbConfig)
	}

	tflog.Debug(ctx, "Read: Completed processing all regions", map[string]interface{}{
		"totalRegionsProcessed": len(regionDbConfigs),
	})

	// Only set override_region if it is defined in the config
	if len(d.Get("override_region").(*schema.Set).List()) > 0 {
		if err := d.Set("override_region", regionDbConfigs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("public_endpoint", publicEndpointConfig); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_endpoint", privateEndpointConfig); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("global_modules", flattenModulesToNames(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("redis_version", redis.StringValue(db.RedisVersion)); err != nil {
		return diag.FromErr(err)
	}

	if _, ok := d.GetOk("auto_minor_version_upgrade"); ok {
		if err := d.Set("auto_minor_version_upgrade", redis.BoolValue(db.AutoMinorVersionUpgrade)); err != nil {
			return diag.FromErr(err)
		}
	}

	// Read global_enable_default_user from API response
	if db.GlobalEnableDefaultUser != nil {
		globalValue := redis.BoolValue(db.GlobalEnableDefaultUser)
		tflog.Debug(ctx, "Read: Setting global_enable_default_user from API", map[string]interface{}{
			"value": globalValue,
		})
		if err := d.Set("global_enable_default_user", globalValue); err != nil {
			return diag.FromErr(err)
		}
	} else {
		tflog.Debug(ctx, "Read: global_enable_default_user is nil in API response", map[string]interface{}{})
	}

	tlsAuthEnabled := *db.CrdbDatabases[0].Security.TLSClientAuthentication
	if err := utils.ApplyCertificateHints(tlsAuthEnabled, d); err != nil {
		return diag.FromErr(err)
	}

	if err := pro.ReadTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudActiveActiveDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*client.ApiClient)

	var diags diag.Diagnostics
	subId := d.Get("subscription_id").(int)

	_, dbId, err := pro.ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	dbErr := api.Client.Database.Delete(ctx, subId, dbId)
	if dbErr != nil {
		diag.FromErr(dbErr)
	}

	err = waitForDatabaseToBeDeleted(ctx, subId, dbId, api)
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceRedisCloudActiveActiveDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	_, dbId, err := pro.ToDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subId := d.Get("subscription_id").(int)
	utils.SubscriptionMutex.Lock(subId)
	defer utils.SubscriptionMutex.Unlock(subId)

	// Forcibly initialise, so we have a non-nil, zero-length slice
	// A pointer to a nil-slice is interpreted as empty and omitted from the json payload
	//goland:noinspection GoPreferNilSlice
	updateAlerts := []*databases.Alert{}
	for _, alert := range d.Get("global_alert").(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		updateAlerts = append(updateAlerts, &databases.Alert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	globalSourceIps := utils.SetToStringSlice(d.Get("global_source_ips").(*schema.Set))

	// Make a list of region-specific configurations
	var regions []*databases.LocalRegionProperties

	tflog.Debug(ctx, "Update: Starting to build region configurations", map[string]interface{}{
		"regionCount": len(d.Get("override_region").(*schema.Set).List()),
	})

	for _, region := range d.Get("override_region").(*schema.Set).List() {
		dbRegion := region.(map[string]interface{})

		overrideAlerts := getStateAlertsFromDbRegion(getStateOverrideRegion(d, dbRegion["name"].(string)))

		// Make a list of region-specific source IPs for use in the regions list below
		var overrideSourceIps []*string
		for _, sourceIp := range dbRegion["override_global_source_ips"].(*schema.Set).List() {
			overrideSourceIps = append(overrideSourceIps, redis.String(sourceIp.(string)))
		}

		regionProps := &databases.LocalRegionProperties{
			Region: redis.String(dbRegion["name"].(string)),
		}

		// Handle enable_default_user with three-state logic:
		// - Not set in config -> don't send to API (inherit from global)
		// - Explicitly true -> send true
		// - Explicitly false -> send false
		regionName := dbRegion["name"].(string)
		enableDefaultUser := dbRegion["enable_default_user"].(bool)
		wasExplicitlySet := isEnableDefaultUserExplicitlySetInConfig(d, regionName)

		tflog.Debug(ctx, "Checking enable_default_user for region", map[string]interface{}{
			"region":            regionName,
			"value":             enableDefaultUser,
			"wasExplicitlySet":  wasExplicitlySet,
		})

		if wasExplicitlySet {
			// Field was explicitly set in Terraform config, send the value
			tflog.Debug(ctx, "Sending enable_default_user to API", map[string]interface{}{
				"region": regionName,
				"value":  enableDefaultUser,
			})
			regionProps.EnableDefaultUser = redis.Bool(enableDefaultUser)
		} else {
			tflog.Debug(ctx, "Not sending enable_default_user (inherit from global)", map[string]interface{}{
				"region": regionName,
			})
		}
		// If not explicitly set, don't send - let it inherit from global

		if len(overrideAlerts) > 0 {
			regionProps.Alerts = &overrideAlerts
		} else if len(updateAlerts) > 0 {
			regionProps.Alerts = &updateAlerts
		}
		if len(overrideSourceIps) > 0 {
			regionProps.SourceIP = overrideSourceIps
		} else if len(globalSourceIps) > 0 {
			regionProps.SourceIP = globalSourceIps
		}
		dataPersistence := dbRegion["override_global_data_persistence"].(string)
		if dataPersistence != "" {
			regionProps.DataPersistence = redis.String(dataPersistence)
		} else if d.Get("global_data_persistence").(string) != "" {
			regionProps.DataPersistence = redis.String(d.Get("global_data_persistence").(string))
		}
		password := dbRegion["override_global_password"].(string)
		// If the password is not set, check if the global password is set and use that
		if password != "" {
			regionProps.Password = redis.String(password)
		} else {
			if d.Get("global_password").(string) != "" {
				regionProps.Password = redis.String(d.Get("global_password").(string))
			}
		}

		regionProps.RemoteBackup = pro.BuildBackupPlan(dbRegion["remote_backup"], nil)

		tflog.Debug(ctx, "Update: Completed building region properties", map[string]interface{}{
			"region":                      regionName,
			"hasEnableDefaultUser":        regionProps.EnableDefaultUser != nil,
			"enableDefaultUserValue":      regionProps.EnableDefaultUser,
		})

		regions = append(regions, regionProps)
	}

	tflog.Debug(ctx, "Update: Completed building all region configurations", map[string]interface{}{
		"totalRegions": len(regions),
	})

	// Populate the database update request with the required fields
	update := databases.UpdateActiveActiveDatabase{
		GlobalAlerts:   &updateAlerts,
		GlobalSourceIP: globalSourceIps,
		Regions:        regions,
	}

	// One of the following fields must be set in the request, validation is handled in the schema (ExactlyOneOf)
	if v, ok := d.GetOk("dataset_size_in_gb"); ok {
		update.DatasetSizeInGB = redis.Float64(v.(float64))
	}

	if v, ok := d.GetOk("memory_limit_in_gb"); ok {
		update.MemoryLimitInGB = redis.Float64(v.(float64))
	}

	// The below fields are optional and will only be sent in the request if they are present in the Terraform configuration
	if len(globalSourceIps) == 0 {
		update.GlobalSourceIP = []*string{redis.String("0.0.0.0/0")}
	}

	if d.Get("global_password").(string) != "" {
		update.GlobalPassword = redis.String(d.Get("global_password").(string))
	}

	if d.Get("global_data_persistence").(string) != "" {
		update.GlobalDataPersistence = redis.String(d.Get("global_data_persistence").(string))
	}

	if v, ok := d.GetOkExists("global_enable_default_user"); ok {
		update.GlobalEnableDefaultUser = redis.Bool(v.(bool))
	}

	if v, ok := d.GetOk("auto_minor_version_upgrade"); ok {
		update.AutoMinorVersionUpgrade = redis.Bool(v.(bool))
	}

	if v, ok := d.GetOk("support_oss_cluster_api"); ok {
		update.SupportOSSClusterAPI = redis.Bool(v.(bool))
	}

	if v, ok := d.GetOk("external_endpoint_for_oss_cluster_api"); ok {
		update.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	}

	if v, ok := d.GetOk("data_eviction"); ok {
		update.DataEvictionPolicy = redis.String(v.(string))
	}

	//The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	clientTLSCertificates := utils.InterfaceToStringSlice(d.Get("client_tls_certificates").([]interface{}))
	enableTLS := d.Get("enable_tls").(bool)
	if enableTLS {
		update.EnableTls = redis.Bool(enableTLS)
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)

			// If the user has enableTls=true and provided an SSL certificate, we want to scrub any TLS certificates
			update.ClientTLSCertificates = &[]*string{}
		}
		update.ClientTLSCertificates = &clientTLSCertificates
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else if len(clientTLSCertificates) > 0 {
			return diag.Errorf("TLS certificates may not be provided while enable_tls is false")
		} else {
			// Default: enable_tls=false, client_ssl_certificate=""
			update.EnableTls = redis.Bool(enableTLS)
		}
	}

	err = api.Client.Database.ActiveActiveUpdate(ctx, subId, dbId, update)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := utils.WaitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	if err := utils.WaitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	// The Tags API is synchronous so we shouldn't have to wait for anything
	if err := pro.WriteTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudActiveActiveDatabaseRead(ctx, d, meta)
}

func getStateOverrideRegion(d *schema.ResourceData, regionName string) map[string]interface{} {
	for _, region := range d.Get("override_region").(*schema.Set).List() {
		dbRegion := region.(map[string]interface{})
		if dbRegion["name"].(string) == regionName {
			return dbRegion
		}
	}
	return nil
}

func getStateRemoteBackup(d *schema.ResourceData, regionName string) []interface{} {
	for _, region := range d.Get("override_region").(*schema.Set).List() {
		dbRegion := region.(map[string]interface{})
		if dbRegion["name"].(string) == regionName {
			return dbRegion["remote_backup"].([]interface{})
		}
	}
	return nil
}

func getStateAlertsFromDbRegion(dbRegion map[string]interface{}) []*databases.Alert {
	// Make a list of region-specific alert configurations for use in the regions list below
	if dbRegion == nil {
		return nil
	} else if dbRegion["override_global_alert"] == nil {
		return nil
	}
	// Initialise to non-nil, zero-length slice.
	//goland:noinspection GoPreferNilSlice
	overrideAlerts := []*databases.Alert{}
	for _, alert := range dbRegion["override_global_alert"].(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})
		overrideAlerts = append(overrideAlerts, &databases.Alert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}
	return overrideAlerts
}

func waitForDatabaseToBeDeleted(ctx context.Context, subId int, dbId int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:        30 * time.Second,
		Pending:      []string{"pending"},
		Target:       []string{"deleted"},
		Timeout:      10 * time.Minute,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for database %d to be deleted", dbId)

			_, err = api.Client.Database.Get(ctx, subId, dbId)
			if err != nil {
				if _, ok := err.(*databases.NotFound); ok {
					return "deleted", "deleted", nil
				}
				return nil, "", err
			}

			return "pending", "pending", nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func flattenModulesToNames(modules []*databases.Module) []string {
	var moduleNames = make([]string, 0)
	for _, module := range modules {
		moduleNames = append(moduleNames, redis.StringValue(module.Name))
	}
	return moduleNames
}

// isEnableDefaultUserExplicitlySetInConfig checks if enable_default_user was explicitly
// set in the Terraform configuration for a specific region in the override_region block.
//
// This is used by the Update function to determine whether to send the field to the API.
// We only need this for Update operations where GetRawConfig() is available.
func isEnableDefaultUserExplicitlySetInConfig(d *schema.ResourceData, regionName string) bool {
	rawConfig := d.GetRawConfig()

	// During Update, raw config should always be available
	if rawConfig.IsNull() {
		return false
	}

	// Check if override_region exists in raw config
	if !rawConfig.Type().HasAttribute("override_region") {
		return false
	}

	overrideRegionAttr := rawConfig.GetAttr("override_region")
	if overrideRegionAttr.IsNull() {
		return false
	}

	// Iterate through the set to find the matching region
	if overrideRegionAttr.Type().IsSetType() {
		iter := overrideRegionAttr.ElementIterator()
		for iter.Next() {
			_, regionVal := iter.Element()

			// Check if this is the region we're looking for
			if regionVal.Type().HasAttribute("name") {
				nameAttr := regionVal.GetAttr("name")
				if !nameAttr.IsNull() && nameAttr.AsString() == regionName {
					// Found the matching region, check if enable_default_user exists
					if regionVal.Type().HasAttribute("enable_default_user") {
						fieldAttr := regionVal.GetAttr("enable_default_user")
						// If the attribute exists and is not null, it was explicitly set
						return !fieldAttr.IsNull()
					}
					// Field doesn't exist in config for this region
					return false
				}
			}
		}
	}

	// Region not found or field not set
	return false
}

// buildRegionConfigFromAPIAndState builds a region config map from API data and state data.
// This function handles all the complex logic for determining which fields to include in the
// region config based on what's in the API response and what was previously in the state.
func buildRegionConfigFromAPIAndState(ctx context.Context, d *schema.ResourceData, db *databases.ActiveActiveDatabase, region string, regionDb *databases.CrdbDatabase, stateOverrideRegion map[string]interface{}) map[string]interface{} {
	tflog.Debug(ctx, "Read: Processing region from state", map[string]interface{}{
		"region":                                 region,
		"stateHasEnableDefaultUser":              stateOverrideRegion["enable_default_user"] != nil,
		"stateHasOverrideGlobalSourceIps":        stateOverrideRegion["override_global_source_ips"] != nil,
		"stateHasOverrideGlobalDataPersistence":  stateOverrideRegion["override_global_data_persistence"] != nil,
		"stateHasOverrideGlobalPassword":         stateOverrideRegion["override_global_password"] != nil && stateOverrideRegion["override_global_password"] != "",
		"stateHasOverrideGlobalAlert":            stateOverrideRegion["override_global_alert"] != nil,
		"stateHasRemoteBackup":                   stateOverrideRegion["remote_backup"] != nil,
	})

	regionDbConfig := map[string]interface{}{
		"name": region,
	}

	// Handle source_ips based on subscription's public_endpoint_access settings
	// When public_endpoint_access is false and source_ips is empty, API returns private IP ranges
	// When public_endpoint_access is true and source_ips is empty, API returns ["0.0.0.0/0"]
	// When source_ips is explicitly set by user, API returns the user's input
	// This is to prevent drift in terraform state as API response will differ from what terraform sees
	var sourceIPs []string
	privateIPRanges := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"}

	// Check if the returned source_ips matches default private IP ranges (when public access is blocked)
	isPrivateIPRange := len(regionDb.Security.SourceIPs) == len(privateIPRanges)
	if isPrivateIPRange {
		for i, ip := range regionDb.Security.SourceIPs {
			if redis.StringValue(ip) != privateIPRanges[i] {
				isPrivateIPRange = false
				break
			}
		}
	}

	// Check if the returned source_ips is the default public access ["0.0.0.0/0"]
	isDefaultPublicAccess := len(regionDb.Security.SourceIPs) == 1 && redis.StringValue(regionDb.Security.SourceIPs[0]) == "0.0.0.0/0"

	// Only set source_ips if they were explicitly configured by the user (not defaults)
	if !isDefaultPublicAccess && !isPrivateIPRange {
		sourceIPs = redis.StringSliceValue(regionDb.Security.SourceIPs...)
	}

	// Only set override fields if they differ from global values FROM API
	// This prevents drift when regions inherit from global (not explicitly overridden in config)
	// IMPORTANT: Compare against API global values (db.Global*), not state values (d.Get())
	// because state may have empty/default values when user didn't specify them
	//
	// We use pure API-to-API comparison without config checking because:
	// 1. Update function correctly handles field removal (sends global value when field removed from config)
	// 2. This makes Apply and Refresh behave identically (no GetRawConfig dependency)
	// 3. Matches the pattern used by Pro database resources

	// override_global_source_ips: only set if differs from global
	// Note: db.GlobalSourceIPs doesn't exist in API, source IPs are per-region only
	globalSourceIPsPtrs := utils.SetToStringSlice(d.Get("global_source_ips").(*schema.Set))
	globalSourceIPs := redis.StringSliceValue(globalSourceIPsPtrs...)
	if !stringSlicesEqual(sourceIPs, globalSourceIPs) && len(sourceIPs) > 0 {
		regionDbConfig["override_global_source_ips"] = sourceIPs
	}

	// override_global_data_persistence: only set if differs from global API value
	if regionDb.DataPersistence != nil && db.GlobalDataPersistence != nil {
		if redis.StringValue(regionDb.DataPersistence) != redis.StringValue(db.GlobalDataPersistence) {
			regionDbConfig["override_global_data_persistence"] = regionDb.DataPersistence
		}
	}

	// override_global_password: only set if differs from global API value
	if regionDb.Security.Password != nil && db.GlobalPassword != nil {
		if *regionDb.Security.Password != redis.StringValue(db.GlobalPassword) {
			regionDbConfig["override_global_password"] = redis.StringValue(regionDb.Security.Password)
		}
	}

	// override_global_alert: only set if differs from global
	// Note: Active-Active API doesn't return global alerts separately, so we compare counts
	globalAlerts := d.Get("global_alert").(*schema.Set).List()
	regionAlerts := pro.FlattenAlerts(regionDb.Alerts)
	if len(globalAlerts) != len(regionAlerts) {
		regionDbConfig["override_global_alert"] = regionAlerts
	}

	// remote_backup: only set if it exists in API response
	if regionDb.Backup != nil {
		stateRemoteBackup := stateOverrideRegion["remote_backup"]
		if stateRemoteBackup != nil {
			stateRemoteBackupList := stateRemoteBackup.([]interface{})
			if len(stateRemoteBackupList) > 0 {
				regionDbConfig["remote_backup"] = pro.FlattenBackupPlan(regionDb.Backup, stateRemoteBackupList, "")
			}
		}
	}

	// enable_default_user: Hybrid approach using GetRawConfig when available, state preservation when not
	// GetRawConfig is available during Apply/Update but NULL during standalone Refresh
	if regionDb.Security.EnableDefaultUser != nil && db.GlobalEnableDefaultUser != nil {
		globalEnableDefaultUser := redis.BoolValue(db.GlobalEnableDefaultUser)
		regionEnableDefaultUser := redis.BoolValue(regionDb.Security.EnableDefaultUser)

		// Check if GetRawConfig is available (indicates we're in Apply/Update context)
		rawConfig := d.GetRawConfig()
		getRawConfigAvailable := !rawConfig.IsNull()

		tflog.Debug(ctx, "Read: enable_default_user - checking GetRawConfig availability", map[string]interface{}{
			"region":                 region,
			"getRawConfigAvailable":  getRawConfigAvailable,
			"globalValue":            globalEnableDefaultUser,
			"regionValue":            regionEnableDefaultUser,
			"isDifferent":            regionEnableDefaultUser != globalEnableDefaultUser,
		})

		if getRawConfigAvailable {
			// GetRawConfig available - use config-based detection
			wasExplicitlySet := isEnableDefaultUserExplicitlySetInConfig(d, region)

			tflog.Debug(ctx, "Read: Using config-based detection (GetRawConfig available)", map[string]interface{}{
				"region":           region,
				"wasExplicitlySet": wasExplicitlySet,
			})

			if wasExplicitlySet {
				tflog.Debug(ctx, "Read: Setting enable_default_user (explicitly in config)", map[string]interface{}{
					"region": region,
					"value":  regionEnableDefaultUser,
				})
				regionDbConfig["enable_default_user"] = regionEnableDefaultUser
			} else if regionEnableDefaultUser != globalEnableDefaultUser {
				tflog.Debug(ctx, "Read: Setting enable_default_user (not in config but differs from global)", map[string]interface{}{
					"region": region,
					"value":  regionEnableDefaultUser,
				})
				regionDbConfig["enable_default_user"] = regionEnableDefaultUser
			} else {
				tflog.Debug(ctx, "Read: NOT setting enable_default_user (not in config, matches global)", map[string]interface{}{
					"region": region,
				})
			}
		} else {
			// GetRawConfig unavailable (Refresh) - use state preservation
			fieldWasInOldState := stateOverrideRegion["enable_default_user"] != nil
			var oldStateValue interface{}
			if fieldWasInOldState {
				oldStateValue = stateOverrideRegion["enable_default_user"]
			}

			tflog.Debug(ctx, "Read: Using state-based preservation (GetRawConfig unavailable)", map[string]interface{}{
				"region":            region,
				"fieldWasInOldState": fieldWasInOldState,
				"oldStateValue":     oldStateValue,
			})

			if fieldWasInOldState {
				// Field was in previous state - preserve it if it still makes sense
				if regionEnableDefaultUser != globalEnableDefaultUser {
					// Still differs from global - definitely keep it
					tflog.Debug(ctx, "Read: Setting enable_default_user (was in old state, differs from global)", map[string]interface{}{
						"region": region,
						"value":  regionEnableDefaultUser,
					})
					regionDbConfig["enable_default_user"] = regionEnableDefaultUser
				} else {
					// Matches global but was in old state - assume user still wants it explicit
					tflog.Debug(ctx, "Read: Setting enable_default_user (was in old state, preserving even though matches global)", map[string]interface{}{
						"region": region,
						"value":  regionEnableDefaultUser,
					})
					regionDbConfig["enable_default_user"] = regionEnableDefaultUser
				}
			} else {
				// Field was NOT in previous state
				if regionEnableDefaultUser != globalEnableDefaultUser {
					// Not in old state but differs - might be new override
					tflog.Debug(ctx, "Read: Setting enable_default_user (not in old state, but differs from global)", map[string]interface{}{
						"region": region,
						"value":  regionEnableDefaultUser,
					})
					regionDbConfig["enable_default_user"] = regionEnableDefaultUser
				} else {
					// Not in old state and matches global - don't add (inherited)
					tflog.Debug(ctx, "Read: NOT setting enable_default_user (not in old state, matches global)", map[string]interface{}{
						"region": region,
					})
				}
			}
		}
	}

	tflog.Debug(ctx, "Read: Completed region config", map[string]interface{}{
		"region":                           region,
		"hasEnableDefaultUser":             regionDbConfig["enable_default_user"] != nil,
		"enableDefaultUserValue":           regionDbConfig["enable_default_user"],
		"hasOverrideGlobalSourceIps":       regionDbConfig["override_global_source_ips"] != nil,
		"hasOverrideGlobalDataPersistence": regionDbConfig["override_global_data_persistence"] != nil,
		"hasOverrideGlobalPassword":        regionDbConfig["override_global_password"] != nil,
		"hasOverrideGlobalAlert":           regionDbConfig["override_global_alert"] != nil,
		"hasRemoteBackup":                  regionDbConfig["remote_backup"] != nil,
	})

	return regionDbConfig
}

// stringSlicesEqual compares two string slices for equality (order matters)
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// alertsEqualMaps performs deep comparison of two alert lists (order-independent)
// Both parameters are []map[string]interface{} from pro.FlattenAlerts()
func alertsEqualMaps(alerts1, alerts2 []map[string]interface{}) bool {
	if len(alerts1) != len(alerts2) {
		return false
	}

	// For each alert in list1, find a matching alert in list2
	for _, alert1 := range alerts1 {
		name1 := alert1["name"].(string)
		value1 := alert1["value"].(int)

		// Look for matching alert in list2
		found := false
		for _, alert2 := range alerts2 {
			name2 := alert2["name"].(string)
			value2 := alert2["value"].(int)

			if name1 == name2 && value1 == value2 {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}
