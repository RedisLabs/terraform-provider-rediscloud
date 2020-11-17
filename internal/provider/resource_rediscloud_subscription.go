package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	"regexp"
	"strconv"
	"time"
)

func resourceRedisCloudSubscription() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRedisCloudSubscriptionCreate,
		ReadContext:   resourceRedisCloudSubscriptionRead,
		UpdateContext: resourceRedisCloudSubscriptionUpdate,
		DeleteContext: resourceRedisCloudSubscriptionDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"payment_method_id": {
				Type:             schema.TypeString,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				Optional:         true,
			},
			"memory_storage": {
				Type:             schema.TypeString,
				Optional:         true,
				ForceNew:         true,
				Default:          "ram",
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.MemoryStorageValues(), false)),
			},
			"persistent_storage_encryption": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Optional: true,
				Default:  false,
			},
			"cloud_provider": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							Default:          "AWS",
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
						},
						"cloud_account_id": {
							Type:             schema.TypeString,
							Optional:         true,
							ForceNew:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
							Default:          "1",
						},
						"region": {
							Type:     schema.TypeSet,
							Required: true,
							ForceNew: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
									"multiple_availability_zones": {
										Type:     schema.TypeBool,
										ForceNew: true,
										Optional: true,
										Default:  false,
									},
									"preferred_availability_zones": {
										Type: schema.TypeList,
										// TODO it should be possible to optionally set this
										Computed: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_deployment_cidr": {
										Type: schema.TypeString,
										// TODO this needs to be ForceNew as it can't be updated, but cannot also be Computed
										// TODO need to see what the returned value is when only using redis internal account
										Optional:         true,
										Computed:         true,
										ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
									},
									"networking_vpc_id": {
										Type:     schema.TypeString,
										ForceNew: true,
										Optional: true,
									},
									"networking_subnet_id": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"database": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"db_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.ProtocolValues(), false)),
						},
						"memory_limit_in_gb": {
							Type:     schema.TypeFloat,
							Required: true,
						},
						"support_oss_cluster_api": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"data_persistence": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "none",
						},
						"replication": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"throughput_measurement_by": {
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice([]string{"number-of-shards", "operations-per-second"}, false)),
						},
						"throughput_measurement_value": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"average_item_size_in_bytes": {
							Type:     schema.TypeInt,
							Optional: true,
							// Setting default to 0 so that the hash func produces the same hash when this field is not
							// specified. SDK's catch-all issue around this: https://github.com/hashicorp/terraform-plugin-sdk/issues/261
							Default: 0,
						},
						"password": {
							Type:      schema.TypeString,
							Required:  true,
							Sensitive: true,
						},
						"public_endpoint": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"private_endpoint": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"module": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"source_ips": {
							Type:     schema.TypeSet,
							Optional: true,
							MinItems: 1,
							Elem: &schema.Schema{
								Type:             schema.TypeString,
								ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
							},
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	// Create CloudProviders
	providers, err := buildCreateCloudProviders(d.Get("cloud_provider"))
	if err != nil {
		return diag.FromErr(err)
	}

	// Create databases
	dbs := buildSubscriptionCreateDatabases(d.Get("database"))

	// Create Subscription
	name := d.Get("name").(string)
	paymentMethodID, err := strconv.Atoi(d.Get("payment_method_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	memoryStorage := d.Get("memory_storage").(string)
	persistentStorageEncryption := d.Get("persistent_storage_encryption").(bool)

	createSubscriptionRequest := subscriptions.CreateSubscription{
		Name:                        redis.String(name),
		DryRun:                      redis.Bool(false),
		PaymentMethodID:             redis.Int(paymentMethodID),
		MemoryStorage:               redis.String(memoryStorage),
		PersistentStorageEncryption: redis.Bool(persistentStorageEncryption),
		CloudProviders:              providers,
		Databases:                   dbs,
	}

	subId, err := api.client.Subscription.Create(ctx, createSubscriptionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	// Confirm Subscription Active status
	err = waitForSubscriptionToBeActive(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	// Locate Databases to confirm Active status
	dbList := api.client.Database.List(ctx, subId)

	for dbList.Next() {
		dbId := *dbList.Value().ID

		if err := waitForDatabaseToBeActive(ctx, subId, dbId, api); err != nil {
			return diag.FromErr(err)
		}
	}
	if dbList.Err() != nil {
		return diag.FromErr(dbList.Err())
	}

	// Some attributes on a database are not accessible by the subscription creation API.
	// Run the subscription update function to apply any additional changes to the databases, such as password and so on.
	return resourceRedisCloudSubscriptionUpdate(ctx, d, meta)
}

func resourceRedisCloudSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	subscription, err := api.client.Subscription.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("name", redis.StringValue(subscription.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("payment_method_id", strconv.Itoa(redis.IntValue(subscription.PaymentMethodID))); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("memory_storage", redis.StringValue(subscription.MemoryStorage)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("persistent_storage_encryption", redis.BoolValue(subscription.StorageEncryption)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("cloud_provider", flattenCloudDetails(subscription.CloudDetails)); err != nil {
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

	if d.HasChanges("name", "payment_method_id") {
		updateSubscriptionRequest := subscriptions.UpdateSubscription{}

		if d.HasChange("name") {
			name := d.Get("name").(string)
			updateSubscriptionRequest.Name = &name
		}

		if d.HasChange("payment_method_id") {
			paymentMethodID, err := strconv.Atoi(d.Get("payment_method_id").(string))
			if err != nil {
				return diag.FromErr(err)
			}

			updateSubscriptionRequest.PaymentMethodID = &paymentMethodID
		}

		err = api.client.Subscription.Update(ctx, subId, updateSubscriptionRequest)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange("database") || d.IsNewResource() {

		nameId, err := getDatabaseNameIdMap(ctx, subId, api)
		if err != nil {
			return diag.FromErr(err)
		}

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
		}

		for _, db := range existing {
			update := buildUpdateDatabase(db)
			dbId := nameId[redis.StringValue(update.Name)]

			log.Printf("[DEBUG] Updating database %d", dbId)

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

func resourceRedisCloudSubscriptionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	nameId, err := getDatabaseNameIdMap(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, v := range d.Get("database").(*schema.Set).List() {
		database := v.(map[string]interface{})

		name := database["name"].(string)
		if id, ok := nameId[name]; ok {
			log.Printf("[DEBUG] Deleting database %d on subscription %d", id, subId)

			dbErr := api.client.Database.Delete(ctx, subId, id)
			if dbErr != nil {
				diag.FromErr(dbErr)
			}
		} else {
			log.Printf("[DEBUG] Database %s no longer exists", name)
		}
	}

	// Delete subscription once all databases are deleted
	err = api.client.Subscription.Delete(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	err = waitForSubscriptionToBeDeleted(ctx, subId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func buildCreateCloudProviders(providers interface{}) ([]*subscriptions.CreateCloudProvider, error) {
	createCloudProviders := make([]*subscriptions.CreateCloudProvider, 0)

	for _, provider := range providers.([]interface{}) {
		providerMap := provider.(map[string]interface{})

		providerStr := providerMap["provider"].(string)
		cloudAccountID, err := strconv.Atoi(providerMap["cloud_account_id"].(string))
		if err != nil {
			return nil, err
		}

		createRegions := make([]*subscriptions.CreateRegion, 0)
		if regions := providerMap["region"].(*schema.Set).List(); len(regions) != 0 {

			for _, region := range regions {
				regionMap := region.(map[string]interface{})

				regionStr := regionMap["region"].(string)
				multipleAvailabilityZones := regionMap["multiple_availability_zones"].(bool)

				createRegion := subscriptions.CreateRegion{
					Region:                    redis.String(regionStr),
					MultipleAvailabilityZones: redis.Bool(multipleAvailabilityZones),
				}

				if v, ok := regionMap["networking_deployment_cidr"]; ok && v != "" {
					createRegion.Networking = &subscriptions.CreateNetworking{
						DeploymentCIDR: redis.String(v.(string)),
					}
				}

				if v, ok := regionMap["networking_vpc_id"]; ok && v != "" {
					if createRegion.Networking == nil {
						createRegion.Networking = &subscriptions.CreateNetworking{}
					}
					createRegion.Networking.VPCId = redis.String(v.(string))
				}

				createRegions = append(createRegions, &createRegion)
			}
		}

		createCloudProvider := &subscriptions.CreateCloudProvider{
			Provider:       redis.String(providerStr),
			CloudAccountID: redis.Int(cloudAccountID),
			Regions:        createRegions,
		}

		createCloudProviders = append(createCloudProviders, createCloudProvider)
	}

	return createCloudProviders, nil
}

func buildSubscriptionCreateDatabases(databases interface{}) []*subscriptions.CreateDatabase {
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
		modules := databaseMap["module"]
		for _, module := range modules.(*schema.Set).List() {
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
		Password: redis.String(db["password"].(string)),
		SourceIP: toStringSlice(db["source_ips"].(*schema.Set)),
	}

	averageItemSize := db["average_item_size_in_bytes"].(int)
	if averageItemSize > 0 {
		create.AverageItemSizeInBytes = redis.Int(averageItemSize)
	}

	return create
}

func buildUpdateDatabase(db map[string]interface{}) databases.UpdateDatabase {
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
		SourceIP:        toStringSlice(db["source_ips"].(*schema.Set)),
	}

	return update
}

func waitForSubscriptionToBeActive(ctx context.Context, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusPending},
		Target:  []string{subscriptions.SubscriptionStatusActive},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be active", id)

			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func waitForSubscriptionToBeDeleted(ctx context.Context, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{subscriptions.SubscriptionStatusDeleting},
		Target:  []string{"deleted"},
		Timeout: 10 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for subscription %d to be deleted", id)

			subscription, err := api.client.Subscription.Get(ctx, id)
			if err != nil {
				if _, ok := err.(*subscriptions.NotFound); ok {
					return "deleted", "deleted", nil
				}
				return nil, "", err
			}

			return redis.StringValue(subscription.Status), redis.StringValue(subscription.Status), nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
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

func flattenCloudDetails(cloudDetails []*subscriptions.CloudDetail) []map[string]interface{} {
	var cdl []map[string]interface{}

	for _, currentCloudDetail := range cloudDetails {

		var regions []interface{}
		for _, currentRegion := range currentCloudDetail.Regions {

			regionMapString := map[string]interface{}{
				"region":                       currentRegion.Region,
				"multiple_availability_zones":  currentRegion.MultipleAvailabilityZones,
				"preferred_availability_zones": currentRegion.PreferredAvailabilityZones,
				"networking_deployment_cidr":   currentRegion.Networking[0].DeploymentCIDR,
				"networking_vpc_id":            currentRegion.Networking[0].VPCId,
				"networking_subnet_id":         currentRegion.Networking[0].SubnetID,
			}

			regions = append(regions, regionMapString)
		}

		cdlMapString := map[string]interface{}{
			"provider":         currentCloudDetail.Provider,
			"cloud_account_id": strconv.Itoa(redis.IntValue(currentCloudDetail.CloudAccountID)),
			"region":           regions,
		}
		cdl = append(cdl, cdlMapString)
	}

	return cdl
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

		averageItemSize := database["average_item_size_in_bytes"].(int)
		existingPassword := database["password"].(string)
		existingSourceIPs := database["source_ips"].(*schema.Set)

		flattened = append(flattened, flattenDatabase(averageItemSize, existingPassword, existingSourceIPs, db))
	}
	return flattened, nil
}

func flattenDatabase(averageItemSizeInBytes int, existingPassword string, existingSourceIp *schema.Set, db *databases.Database) map[string]interface{} {
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
		"db_id":                        redis.IntValue(db.ID),
		"name":                         redis.StringValue(db.Name),
		"protocol":                     redis.StringValue(db.Protocol),
		"memory_limit_in_gb":           redis.Float64Value(db.MemoryLimitInGB),
		"support_oss_cluster_api":      redis.BoolValue(db.SupportOSSClusterAPI),
		"data_persistence":             redis.StringValue(db.DataPersistence),
		"replication":                  redis.BoolValue(db.Replication),
		"throughput_measurement_by":    redis.StringValue(db.ThroughputMeasurement.By),
		"throughput_measurement_value": redis.IntValue(db.ThroughputMeasurement.Value),
		"public_endpoint":              redis.StringValue(db.PublicEndpoint),
		"private_endpoint":             redis.StringValue(db.PrivateEndpoint),
		"module":                       flattenModules(db.Modules),
		"password":                     password,
		"source_ips":                   sourceIPs,
	}

	if averageItemSizeInBytes > 0 {
		tf["average_item_size_in_bytes"] = averageItemSizeInBytes
	}

	return tf
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

func diff(oldSet *schema.Set, newSet *schema.Set, hashKey func(interface{}) string) ([]map[string]interface{}, []map[string]interface{}, []map[string]interface{}) {
	oldMap := map[string]map[string]interface{}{}
	newMap := map[string]map[string]interface{}{}

	for _, v := range oldSet.List() {
		oldMap[hashKey(v)] = v.(map[string]interface{})
	}
	for _, v := range newSet.List() {
		newMap[hashKey(v)] = v.(map[string]interface{})
	}

	var addition, existing, deletion []map[string]interface{}

	for k, v := range newMap {
		if _, ok := oldMap[k]; ok {
			existing = append(existing, v)
		} else {
			addition = append(addition, v)
		}
	}

	for k, v := range oldMap {
		if _, ok := newMap[k]; !ok {
			deletion = append(deletion, v)
		}
	}

	return addition, existing, deletion
}
