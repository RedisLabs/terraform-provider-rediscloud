package provider

import (
	"context"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudActiveActiveSubscriptionDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within an active-active subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActiveSubscriptionDatabaseCreate,
		ReadContext:   resourceRedisCloudActiveActiveSubscriptionDatabaseRead,
		UpdateContext: resourceRedisCloudActiveActiveSubscriptionDatabaseUpdate,
		DeleteContext: resourceRedisCloudActiveActiveSubscriptionDatabaseDelete,

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
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 40)),
			},
			"memory_limit_in_gb": {
				Description: "Maximum memory usage for this specific database",
				Type:        schema.TypeFloat,
				Required:    true,
			},
			// TODO: are the below two attributes optional?
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
			"global_data_persistence": {
				Description: "Rate of database data persistence (in persistent storage)",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "none",
			},
			"global_password": {
				Description: "Password used to access the database. If left empty, the password will be generated automatically",
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Computed:    true,
			},
			"public_endpoint": {
				Description: "Public endpoint to access the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"private_endpoint": {
				Description: "Private endpoint to access the database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			// TODO: this would make more sense as "override_global_alert", the same as "override_region".
			// However, it is specified as a list of "override_global_alerts" in the SOW https://docs.google.com/document/d/1STtcqlxNdYoCCiEyLust9yD2Q_SphRLm078vrltRmBA/edit#heading=h.5ymgmymxz0e8
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
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
						},
						"value": {
							Description: "Alert value",
							Type:        schema.TypeInt,
							Required:    true,
						},
					},
				},
			},
			"global_source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
				},
			},
			// TODO: consider naming this override_region_config
			"override_region": {
				// TODO: description
				Description: "",
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
										ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.AlertNameValues(), false)),
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
							Computed:    true,
						},
						"override_global_source_ips": {
							Description: "Set of CIDR addresses to allow access to the database",
							Type:        schema.TypeSet,
							Optional:    true,
							MinItems:    1,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
							},
						},
						"override_global_data_persistence": {
							Description: "Rate of database data persistence (in persistent storage)",
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "none",
						},
					},
				},
				// TODO: are the below attributes required?
				// "hashing_policy": {
				// 	Description: "List of regular expression rules to shard the database by. See the documentation on clustering for more information on the hashing policy - https://docs.redislabs.com/latest/rc/concepts/clustering/",
				// 	Type:        schema.TypeList,
				// 	Optional:    true,
				// 	Computed:    true,
				// 	Elem: &schema.Schema{
				// 		Type: schema.TypeString,
				// 		// Can't check that these are valid regex rules as the service wants something like `(?<tag>.*)`
				// 		// which isn't a valid Go regex
				// 	},
				// },
				// "enable_tls": {
				// 	Description: "Use TLS for authentication",
				// 	Type:        schema.TypeBool,
				// 	Optional:    true,
				// 	Default:     false,
				// },
				// "client_ssl_certificate": {
				// 	Description: "SSL certificate to authenticate user connections",
				// 	Type:        schema.TypeString,
				// 	Optional:    true,
				// 	Default:     "",
				// },
				// "periodic_backup_path": {
				// 	Description: "Path that will be used to store database backup files",
				// 	Type:        schema.TypeString,
				// 	Optional:    true,
				// 	Default:     "",
				// },
				// "replica_of": {
				// 	Description: "Set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided",
				// 	Type:        schema.TypeSet,
				// 	Optional:    true,
				// 	Elem: &schema.Schema{
				// 		Type:             schema.TypeString,
				// 		ValidateDiagFunc: validateDiagFunc(validation.IsURLWithScheme([]string{"redis"})),
				// 	},
				// },
				// "data_eviction": {
				// 	Description: "(Optional) The data items eviction policy (either: 'allkeys-lru', 'allkeys-lfu', 'allkeys-random', 'volatile-lru', 'volatile-lfu', 'volatile-random', 'volatile-ttl' or 'noeviction'. Default: 'volatile-lru')",
				// 	Type:        schema.TypeString,
				// 	Optional:    true,
				// 	Default:     "volatile-lru",
				// },
			},
		},
	}
}

