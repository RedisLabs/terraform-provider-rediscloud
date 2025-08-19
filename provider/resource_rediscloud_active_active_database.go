package provider

import (
	"context"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
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
				subId, dbId, err := toDatabaseId(d.Id())
				if err != nil {
					return nil, err
				}
				if err := d.Set("subscription_id", subId); err != nil {
					return nil, err
				}
				if err := d.Set("db_id", dbId); err != nil {
					return nil, err
				}
				d.SetId(buildResourceId(subId, dbId))
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
				if err := remoteBackupIntervalSetCorrectly(key)(ctx, diff, i); err != nil {
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
										ValidateDiagFunc: isTime(),
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
				ValidateDiagFunc: validateTagsfunc,
			},
		},
	}
}

func resourceRedisCloudActiveActiveDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId := d.Get("subscription_id").(int)
	subscriptionMutex.Lock(subId)

	name := d.Get("name").(string)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	useExternalEndpointForOSSClusterAPI := d.Get("external_endpoint_for_oss_cluster_api").(bool)
	globalSourceIp := setToStringSlice(d.Get("global_source_ips").(*schema.Set))

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
	planModules := interfaceToStringSlice(d.Get("global_modules").([]interface{}))
	for _, module := range planModules {
		createModule := &databases.Module{
			Name: module,
		}
		createModules = append(createModules, createModule)
	}

	// Get regions from /subscriptions/{subscriptionId}/regions, this will use the Regions API
	regions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		subscriptionMutex.Unlock(subId)
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

	// Confirm Subscription Active status before creating database
	err = waitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		subscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	dbId, err := api.client.Database.ActiveActiveCreate(ctx, subId, createDatabase)
	if err != nil {
		subscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	d.SetId(buildResourceId(subId, dbId))

	// Confirm Database Active status
	err = waitForDatabaseToBeActive(ctx, subId, dbId, api)
	if err != nil {
		subscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		subscriptionMutex.Unlock(subId)
		return diag.FromErr(err)
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	subscriptionMutex.Unlock(subId)
	return resourceRedisCloudActiveActiveDatabaseUpdate(ctx, d, meta)
}

func resourceRedisCloudActiveActiveDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// We are not import this resource, so we can read the subscription_id defined in this resource.
	if subId == 0 {
		subId = d.Get("subscription_id").(int)
	}

	db, err := api.client.Database.GetActiveActive(ctx, subId, dbId)
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
	for _, regionDb := range db.CrdbDatabases {
		region := redis.StringValue(regionDb.Region)
		// Set the endpoints for the region
		publicEndpointConfig[region] = redis.StringValue(regionDb.PublicEndpoint)
		privateEndpointConfig[region] = redis.StringValue(regionDb.PrivateEndpoint)
		// Check if the region is in the state as an override
		stateOverrideRegion := getStateOverrideRegion(d, region)
		if stateOverrideRegion == nil {
			continue
		}
		regionDbConfig := map[string]interface{}{
			"name": region,
		}
		var sourceIPs []string
		if !(len(regionDb.Security.SourceIPs) == 1 && redis.StringValue(regionDb.Security.SourceIPs[0]) == "0.0.0.0/0") {
			// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
			sourceIPs = redis.StringSliceValue(regionDb.Security.SourceIPs...)
		}
		if stateSourceIPs := getStateOverrideRegion(d, region)["override_global_source_ips"]; stateSourceIPs != nil {
			if len(stateSourceIPs.(*schema.Set).List()) > 0 {
				regionDbConfig["override_global_source_ips"] = sourceIPs
			}
		}

		if stateDataPersistence := getStateOverrideRegion(d, region)["override_global_data_persistence"]; stateDataPersistence != nil {
			if stateDataPersistence.(string) != "" {
				regionDbConfig["override_global_data_persistence"] = regionDb.DataPersistence
			}
		}

		if stateOverridePassword := getStateOverrideRegion(d, region)["override_global_password"]; stateOverridePassword != "" {
			if *regionDb.Security.Password == d.Get("global_password").(string) {
				regionDbConfig["override_global_password"] = ""
			} else {
				regionDbConfig["override_global_password"] = redis.StringValue(regionDb.Security.Password)
			}
		}

		stateOverrideAlerts := getStateAlertsFromDbRegion(getStateOverrideRegion(d, region))
		if len(stateOverrideAlerts) > 0 {
			regionDbConfig["override_global_alert"] = flattenAlerts(regionDb.Alerts)
		}

		regionDbConfig["remote_backup"] = flattenBackupPlan(regionDb.Backup, getStateRemoteBackup(d, region), "")

		regionDbConfigs = append(regionDbConfigs, regionDbConfig)
	}

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

	tlsAuthEnabled := *db.CrdbDatabases[0].Security.TLSClientAuthentication
	if err := applyCertificateHints(tlsAuthEnabled, d); err != nil {
		return diag.FromErr(err)
	}

	if err := readTags(ctx, api, subId, dbId, d); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudActiveActiveDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var diags diag.Diagnostics
	subId := d.Get("subscription_id").(int)

	_, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	dbErr := api.client.Database.Delete(ctx, subId, dbId)
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
	api := meta.(*apiClient)

	_, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subId := d.Get("subscription_id").(int)
	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

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

	globalSourceIps := setToStringSlice(d.Get("global_source_ips").(*schema.Set))

	// Make a list of region-specific configurations
	var regions []*databases.LocalRegionProperties
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

		regionProps.RemoteBackup = buildBackupPlan(dbRegion["remote_backup"], nil)

		regions = append(regions, regionProps)
	}

	// Populate the database update request with the required fields
	update := databases.UpdateActiveActiveDatabase{
		SupportOSSClusterAPI:                redis.Bool(d.Get("support_oss_cluster_api").(bool)),
		UseExternalEndpointForOSSClusterAPI: redis.Bool(d.Get("external_endpoint_for_oss_cluster_api").(bool)),
		DataEvictionPolicy:                  redis.String(d.Get("data_eviction").(string)),
		GlobalAlerts:                        &updateAlerts,
		GlobalSourceIP:                      globalSourceIps,
		Regions:                             regions,
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

	//The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	clientTLSCertificates := interfaceToStringSlice(d.Get("client_tls_certificates").([]interface{}))
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

	update.UseExternalEndpointForOSSClusterAPI = redis.Bool(d.Get("external_endpoint_for_oss_cluster_api").(bool))

	err = api.client.Database.ActiveActiveUpdate(ctx, subId, dbId, update)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
		return diag.FromErr(err)
	}

	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	// The Tags API is synchronous so we shouldn't have to wait for anything
	if err := writeTags(ctx, api, subId, dbId, d); err != nil {
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

func waitForDatabaseToBeDeleted(ctx context.Context, subId int, dbId int, api *apiClient) error {
	wait := &retry.StateChangeConf{
		Delay:        30 * time.Second,
		Pending:      []string{"pending"},
		Target:       []string{"deleted"},
		Timeout:      10 * time.Minute,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for database %d to be deleted", dbId)

			_, err = api.client.Database.Get(ctx, subId, dbId)
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
