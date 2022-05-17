package provider

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudDatabase() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates database resource within a subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudDatabaseCreate,
		ReadContext:   resourceRedisCloudDatabaseRead,
		UpdateContext: resourceRedisCloudDatabaseUpdate,
		DeleteContext: resourceRedisCloudDatabaseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				subId, err := strconv.Atoi(d.Id())
				if err != nil {
					return nil, err
				}

				// Populate the names of databases that already exist so that `flattenDatabases` can iterate over them
				api := meta.(*apiClient)
				list := api.client.Database.List(ctx, subId)
				var dbs []map[string]interface{}
				for list.Next() {
					dbs = append(dbs, map[string]interface{}{
						"name": redis.StringValue(list.Value().Name),
					})
				}
				if list.Err() != nil {
					return nil, list.Err()
				}
				if err := d.Set("database", dbs); err != nil {
					return nil, err
				}

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
			"protocol": {
				Description:      "The protocol that will be used to access the database, (either ‘redis’ or 'memcached’) ",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.ProtocolValues(), false)),
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
			"data_persistence": {
				Description: "Rate of database data persistence (in persistent storage)",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "none",
			},
			"replication": {
				Description: "Databases replication",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			"throughput_measurement_by": {
				Description:      "Throughput measurement method, (either ‘number-of-shards’ or ‘operations-per-second’)",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
			},
			"throughput_measurement_value": {
				Description: "Throughput value (as applies to selected measurement method)",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"average_item_size_in_bytes": {
				Description: "Relevant only to ram-and-flash clusters. Estimated average size (measured in bytes) of the items stored in the database",
				Type:        schema.TypeInt,
				Optional:    true,
				// Setting default to 0 so that the hash func produces the same hash when this field is not
				// specified. SDK's catch-all issue around this: https://github.com/hashicorp/terraform-plugin-sdk/issues/261
				Default: 0,
			},
			"password": {
				Description: "Password used to access the database",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
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
			"client_ssl_certificate": {
				Description: "SSL certificate to authenticate user connections",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"periodic_backup_path": {
				Description: "Path that will be used to store database backup files",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},
			"replica_of": {
				Description: "Set of Redis database URIs, in the format `redis://user:password@host:port`, that this database will be a replica of. If the URI provided is Redis Labs Cloud instance, only host and port should be provided",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsURLWithScheme([]string{"redis"})),
				},
			},
			"alert": {
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
			"module": {
				Description: "A module object",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the module to enable",
							Type:        schema.TypeString,
							Required:    true,
						},
					},
				},
			},
			"source_ips": {
				Description: "Set of CIDR addresses to allow access to the database",
				Type:        schema.TypeSet,
				Optional:    true,
				MinItems:    1,
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
				},
			},
			"hashing_policy": {
				Description: "List of regular expression rules to shard the database by. See the documentation on clustering for more information on the hashing policy - https://docs.redislabs.com/latest/rc/concepts/clustering/",
				Type:        schema.TypeList,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					// Can't check that these are valid regex rules as the service wants something like `(?<tag>.*)`
					// which isn't a valid Go regex
				},
			},
			"enable_tls": {
				Description: "Use TLS for authentication",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
}

func resourceRedisCloudDatabaseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	subId := d.Get("subscription_id").(int)

	// NEW
	name := d.Get("name").(string)
	protocol := d.Get("protocol").(string)
	memoryLimitInGB := d.Get("memory_limit_in_gb").(float64)
	supportOSSClusterAPI := d.Get("support_oss_cluster_api").(bool)
	dataPersistence := d.Get("data_persistence").(string)
	replication := d.Get("replication").(bool)
	throughputMeasurementBy := d.Get("throughput_measurement_by").(string)
	throughputMeasurementValue := d.Get("throughput_measurement_value").(int)
	averageItemSizeInBytes := d.Get("average_item_size_in_bytes").(int)

	createModules := make([]*databases.CreateModule, 0)
	modules := d.Get("module").(*schema.Set)
	for _, module := range modules.List() {
		moduleMap := module.(map[string]interface{})

		modName := moduleMap["name"].(string)

		createModule := &databases.CreateModule{
			Name: redis.String(modName),
		}

		createModules = append(createModules, createModule)
	}

	createAlerts := make([]*databases.CreateAlert, 0)
	alerts := d.Get("alert").(*schema.Set)
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

	createDatabase := databases.CreateDatabase{
		Name:                 redis.String(name),
		Protocol:             redis.String(protocol),
		MemoryLimitInGB:      redis.Float64(memoryLimitInGB),
		SupportOSSClusterAPI: redis.Bool(supportOSSClusterAPI),
		DataPersistence:      redis.String(dataPersistence),
		Replication:          redis.Bool(replication),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(throughputMeasurementBy),
			Value: redis.Int(throughputMeasurementValue),
		},
		Modules:                createModules,
		AverageItemSizeInBytes: redis.Int(averageItemSizeInBytes),
		Alerts:                 createAlerts,
	}

	dbId, err := api.client.Database.Create(ctx, subId, createDatabase)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(dbId))

	// Confirm Subscription Active status
	err = waitForDatabaseToBeActive(ctx, subId, dbId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Locate Databases to confirm Active status

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	dbId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	subId := d.Get("subscription_id").(int)

	//subscription, err := api.client.Subscription.Get(ctx, subId)
	database, err := api.client.Database.Get(ctx, subId, dbId)
	if err != nil {
		return diag.FromErr(err)
	}

	// Sets
	if err := d.Set("name", redis.StringValue(subscription.Name)); err != nil {
		return diag.FromErr(err)
	}

	if subscription.PaymentMethodID != nil && redis.IntValue(subscription.PaymentMethodID) != 0 {
		paymentMethodID := strconv.Itoa(redis.IntValue(subscription.PaymentMethodID))
		if err := d.Set("payment_method_id", paymentMethodID); err != nil {
			return diag.FromErr(err)
		}
	}
	if err := d.Set("payment_method", redis.StringValue(subscription.PaymentMethod)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_storage", redis.StringValue(subscription.MemoryStorage)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("cloud_provider", flattenCloudDetails(subscription.CloudDetails, true)); err != nil {
		return diag.FromErr(err)
	}

	flatDbs, err := flattenDatabases(ctx, subId, d.Get("database").(*schema.Set).List(), api)
	if err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("database", flatDbs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	if d.HasChange("allowlist") {
		cidrs := setToStringSlice(d.Get("allowlist.0.cidrs").(*schema.Set))
		sgs := setToStringSlice(d.Get("allowlist.0.security_group_ids").(*schema.Set))

		err := api.client.Subscription.UpdateCIDRAllowlist(ctx, subId, subscriptions.UpdateCIDRAllowlist{
			CIDRIPs:          cidrs,
			SecurityGroupIDs: sgs,
		})
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChanges("name", "payment_method_id") {
		updateSubscriptionRequest := subscriptions.UpdateSubscription{}

		if d.HasChange("name") {
			name := d.Get("name").(string)
			updateSubscriptionRequest.Name = &name
		}

		if d.HasChange("payment_method_id") {
			paymentMethodID, err := readPaymentMethodID(d)
			if err != nil {
				return diag.FromErr(err)
			}

			updateSubscriptionRequest.PaymentMethodID = paymentMethodID
		}

		err = api.client.Subscription.Update(ctx, subId, updateSubscriptionRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("database") || d.IsNewResource() {
		oldDb, newDb := d.GetChange("database")
		addition, existing, deletion := diff(oldDb.(*schema.Set), newDb.(*schema.Set), func(v interface{}) string {
			m := v.(map[string]interface{})
			return m["name"].(string)
		})

		if d.IsNewResource() {
			// Terraform will report all of the databases that were just created in resourceRedisCloudSubscriptionCreate
			// as newly added, but they have been created by the create subscription call. All that needs to happen to
			// them is to be updated like 'normal' existing databases.
			existing = addition
		} else {
			// this is not a new resource, so these databases really do new to be created
			for _, db := range addition {
				// This loop with addition is triggered when another database is added to the subscription.
				request := buildCreateDatabase(db)
				id, err := api.client.Database.Create(ctx, subId, request)
				if err != nil {
					return diag.FromErr(err)
				}

				log.Printf("[DEBUG] Created database %d", id)

				if err := waitForDatabaseToBeActive(ctx, subId, id, api); err != nil {
					return diag.FromErr(err)
				}
			}

			// Certain values - like the hashing policy - can only be set on an update, so the newly created databases
			// need to be updated straight away
			existing = append(existing, addition...)
		}

		nameId, err := getDatabaseNameIdMap(ctx, subId, api)
		if err != nil {
			return diag.FromErr(err)
		}

		for _, db := range existing {
			update := buildUpdateDatabase(db)
			dbId := nameId[redis.StringValue(update.Name)]

			log.Printf("[DEBUG] Updating database %s (%d)", redis.StringValue(update.Name), dbId)

			err = api.client.Database.Update(ctx, subId, dbId, update)
			if err != nil {
				return diag.FromErr(err)
			}

			if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
				return diag.FromErr(err)
			}
		}

		for _, db := range deletion {
			name := db["name"].(string)
			id := nameId[name]

			log.Printf("[DEBUG] Deleting database %s (%d)", name, id)

			err = api.client.Database.Delete(ctx, subId, id)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
}

func resourceRedisCloudDatabaseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	dbMap := d.Get("database").(map[string]interface{})
	dbId := dbMap["db_id"].(int)
	dbErr := api.client.Database.Delete(ctx, subId, dbId)
	if dbErr != nil {
		diag.FromErr(dbErr)
	}

	d.SetId("")

	return diags
}

func buildCreateDatabases(databases interface{}) []*subscriptions.CreateDatabase {
	createDatabases := make([]*subscriptions.CreateDatabase, 0)

	for _, database := range databases.(*schema.Set).List() {
		databaseMap := database.(map[string]interface{})

		name := databaseMap["name"].(string)
		protocol := databaseMap["protocol"].(string)
		memoryLimitInGB := databaseMap["memory_limit_in_gb"].(float64)
		supportOSSClusterAPI := databaseMap["support_oss_cluster_api"].(bool)
		dataPersistence := databaseMap["data_persistence"].(string)
		replication := databaseMap["replication"].(bool)
		throughputMeasurementBy := databaseMap["throughput_measurement_by"].(string)
		throughputMeasurementValue := databaseMap["throughput_measurement_value"].(int)
		averageItemSizeInBytes := databaseMap["average_item_size_in_bytes"].(int)

		createModules := make([]*subscriptions.CreateModules, 0)
		modules := databaseMap["module"].(*schema.Set)
		for _, module := range modules.List() {
			moduleMap := module.(map[string]interface{})

			modName := moduleMap["name"].(string)

			createModule := &subscriptions.CreateModules{
				Name: redis.String(modName),
			}

			createModules = append(createModules, createModule)
		}

		createDatabase := &subscriptions.CreateDatabase{
			Name:                 redis.String(name),
			Protocol:             redis.String(protocol),
			MemoryLimitInGB:      redis.Float64(memoryLimitInGB),
			SupportOSSClusterAPI: redis.Bool(supportOSSClusterAPI),
			DataPersistence:      redis.String(dataPersistence),
			Replication:          redis.Bool(replication),
			ThroughputMeasurement: &subscriptions.CreateThroughput{
				By:    redis.String(throughputMeasurementBy),
				Value: redis.Int(throughputMeasurementValue),
			},
			Quantity: redis.Int(1),
			Modules:  createModules,
		}

		if averageItemSizeInBytes > 0 {
			createDatabase.AverageItemSizeInBytes = &averageItemSizeInBytes
		}

		createDatabases = append(createDatabases, createDatabase)
	}

	return createDatabases
}

func buildCreateDatabase(db map[string]interface{}) databases.CreateDatabase {
	var alerts []*databases.CreateAlert
	for _, alert := range db["alert"].(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.CreateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	createModules := make([]*databases.CreateModule, 0)
	module := db["module"].(*schema.Set)
	for _, module := range module.List() {
		moduleMap := module.(map[string]interface{})

		modName := moduleMap["name"].(string)

		createModule := &databases.CreateModule{
			Name: redis.String(modName),
		}

		createModules = append(createModules, createModule)
	}

	create := databases.CreateDatabase{
		DryRun:               redis.Bool(false),
		Name:                 redis.String(db["name"].(string)),
		Protocol:             redis.String(db["protocol"].(string)),
		MemoryLimitInGB:      redis.Float64(db["memory_limit_in_gb"].(float64)),
		SupportOSSClusterAPI: redis.Bool(db["support_oss_cluster_api"].(bool)),
		DataPersistence:      redis.String(db["data_persistence"].(string)),
		Replication:          redis.Bool(db["replication"].(bool)),
		ThroughputMeasurement: &databases.CreateThroughputMeasurement{
			By:    redis.String(db["throughput_measurement_by"].(string)),
			Value: redis.Int(db["throughput_measurement_value"].(int)),
		},
		Alerts:    alerts,
		ReplicaOf: setToStringSlice(db["replica_of"].(*schema.Set)),
		Password:  redis.String(db["password"].(string)),
		SourceIP:  setToStringSlice(db["source_ips"].(*schema.Set)),
		Modules:   createModules,
	}

	averageItemSize := db["average_item_size_in_bytes"].(int)
	if averageItemSize > 0 {
		create.AverageItemSizeInBytes = redis.Int(averageItemSize)
	}

	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := db["client_ssl_certificate"].(string)
	enableTLS := db["enable_tls"].(bool)
	if enableTLS {
		// TLS only: enable_tls=true, client_ssl_certificate="".
		create.EnableTls = redis.Bool(enableTLS)
		// mTLS: enableTls=true, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			create.ClientSSLCertificate = redis.String(clientSSLCertificate)
		}
	} else {
		// mTLS (backward compatibility): enable_tls=false, non-empty client_ssl_certificate.
		if clientSSLCertificate != "" {
			create.ClientSSLCertificate = redis.String(clientSSLCertificate)
		} else {
			// Default: enable_tls=false, client_ssl_certificate=""
			create.EnableTls = redis.Bool(enableTLS)
		}
	}

	backupPath := db["periodic_backup_path"].(string)
	if backupPath != "" {
		create.PeriodicBackupPath = redis.String(backupPath)
	}

	if v, ok := db["external_endpoint_for_oss_cluster_api"]; ok {
		create.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	}

	return create
}

func buildUpdateDatabase(db map[string]interface{}) databases.UpdateDatabase {
	var alerts []*databases.UpdateAlert
	for _, alert := range db["alert"].(*schema.Set).List() {
		dbAlert := alert.(map[string]interface{})

		alerts = append(alerts, &databases.UpdateAlert{
			Name:  redis.String(dbAlert["name"].(string)),
			Value: redis.Int(dbAlert["value"].(int)),
		})
	}

	update := databases.UpdateDatabase{
		Name:                 redis.String(db["name"].(string)),
		MemoryLimitInGB:      redis.Float64(db["memory_limit_in_gb"].(float64)),
		SupportOSSClusterAPI: redis.Bool(db["support_oss_cluster_api"].(bool)),
		Replication:          redis.Bool(db["replication"].(bool)),
		ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
			By:    redis.String(db["throughput_measurement_by"].(string)),
			Value: redis.Int(db["throughput_measurement_value"].(int)),
		},
		DataPersistence: redis.String(db["data_persistence"].(string)),
		Password:        redis.String(db["password"].(string)),
		SourceIP:        setToStringSlice(db["source_ips"].(*schema.Set)),
		Alerts:          alerts,
	}

	update.ReplicaOf = setToStringSlice(db["replica_of"].(*schema.Set))
	if update.ReplicaOf == nil {
		update.ReplicaOf = make([]*string, 0)
	}

	// The cert validation is done by the API (HTTP 400 is returned if it's invalid).
	clientSSLCertificate := db["client_ssl_certificate"].(string)
	enableTLS := db["enable_tls"].(bool)
	if enableTLS {
		// TLS only: enable_tls=true, client_ssl_certificate="".
		update.EnableTls = redis.Bool(enableTLS)
		// mTLS: enableTls=true, non-empty client_ssl_certificate.
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

	regex := db["hashing_policy"].([]interface{})
	if len(regex) != 0 {
		update.RegexRules = interfaceToStringSlice(regex)
	}

	backupPath := db["periodic_backup_path"].(string)
	if backupPath != "" {
		update.PeriodicBackupPath = redis.String(backupPath)
	}

	if v, ok := db["external_endpoint_for_oss_cluster_api"]; ok {
		update.UseExternalEndpointForOSSClusterAPI = redis.Bool(v.(bool))
	}

	return update
}

func waitForDatabaseToBeActive(ctx context.Context, subId, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay: 10 * time.Second,
		Pending: []string{
			databases.StatusDraft,
			databases.StatusPending,
			databases.StatusActiveChangePending,
			databases.StatusRCPActiveChangeDraft,
			databases.StatusActiveChangeDraft,
			databases.StatusRCPDraft,
			databases.StatusRCPChangePending,
		},
		Target:  []string{databases.StatusActive},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for database %d to be active", id)

			database, err := api.client.Database.Get(ctx, subId, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(database.Status), redis.StringValue(database.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}


func isNil(i interface{}) bool {
	if i == nil {
		return true
	}

	if l, ok := i.([]interface{}); ok {
		if len(l) == 0 {
			return true
		}
	}

	if m, ok := i.(map[string]interface{}); ok {
		if len(m) == 0 {
			return true
		}
	}

	return false
}

func flattenDatabases(ctx context.Context, subId int, databases []interface{}, api *apiClient) ([]interface{}, error) {
	nameId, err := getDatabaseNameIdMap(ctx, subId, api)
	if err != nil {
		return nil, err
	}

	var flattened []interface{}
	for _, v := range databases {
		database := v.(map[string]interface{})
		name := database["name"].(string)
		id, ok := nameId[name]
		if !ok {
			log.Printf("database %d not found: %s", id, err)
			continue
		}

		db, err := api.client.Database.Get(ctx, subId, id)
		if err != nil {
			return nil, err
		}

		cert := database["client_ssl_certificate"].(string)
		backupPath := database["periodic_backup_path"].(string)
		averageItemSize := database["average_item_size_in_bytes"].(int)
		existingPassword := database["password"].(string)
		existingSourceIPs := database["source_ips"].(*schema.Set)
		external := database["external_endpoint_for_oss_cluster_api"].(bool)

		flattened = append(flattened, flattenDatabase(cert, external, backupPath, averageItemSize, existingPassword, existingSourceIPs, db))
	}
	return flattened, nil
}

func flattenDatabase(certificate string, externalOSSAPIEndpoint bool, backupPath string, averageItemSizeInBytes int, existingPassword string, existingSourceIp *schema.Set, db *databases.Database) map[string]interface{} {
	password := existingPassword
	if redis.StringValue(db.Protocol) == "redis" {
		// TODO need to check if this is expected behaviour or not
		password = redis.StringValue(db.Security.Password)
	}

	var sourceIPs []string
	if len(db.Security.SourceIPs) == 1 && redis.StringValue(db.Security.SourceIPs[0]) == "0.0.0.0/0" {
		// The API handles an empty list as ["0.0.0.0/0"] but need to be careful to match the input to avoid Terraform detecting drift
		if existingSourceIp.Len() != 0 {
			sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
		}
	} else {
		sourceIPs = redis.StringSliceValue(db.Security.SourceIPs...)
	}

	tf := map[string]interface{}{
		"db_id":                                 redis.IntValue(db.ID),
		"name":                                  redis.StringValue(db.Name),
		"protocol":                              redis.StringValue(db.Protocol),
		"memory_limit_in_gb":                    redis.Float64Value(db.MemoryLimitInGB),
		"support_oss_cluster_api":               redis.BoolValue(db.SupportOSSClusterAPI),
		"data_persistence":                      redis.StringValue(db.DataPersistence),
		"replication":                           redis.BoolValue(db.Replication),
		"throughput_measurement_by":             redis.StringValue(db.ThroughputMeasurement.By),
		"throughput_measurement_value":          redis.IntValue(db.ThroughputMeasurement.Value),
		"public_endpoint":                       redis.StringValue(db.PublicEndpoint),
		"private_endpoint":                      redis.StringValue(db.PrivateEndpoint),
		"module":                                flattenModules(db.Modules),
		"alert":                                 flattenAlerts(db.Alerts),
		"external_endpoint_for_oss_cluster_api": externalOSSAPIEndpoint,
		"password":                              password,
		"source_ips":                            sourceIPs,
		"hashing_policy":                        flattenRegexRules(db.Clustering.RegexRules),
		"enable_tls":                            redis.Bool(*db.Security.EnableTls),
	}

	if db.ReplicaOf != nil {
		tf["replica_of"] = redis.StringSliceValue(db.ReplicaOf.Endpoints...)
	}

	if redis.BoolValue(db.Security.SSLClientAuthentication) {
		tf["client_ssl_certificate"] = certificate
	}

	if averageItemSizeInBytes > 0 {
		tf["average_item_size_in_bytes"] = averageItemSizeInBytes
	}

	if backupPath != "" {
		tf["periodic_backup_path"] = backupPath
	}

	return tf
}

func flattenAlerts(alerts []*databases.Alert) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, alert := range alerts {
		tf := map[string]interface{}{
			"name":  redis.StringValue(alert.Name),
			"value": redis.IntValue(alert.Value),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func flattenModules(modules []*databases.Module) []map[string]interface{} {

	var tfs = make([]map[string]interface{}, 0)
	for _, module := range modules {

		tf := map[string]interface{}{
			"name": redis.StringValue(module.Name),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func flattenRegexRules(rules []*databases.RegexRule) []string {
	ret := make([]string, len(rules))
	for _, rule := range rules {
		ret[rule.Ordinal] = rule.Pattern
	}

	if len(ret) == 2 && ret[0] == ".*\\{(?<tag>.*)\\}.*" && ret[1] == "(?<tag>.*)" {
		// This is the default regex rules - https://docs.redislabs.com/latest/rc/concepts/clustering/#custom-hashing-policy
		return []string{}
	}

	return ret
}

func getDatabaseNameIdMap(ctx context.Context, subId int, client *apiClient) (map[string]int, error) {
	ret := map[string]int{}
	list := client.client.Database.List(ctx, subId)
	for list.Next() {
		db := list.Value()
		ret[redis.StringValue(db.Name)] = redis.IntValue(db.ID)
	}
	if list.Err() != nil {
		return nil, list.Err()
	}
	return ret, nil
}

// diff: Checks the difference between two Sets based on their hash keys and check if they were modified by generating
//       a hash based on their attributes.
func diff(oldSet *schema.Set, newSet *schema.Set, hashKey func(interface{}) string) ([]map[string]interface{}, []map[string]interface{}, []map[string]interface{}) {

	oldHashedMap := map[string]*hashedSet{}
	newHashedMap := map[string]*hashedSet{}

	for _, v := range oldSet.List() {
		h := hashedSet{}
		oldHashedMap[hashKey(v)] = h.init(oldSet, v)
	}
	for _, v := range newSet.List() {
		h := hashedSet{}
		newHashedMap[hashKey(v)] = h.init(newSet, v)
	}

	var addition, existing, deletion []map[string]interface{}

	for k, newVal := range newHashedMap {
		// Check if we're updating an existing block.
		if oldVal, ok := oldHashedMap[k]; ok {
			// The hashes are the same - this block has NOT been changed.
			if oldVal.hash == newVal.hash {
				continue
			}
			// The hashes are different - this block was modified.
			existing = append(existing, newVal.m)
			// This block was recently added.
		} else {
			addition = append(addition, newVal.m)
		}
	}

	for k, oldVal := range oldHashedMap {
		// This block was deleted.
		if _, ok := newHashedMap[k]; !ok {
			deletion = append(deletion, oldVal.m)
		}
	}

	return addition, existing, deletion
}