func resourceRedisCloudActiveActiveSubscriptionDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId := d.Get("subscription_id").(int)
	subscriptionMutex.Lock(subId)

	name := d.Get("name").(string)
	protocol := "redis" // TODO: test leaving this out, or setting it to "redis" at the API client level
	memoryLimitInGB := d.Get("memory_limit_in_gb").(float64)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	useExternalEndpointForOSSClusterAPI := d.Get("external_endpoint_for_oss_cluster_api").(bool)
	globalDataPersistence := d.Get("global_data_persistence").(string)
	globalPassword := d.Get("global_password").(string)
	globalSourceIp := setToStringSlice(d.Get("global_source_ips").(*schema.Set))

	createAlerts := make([]*databases.CreateAlert, 0)
	alerts := d.Get("override_global_alert").(*schema.Set)
	for _, alert := range alerts.List() {
		alertMap := alert.(map[string]interface{})

		alertName := alertMap["name"].(string)
		alertValue := alertMap["value"].(int)

		createAlert := &databases.CreateAlert{
			Name:  redis.String(alertName),
			Value: redis.Int(alertValue),
		}

		createAlerts = append(createAlerts, createAlert)
	}

	// Get regions from /subscriptions/{subscriptionId}/regions, this will use the Regions API
	regions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
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
		Protocol:                            redis.String(protocol),
		MemoryLimitInGB:                     redis.Float64(memoryLimitInGB),
		SupportOSSClusterAPI:                redis.Bool(supportOSSClusterAPI),
		UseExternalEndpointForOSSClusterAPI: redis.Bool(useExternalEndpointForOSSClusterAPI),
		GlobalDataPersistence:               redis.String(globalDataPersistence),
		GlobalSourceIP:                      globalSourceIp,
		GlobalAlerts:                        createAlerts,
		LocalThroughputMeasurement:          localThroughputs,
	}
	if globalPassword != "" {
		createDatabase.GlobalPassword = redis.String(globalPassword)
	}

	dbId, err := api.client.Database.ActiveActiveCreate(ctx, subId, createDatabase)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildResourceId(subId, dbId))

	// Confirm Subscription Active status
	err = waitForDatabaseToBeActive(ctx, subId, dbId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Locate Databases to confirm Active status

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	subscriptionMutex.Unlock(subId)
	return resourceRedisCloudActiveActiveSubscriptionDatabaseUpdate(ctx, d, meta)
}

