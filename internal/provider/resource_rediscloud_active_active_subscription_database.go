package provider

import (
	"context"
	"reflect"
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
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(0, 40)),
			},
			"memory_limit_in_gb": {
				Description: "Maximum memory usage for this specific database",
				Type:        schema.TypeFloat,
				Required:    true,
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
				Default:     false,
			},
			"client_ssl_certificate": {
				Description: "SSL certificate to authenticate user connections.",
				Type:        schema.TypeString,
				Optional:    true,
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
				Default:     "none",
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
			},
			"public_endpoint": {
				Description: "Region public and private endpoints to access the database",
				Type:        schema.TypeMap,
				Computed:    true,
			},
			"private_endpoint": {
				Description: "Region public and private endpoints to access the database",
				Type:        schema.TypeMap,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudActiveActiveSubscriptionDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId := d.Get("subscription_id").(int)
	subscriptionMutex.Lock(subId)

	name := d.Get("name").(string)
	memoryLimitInGB := d.Get("memory_limit_in_gb").(float64)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	useExternalEndpointForOSSClusterAPI := d.Get("external_endpoint_for_oss_cluster_api").(bool)
	globalDataPersistence := d.Get("global_data_persistence").(string)
	globalPassword := d.Get("global_password").(string)
	globalSourceIp := setToStringSlice(d.Get("global_source_ips").(*schema.Set))

	createAlerts := make([]*databases.CreateAlert, 0)
	alerts := d.Get("global_alert").(*schema.Set)
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

	// Confirm Subscription Active status before creating database
	err = waitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	dbId, err := api.client.Database.ActiveActiveCreate(ctx, subId, createDatabase)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildResourceId(subId, dbId))

	// Confirm Database Active status
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

	if err := d.Set("support_oss_cluster_api", redis.BoolValue(db.SupportOSSClusterAPI)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("external_endpoint_for_oss_cluster_api",
		d.Get("external_endpoint_for_oss_cluster_api").(bool)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("global_data_persistence", d.Get("global_data_persistence")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("global_alert", d.Get("global_alert")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("global_password", d.Get("global_password")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("global_source_ips", d.Get("global_source_ips")); err != nil {
		return diag.FromErr(err)
	}

	var region_db_configs []map[string]interface{}
	public_endpoint_config := make(map[string]interface{})
	private_endpoint_config := make(map[string]interface{})
	for _, region_db := range db.CrdbDatabases {
		var sourceIPs []string
		if !(len(region_db.Security.SourceIPs) == 1 && redis.StringValue(region_db.Security.SourceIPs[0]) == "0.0.0.0/0") {
			// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
			sourceIPs = redis.StringSliceValue(region_db.Security.SourceIPs...)
		}
		region_db_config := map[string]interface{}{
			"name":                             redis.StringValue(region_db.Region),
			"override_global_data_persistence": redis.StringValue(region_db.DataPersistence),
			"override_global_source_ips":       sourceIPs,
		}
		if *region_db.Security.Password == d.Get("global_password").(string) {
			region_db_config["override_global_password"] = ""
		} else {
			region_db_config["override_global_password"] = redis.StringValue(region_db.Security.Password)
		}
		var global_alerts []*databases.Alert
		for _, alert := range d.Get("global_alert").(*schema.Set).List() {
			dbAlert := alert.(map[string]interface{})
			global_alerts = append(global_alerts, &databases.Alert{
				Name:  redis.String(dbAlert["name"].(string)),
				Value: redis.Int(dbAlert["value"].(int)),
			})
		}
		if reflect.DeepEqual(global_alerts, region_db.Alerts) {
			region_db_config["override_global_alert"] = []interface{}{}
		} else {
			region_db_config["override_global_alert"] = flattenAlerts(region_db.Alerts)
		}

		public_endpoint_config[redis.StringValue(region_db.Region)] = redis.StringValue(region_db.PublicEndpoint)
		private_endpoint_config[redis.StringValue(region_db.Region)] = redis.StringValue(region_db.PrivateEndpoint)

		region_db_configs = append(region_db_configs, region_db_config)
	}

	if err := d.Set("override_region", region_db_configs); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("public_endpoint", public_endpoint_config); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("private_endpoint", private_endpoint_config); err != nil {
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
	for _, alert := range d.Get("global_alert").(*schema.Set).List() {
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
		for _, alert := range dbRegion["override_global_alert"].(*schema.Set).List() {
			dbAlert := alert.(map[string]interface{})

			override_alerts = append(override_alerts, &databases.UpdateAlert{
				Name:  redis.String(dbAlert["name"].(string)),
				Value: redis.Int(dbAlert["value"].(int)),
			})
		}

		// Make a list of region-specific source IPs for use in the regions list below
		var override_source_ips []*string
		for _, source_ip := range dbRegion["override_global_source_ips"].(*schema.Set).List() {
			override_source_ips = append(override_source_ips, redis.String(source_ip.(string)))
		}

		region_props := &databases.LocalRegionProperties{
			Region:          redis.String(dbRegion["name"].(string)),
			DataPersistence: redis.String(dbRegion["override_global_data_persistence"].(string)),
			SourceIP:        override_source_ips,
			Alerts:          override_alerts,
		}
		password := dbRegion["override_global_password"].(string)
		// If the password is not set, check if the global password is set and use that
		if password != "" {
			region_props.Password = redis.String(password)
		} else {
			if d.Get("global_password").(string) != "" {
				region_props.Password = redis.String(d.Get("global_password").(string))
			}
		}
		regions = append(regions, region_props)
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

	//The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := d.Get("client_ssl_certificate").(string)
	enableTLS := d.Get("enable_tls").(bool)
	if enableTLS {
		//TLS only: enable_tls=true, client_ssl_certificate="".
		update.EnableTls = redis.Bool(enableTLS)
		//mTLS: enableTls=true, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
		}
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			update.ClientSSLCertificate = redis.String(clientSSLCertificate)
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

	return resourceRedisCloudActiveActiveSubscriptionDatabaseRead(ctx, d, meta)
}
