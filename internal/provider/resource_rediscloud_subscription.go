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
			Create: schema.DefaultTimeout(10 * time.Minute),
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
				Default:          "ram",
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(databases.MemoryStorage(), false)), // TODO shoudl be MemoryStorageValues()
			},
			"persistent_storage_encryption": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"cloud_providers": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provider": {
							Type:             schema.TypeString,
							Optional:         true,
							Default:          "AWS",
							ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
						},
						"cloud_account_id": {
							Type:             schema.TypeString,
							Optional:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
							Default:          "1",
						},
						"regions": {
							Type:     schema.TypeSet,
							Required: true,
							MinItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"region": {
										Type:     schema.TypeString,
										Required: true,
									},
									"multiple_availability_zones": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"preferred_availability_zones": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"networking_deployment_cidr": {
										Type:             schema.TypeString,
										Optional:         true,
										ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
									},
									"networking_vpc_id": {
										Type:     schema.TypeString,
										Optional: true,
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
				Set: func(v interface{}) int {
					m := v.(map[string]interface{})
					return m["db_id"].(int)
				},
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
						// TODO modules support - note that certain modules conflict with certain values of throughput_measurement_by
						"average_item_size_in_bytes": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"password": {
							Type:      schema.TypeString,
							Computed:  true,
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
					},
				},
			},
		},
	}
}

func resourceRedisCloudSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	// Create CloudProviders
	providers := buildCreateCloudProviders(d.Get("cloud_providers"))

	// Create databases
	databases := buildCreateDatabases(d.Get("database"))

	// Create Subscription
	name := d.Get("name").(string)
	paymentMethodID, _ := strconv.Atoi(d.Get("payment_method_id").(string)) // TODO
	memoryStorage := d.Get("memory_storage").(string)
	persistentStorageEncryption := d.Get("persistent_storage_encryption").(bool)

	createSubscriptionRequest := subscriptions.CreateSubscription{
		Name:                        redis.String(name),
		DryRun:                      redis.Bool(false),
		PaymentMethodID:             redis.Int(paymentMethodID),
		MemoryStorage:               redis.String(memoryStorage),
		PersistentStorageEncryption: redis.Bool(persistentStorageEncryption),
		CloudProviders:              providers,
		Databases:                   databases,
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
		database := dbList.Value()

		err := waitForDatabaseToBeActive(ctx, subId, *database.ID, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if dbList.Err() != nil {
		return diag.FromErr(dbList.Err())
	}

	return resourceRedisCloudSubscriptionRead(ctx, d, meta)
	// TODO need to run database update to modify values that can only be accessed through creating/updating a database
}

func resourceRedisCloudSubscriptionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Read subscription, (also returned cloud providers/Details)
	subscription, err := api.client.Subscription.Get(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", redis.StringValue(subscription.Name))
	d.Set("payment_method_id", strconv.Itoa(redis.IntValue(subscription.PaymentMethodID)))
	d.Set("memory_storage", redis.StringValue(subscription.MemoryStorage))

	d.Set("cloud_providers", flattenCloudDetails(subscription.CloudDetails))

	// Read databases that are not returned with Subscription
	databaseList := api.client.Database.List(ctx, subId)

	d.Set("database", flattenDatabases(*databaseList, d.Get("database").(*schema.Set)))

	return diags
}

func resourceRedisCloudSubscriptionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	updateSubscriptionRequest := subscriptions.UpdateSubscription{}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		updateSubscriptionRequest.Name = &name
	}

	if d.HasChange("payment_method_id") {
		paymentMethodID, _ := strconv.Atoi(d.Get("payment_method_id").(string)) // TODO
		updateSubscriptionRequest.PaymentMethodID = &paymentMethodID
	}

	// TODO this really should only be done if there are changes
	err = api.client.Subscription.Update(ctx, subId, updateSubscriptionRequest)
	if err != nil {
		return diag.FromErr(err)
	}

	if d.HasChange("database") {

		for _, database := range d.Get("database").(*schema.Set).List() {
			databaseMap := database.(map[string]interface{})

			dbId := databaseMap["db_id"].(int)
			name := databaseMap["name"].(string)
			memoryLimitInGB := databaseMap["memory_limit_in_gb"].(float64)
			supportOSSClusterAPI := databaseMap["support_oss_cluster_api"].(bool)
			replication := databaseMap["replication"].(bool)
			throughputMeasurementBy := databaseMap["throughput_measurement_by"].(string)
			throughputMeasurementValue := databaseMap["throughput_measurement_value"].(int)
			dataPersistence := databaseMap["data_persistence"].(string)

			err := api.client.Database.Update(ctx, subId, dbId, databases.UpdateDatabase{
				Name:                 redis.String(name),
				MemoryLimitInGB:      redis.Float64(memoryLimitInGB),
				SupportOSSClusterAPI: redis.Bool(supportOSSClusterAPI),
				Replication:          redis.Bool(replication),
				ThroughputMeasurement: &databases.UpdateThroughputMeasurement{
					By:    redis.String(throughputMeasurementBy),
					Value: redis.Int(throughputMeasurementValue),
				},
				DataPersistence: redis.String(dataPersistence),
			})

			if err != nil {
				return diag.FromErr(err)
			}
		}
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

	// Locate databases for sub and delete.
	databases := api.client.Database.List(ctx, subId)

	for databases.Next() {

		database := databases.Value()
		databaseId := database.ID

		log.Printf("[DEBUG] Deleting database %d on subscription %d", databaseId, subId)

		dbErr := api.client.Database.Delete(ctx, subId, *databaseId)
		if dbErr != nil {
			diag.FromErr(dbErr)
		}
	}
	if databases.Err() != nil {
		diag.FromErr(databases.Err())
	}

	// Delete subscription once all databases are deleted
	err = api.client.Subscription.Delete(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func buildCreateCloudProviders(providers interface{}) []*subscriptions.CreateCloudProvider {
	createCloudProviders := make([]*subscriptions.CreateCloudProvider, 0)

	for _, provider := range providers.(*schema.Set).List() {
		providerMap := provider.(map[string]interface{})

		providerStr := providerMap["provider"].(string)
		cloudAccountID, _ := strconv.Atoi(providerMap["cloud_account_id"].(string)) // TODO

		createRegions := make([]*subscriptions.CreateRegion, 0)
		if regions := providerMap["regions"].(*schema.Set).List(); len(regions) != 0 {

			for _, region := range regions {
				regionMap := region.(map[string]interface{})

				regionStr := regionMap["region"].(string)
				multipleAvailabilityZones := regionMap["multiple_availability_zones"].(bool)

				createRegion := subscriptions.CreateRegion{
					Region:                    redis.String(regionStr),
					MultipleAvailabilityZones: redis.Bool(multipleAvailabilityZones),
					//PreferredAvailabilityZones: []string{"eu-west-1"},
				}

				if v, ok := regionMap["networking_deployment_cidr"]; ok {
					createRegion.Networking = &subscriptions.CreateNetworking{
						DeploymentCIDR: redis.String(v.(string)),
						//VPCId:          mRegion["networking_vpc_id"].(string),
					}
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

	return createCloudProviders
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
		}

		if averageItemSizeInBytes > 0 {
			createDatabase.AverageItemSizeInBytes = &averageItemSizeInBytes
		}

		createDatabases = append(createDatabases, createDatabase)
	}

	return createDatabases
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

func waitForDatabaseToBeActive(ctx context.Context, subId, id int, api *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{databases.StatusDraft},
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

		var regions []map[string]interface{}
		for _, currentRegion := range currentCloudDetail.Regions {

			regionMapString := map[string]interface{}{
				"region":                      redis.StringValue(currentRegion.Region),
				"multiple_availability_zones": redis.BoolValue(currentRegion.MultipleAvailabilityZones),
				//"preferred_availability_zones": currentRegion.PreferredAvailabilityZones
				"networking_deployment_cidr": redis.StringValue(currentRegion.Networking[0].DeploymentCIDR),
				"networking_vpc_id":          redis.StringValue(currentRegion.Networking[0].VPCId),
			}
			regions = append(regions, regionMapString)
		}

		cdlMapString := map[string]interface{}{
			"provider":         redis.StringValue(currentCloudDetail.Provider),
			"cloud_account_id": strconv.Itoa(redis.IntValue(currentCloudDetail.CloudAccountID)),
			"regions":          regions,
		}
		cdl = append(cdl, cdlMapString)
	}

	return cdl
}

func flattenDatabases(list databases.ListDatabase, databaseSet *schema.Set) []map[string]interface{} {
	var dbl []map[string]interface{}

	for list.Next() {

		currentDatabase := list.Value()

		dbMapString := map[string]interface{}{
			"db_id":                        redis.IntValue(currentDatabase.ID),
			"name":                         redis.StringValue(currentDatabase.Name),
			"protocol":                     redis.StringValue(currentDatabase.Protocol),
			"memory_limit_in_gb":           redis.Float64Value(currentDatabase.MemoryLimitInGB),
			"support_oss_cluster_api":      redis.BoolValue(currentDatabase.SupportOSSClusterAPI),
			"data_persistence":             redis.StringValue(currentDatabase.DataPersistence),
			"replication":                  redis.BoolValue(currentDatabase.Replication),
			"throughput_measurement_by":    redis.StringValue(currentDatabase.ThroughputMeasurement.By),
			"throughput_measurement_value": redis.IntValue(currentDatabase.ThroughputMeasurement.Value),
			"password":                     redis.StringValue(currentDatabase.Security.Password),
			"public_endpoint":              redis.StringValue(currentDatabase.PublicEndpoint),
			"private_endpoint":             redis.StringValue(currentDatabase.PrivateEndpoint),
		}

		averageItemSizeInBytes := locateAverageItemSizeInBytes(*currentDatabase.Name, databaseSet)
		if averageItemSizeInBytes > 0 {
			dbMapString["average_item_size_in_bytes"] = averageItemSizeInBytes
		}

		dbl = append(dbl, dbMapString)
	}

	return dbl
}

func locateAverageItemSizeInBytes(dbName string, databases *schema.Set) int {

	var averageItemSizeInBytes int

	for _, database := range databases.List() {
		databaseMap := database.(map[string]interface{})

		name := databaseMap["name"].(string)

		if name == dbName {
			return databaseMap["average_item_size_in_bytes"].(int)
		}
	}

	return averageItemSizeInBytes
}