func resourceRedisCloudActiveActiveSubscriptionDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	db, err := api.client.Database.Get(ctx, subId, dbId)
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

	if err := d.Set("protocol", redis.StringValue(db.Protocol)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("memory_limit_in_gb", redis.Float64Value(db.MemoryLimitInGB)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("data_persistence", redis.StringValue(db.DataPersistence)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("data_eviction", redis.StringValue(db.DataEvictionPolicy)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("replication", redis.BoolValue(db.Replication)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("throughput_measurement_by", redis.StringValue(db.ThroughputMeasurement.By)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("throughput_measurement_value", redis.IntValue(db.ThroughputMeasurement.Value)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("public_endpoint", redis.StringValue(db.PublicEndpoint)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("private_endpoint", redis.StringValue(db.PrivateEndpoint)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("modules", flattenModules(db.Modules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("alert", flattenAlerts(db.Alerts)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("average_item_size_in_bytes", d.Get("average_item_size_in_bytes").(int)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("external_endpoint_for_oss_cluster_api",
		d.Get("external_endpoint_for_oss_cluster_api").(bool)); err != nil {
		return diag.FromErr(err)
	}

	password := d.Get("password").(string)
	if redis.StringValue(db.Protocol) == "redis" {
		// Only db with the "redis" protocol returns the password.
		password = redis.StringValue(db.Security.Password)
	}

	if err := d.Set("password", password); err != nil {
		return diag.FromErr(err)
	}
	var sourceIPs []string
	if !(len(db.Security.SourceIPs) == 1 && redis.StringValue(db.Security.SourceIPs[0]) == "0.0.0.0/0") {
		// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
		sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
	}

	if err := d.Set("source_ips", sourceIPs); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("hashing_policy", flattenRegexRules(db.Clustering.RegexRules)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("enable_tls", redis.Bool(*db.Security.EnableTls)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudActiveActiveSubscriptionDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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
	return diags
}

func resourceRedisCloudActiveActiveSubscriptionDatabaseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	_, dbId, err := toDatabaseId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subId := d.Get("subscription_id").(int)
	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	var global_alerts []*databases.UpdateAlert
	for _, alert := range d.Get("override_global_alert").(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		global_alerts = append(global_alerts, &databases.UpdateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	// Make a list of region-specific configurations
	var regions []*databases.LocalRegionProperties
	for _, region := range d.Get("override_region").(*schema.Set).List() {
		dbRegion := region.(map[string]interface{})

		// Make a list of region-specific alert configurations for use in the regions list below
		var override_alerts []*databases.UpdateAlert
		// TODO: change this if we have to use a list of alerts
		for _, alert := range d.Get("override_global_alert").(*schema.Set).List() {
			dbAlert := alert.(map[string]interface{})

			override_alerts = append(override_alerts, &databases.UpdateAlert{
				Name:  redis.String(dbAlert["name"].(string)),
				Value: redis.Int(dbAlert["value"].(int)),
			})
		}

		regions = append(regions, &databases.LocalRegionProperties{
			Region: redis.String(dbRegion["name"].(string)),
			// TODO: do we need RemoteBackup?
			// LocalThroughputMeasurement: &databases.LocalThroughput{
			// 	Region:                   redis.String(dbRegion["name"].(string)),
			// 	WriteOperationsPerSecond: redis.Int(dbRegion["write_operations_per_second"].(int)),
			// 	ReadOperationsPerSecond:  redis.Int(dbRegion["read_operations_per_second"].(int)),
			// },
			DataPersistence: redis.String(dbRegion["override_global_data_persistence"].(string)),
			Password:        redis.String(dbRegion["override_global_password"].(string)),
			//TODO: SourceIP:        redis.StringSlice(dbRegion["override_global_source_ips"].([]string)...),
			Alerts: override_alerts,
		})
	}

	// Populate the database update request with the required fields
	update := databases.UpdateActiveActiveDatabase{
		MemoryLimitInGB:                     redis.Float64(d.Get("memory_limit_in_gb").(float64)),
		SupportOSSClusterAPI:                redis.Bool(d.Get("support_oss_cluster_api").(bool)),
		UseExternalEndpointForOSSClusterAPI: redis.Bool(d.Get("external_endpoint_for_oss_cluster_api").(bool)),
		//DataEvictionPolicy: redis.String(d.Get("data_eviction").(string)),
		GlobalDataPersistence: redis.String(d.Get("global_data_persistence").(string)),
		GlobalAlerts:          global_alerts,
		Regions:               regions,
	}

	// The below fields are optional and will only be sent in the request if they are present in the Terraform configuration
	if len(setToStringSlice(d.Get("global_source_ips").(*schema.Set))) == 0 {
		update.GlobalSourceIP = []*string{redis.String("0.0.0.0/0")}
	}

	if d.Get("global_password").(string) != "" {
		update.GlobalPassword = redis.String(d.Get("global_password").(string))
	}

	// TODO: determine if these fields are required
	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	// clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	// enableTLS := d.Get("enable_tls").(bool)
	// if enableTLS {
	// TLS only: enable_tls=true, client_ssl_certificate="".
	// update.EnableTls = redis.Bool(enableTLS)
	// mTLS: enableTls=true, non-empty client_ssl_certificate.
	// 	// if clientSSLCertificate != "" {
	// 		update.ClientSSLCertificate = redis.String(clientSSLCertificate)
	// 	}
	// } else {
	// 	// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
	// 	if clientSSLCertificate != "" {
	// 		update.ClientSSLCertificate = redis.String(clientSSLCertificate)
	// 	} else {
	// 		// Default: enable_tls=false, client_ssl_certificate=""
	// 		update.EnableTls = redis.Bool(enableTLS)
	// 	}
	// }

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

	return resourceRedisCloudActiveActiveSubscriptionDatabaseRead(ctx, d, meta)
}
