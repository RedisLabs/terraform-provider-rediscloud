package provider

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/regions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudActiveActiveRegion() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates an Active Active Region and within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActiveRegionCreate,
		ReadContext:   resourceRedisCloudActiveActiveRegionRead,
		UpdateContext: resourceRedisCloudActiveActiveRegionUpdate,
		DeleteContext: resourceRedisCloudActiveActiveRegionDelete,
		Importer: &schema.ResourceImporter{
			// Let the READ operation do the heavy lifting for importing values from the API.
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "ID of the subscription that the regions belong to",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
			},
			"delete_regions": {
				Description: "Delete regions flag has to be set for re-creating and deleting regions",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"region": {
				Description: "Cloud networking details, per region (multiple regions for Active-Active cluster)",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region_id": {
							Description: "The region id",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"region": {
							Description: "Deployment region as defined by cloud provider",
							Type:        schema.TypeString,
							Required:    true,
						},
						"vpc_id": {
							Description: "Identifier of the VPC to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"recreate_region": {
							Description: "Defines wheter the regions should be re-created",
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
						},
						"networking_deployment_cidr": {
							Description:      "Deployment CIDR mask",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
						},

						"database": {
							Description: "The database resource",
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Description: "A numeric id for the database",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"database_name": {
										Description: "A meaningful name to identify the database",
										Type:        schema.TypeString,
										Required:    true,
									},
									"local_write_operations_per_second": {
										Description: "Write operations per second for creation plan databases",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"local_read_operations_per_second": {
										Description: "Write operations per second for creation plan databases",
										Type:        schema.TypeInt,
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudActiveActiveRegionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Get existing regions so we can do a manual diff
	// Query API for existing Regions for a given Subscription
	existingRegions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create an existingRegionMap<regionName, region>
	existingRegionMap := make(map[string]*regions.Region)
	for _, existingRegion := range existingRegions.Regions {
		existingRegionMap[*existingRegion.Region] = existingRegion
	}

	regionsFromResourceData := buildCreateActiveActiveRegions(d.Get("region").(*schema.Set))

	// Filter non-existing regions
	createRegions := make([]*regions.Region, 0)
	for _, currentRegion := range regionsFromResourceData {
		if _, ok := existingRegionMap[*currentRegion.Region]; !ok {
			createRegions = append(createRegions, currentRegion)
		}
	}

	err = regionCreate(subId, createRegions, existingRegionMap, regionsFromResourceData, ctx, d, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func regionCreate(subId int, createRegions []*regions.Region, existingRegionMap map[string]*regions.Region, regionsFromResourceData []*regions.Region, ctx context.Context, d *schema.ResourceData, api *apiClient) error {
	// If no new regions were defined return
	if len(createRegions) == 0 {
		return nil
	}

	// Call GO API createRegion for all non-existing regions
	for _, currentRegion := range createRegions {
		createDatabases := make([]*regions.CreateDatabase, 0)
		for _, database := range currentRegion.Databases {
			localThroughputMeasurement := regions.CreateLocalThroughput{
				Region:                   currentRegion.Region,
				ReadOperationsPerSecond:  database.ReadOperationsPerSecond,
				WriteOperationsPerSecond: database.WriteOperationsPerSecond,
			}
			createDatabase := regions.CreateDatabase{
				Name:                       database.DatabaseName,
				LocalThroughputMeasurement: &localThroughputMeasurement,
			}
			createDatabases = append(createDatabases, &createDatabase)
		}

		createRegion := regions.CreateRegion{
			Region:         currentRegion.Region,
			DeploymentCIDR: currentRegion.DeploymentCIDR,
			Databases:      createDatabases,
		}

		_, err := api.client.Regions.Create(ctx, subId, createRegion)

		if err != nil {
			return err
		}
	}

	d.SetId(strconv.Itoa(subId))

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	// Wait for the subscription to be active before deleting it.
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return err
	}

	// There is a timing issue where the subscription is marked as active before the creation-plan databases are deleted.
	// This additional wait ensures that the databases are deleted before the subscription is deleted.
	time.Sleep(10 * time.Second) //lintignore:R018
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return err
	}

	return nil
}

func resourceRedisCloudActiveActiveRegionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)
	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// Get existing regions so we can do a manual diff
	// Query API for existing Regions for a given Subscription
	existingRegions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create an existingRegionMap<regionName, region>
	existingRegionMap := make(map[string]*regions.Region)
	for _, existingRegion := range existingRegions.Regions {
		existingRegionMap[*existingRegion.Region] = existingRegion
	}

	regionsFromResourceData := buildCreateActiveActiveRegions(d.Get("region").(*schema.Set))

	// Handling Delete Regions
	deleteRegions := make([]*regions.Region, 0)
	regionsToKeepMap := make(map[string]*regions.Region)
	for _, regionToKeep := range regionsFromResourceData {
		regionsToKeepMap[*regionToKeep.Region] = regionToKeep
	}

	for _, currentRegion := range existingRegions.Regions {
		if _, ok := regionsToKeepMap[*currentRegion.Region]; !ok {
			deleteRegions = append(deleteRegions, currentRegion)
		}
	}

	deleteRegionsFlag := d.Get("delete_regions").(bool)
	// Validations
	if len(deleteRegions) > 0 && !deleteRegionsFlag {
		return diag.Errorf("Region has been removed, but delete_regions flag was not set!")
	}

	for _, currentRegion := range regionsFromResourceData {
		if !*currentRegion.RecreateRegion && *existingRegionMap[*currentRegion.Region].DeploymentCIDR != *currentRegion.DeploymentCIDR {
			return diag.Errorf("Region %s needs to be recreated but recreate_region flag was not set!", *currentRegion.Region)
		}
	}

	if deleteRegionsFlag && len(deleteRegions) > 0 {
		regiondDelete(ctx, d, subId, deleteRegions, meta)
		// Updating existing ragion map
		for _, removedRegion := range deleteRegions {
			delete(existingRegionMap, *removedRegion.Region)
		}
	}

	// Handling region create
	// Filter non-existing regions
	createRegions := make([]*regions.Region, 0)
	for _, currentRegion := range regionsFromResourceData {
		if _, ok := existingRegionMap[*currentRegion.Region]; !ok {
			createRegions = append(createRegions, currentRegion)
		}
	}
	if len(createRegions) > 0 {
		err := regionCreate(subId, createRegions, existingRegionMap, regionsFromResourceData, ctx, d, api)
		if err != nil {
			return diag.FromErr(err)
		}

		// Updating existing ragion map
		for _, newRegion := range createRegions {
			existingRegionMap[*newRegion.Region] = newRegion
		}
	}

	// Handling region re-create
	reCreateRegions := make([]*regions.Region, 0)
	for _, currentRegion := range regionsFromResourceData {
		existingRegion := existingRegionMap[*currentRegion.Region]
		if !cmp.Equal(existingRegion, currentRegion) {
			if shouldRecreateRegion(existingRegion, currentRegion, deleteRegionsFlag) {
				reCreateRegions = append(reCreateRegions, currentRegion)
			}
		}
	}

	if len(reCreateRegions) > 0 {
		regiondDelete(ctx, d, subId, reCreateRegions, meta)
		resourceRedisCloudActiveActiveRegionCreate(ctx, d, meta)
	}

	// Handling DB updates
	err = performDbUpdates(ctx, api, subId, regionsFromResourceData, existingRegionMap)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func performDbUpdates(ctx context.Context, api *apiClient, subId int, regionsFromResourceData []*regions.Region, existingRegionMap map[string]*regions.Region) error {
	updateDBMap := make(map[int][]*databases.LocalRegionProperties)
	for _, currentRegion := range regionsFromResourceData {
		existingRegion := existingRegionMap[*currentRegion.Region]
		if shouldUpdateDatabaseOnly(existingRegion, currentRegion) {
			//Collect databases to a map <dbId, db>
			existingDBMap := make(map[int]*regions.Database)
			for _, db := range existingRegion.Databases {
				existingDBMap[*db.DatabaseId] = db
			}

			for _, db := range currentRegion.Databases {
				if !cmp.Equal(db, existingDBMap[*db.DatabaseId]) {
					localThroughput := databases.LocalThroughput{
						Region:                   currentRegion.Region,
						WriteOperationsPerSecond: db.WriteOperationsPerSecond,
						ReadOperationsPerSecond:  db.ReadOperationsPerSecond,
					}
					localRegionProperty := databases.LocalRegionProperties{
						Region:                     currentRegion.Region,
						LocalThroughputMeasurement: &localThroughput,
					}
					updateDBMap[*db.DatabaseId] = append(updateDBMap[*db.DatabaseId], &localRegionProperty)
				}
			}
		}
	}

	if len(updateDBMap) > 0 {
		for dbId, localRegionProperties := range updateDBMap {
			updateActiveActiveDb := databases.UpdateActiveActiveDatabase{
				Regions: localRegionProperties,
			}
			err := api.client.Database.ActiveActiveUpdate(ctx, subId, dbId, updateActiveActiveDb)
			if err != nil {
				return err
			}
		}
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	// Wait for the subscription to be active before deleting it.
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return err
	}

	// There is a timing issue where the subscription is marked as active before the creation-plan databases are deleted.
	// This additional wait ensures that the databases are deleted before the subscription is deleted.
	time.Sleep(10 * time.Second) //lintignore:R018
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return err
	}

	return nil
}

func resourceRedisCloudActiveActiveRegionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	regions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		if _, ok := err.(*subscriptions.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("subscription_id", strconv.Itoa(*regions.SubscriptionId)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("region", buildActiveActiveRegionsResourceData(regions.Regions, d)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

// Does nothing
func resourceRedisCloudActiveActiveRegionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceRedisCloudActiveActiveRegionRead(ctx, d, meta)
}

func regiondDelete(ctx context.Context, d *schema.ResourceData, subId int, regionsToDelete []*regions.Region, meta interface{}) error {
	// use the meta value to retrieve your client from the provider configure method
	api := meta.(*apiClient)

	var deleteRegionArray []*regions.DeleteRegion
	for _, region := range regionsToDelete {
		deleteRegion := regions.DeleteRegion{
			Region: region.Region,
		}
		deleteRegionArray = append(deleteRegionArray, &deleteRegion)
	}

	deleteRegions := regions.DeleteRegions{
		Regions: deleteRegionArray,
	}

	err := api.client.Regions.DeleteWithQuery(ctx, subId, deleteRegions)
	if err != nil {
		return err
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	// Wait for the subscription to be active before deleting it.
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return err
	}

	// There is a timing issue where the subscription is marked as active before the creation-plan databases are deleted.
	// This additional wait ensures that the databases are deleted before the subscription is deleted.
	time.Sleep(10 * time.Second) //lintignore:R018
	if err := waitForSubscriptionToBeActive(ctx, subId, api); err != nil {
		return err
	}

	return nil
}

func buildActiveActiveRegionsResourceData(regions []*regions.Region, d *schema.ResourceData) []map[string]interface{} {
	var resourceDataMap []map[string]interface{}

	resDataRegionsMap := d.Get("region").(*schema.Set)
	regionByRegionIdMap := make(map[string]bool)
	for _, resDataRegion := range resDataRegionsMap.List() {
		resDataRegionMap := resDataRegion.(map[string]interface{})
		regionByRegionIdMap[resDataRegionMap["region"].(string)] = resDataRegionMap["recreate_region"].(bool)
	}

	for _, currentRegion := range regions {
		var databases []interface{}
		for _, database := range currentRegion.Databases {
			datbaseMapString := map[string]interface{}{
				"id":                                database.DatabaseId,
				"database_name":                     database.DatabaseName,
				"local_read_operations_per_second":  database.ReadOperationsPerSecond,
				"local_write_operations_per_second": database.WriteOperationsPerSecond,
			}
			databases = append(databases, datbaseMapString)
		}

		regionMapString := map[string]interface{}{
			"region_id":                  currentRegion.RegionId,
			"region":                     currentRegion.Region,
			"recreate_region":            regionByRegionIdMap[*currentRegion.Region],
			"networking_deployment_cidr": currentRegion.DeploymentCIDR,
			"vpc_id":                     currentRegion.VpcId,
			"database":                   databases,
		}
		resourceDataMap = append(resourceDataMap, regionMapString)
	}

	return resourceDataMap
}

func buildCreateActiveActiveRegions(r *schema.Set) []*regions.Region {
	createRegions := make([]*regions.Region, 0)
	for _, region := range r.List() {
		regionMap := region.(map[string]interface{})

		// CreateDatabases
		createDatabases := make([]*regions.Database, 0)
		if databases := regionMap["database"].(*schema.Set).List(); len(databases) != 0 {
			for _, database := range databases {
				databaseMap := database.(map[string]interface{})
				createDatabase := regions.Database{
					DatabaseId:               redis.Int(databaseMap["id"].(int)),
					DatabaseName:             redis.String(databaseMap["database_name"].(string)),
					ReadOperationsPerSecond:  redis.Int(databaseMap["local_read_operations_per_second"].(int)),
					WriteOperationsPerSecond: redis.Int(databaseMap["local_write_operations_per_second"].(int)),
				}
				createDatabases = append(createDatabases, &createDatabase)
			}
		}

		createRegion := regions.Region{
			Region:         redis.String(regionMap["region"].(string)),
			RecreateRegion: redis.Bool(regionMap["recreate_region"].(bool)),
			DeploymentCIDR: redis.String(regionMap["networking_deployment_cidr"].(string)),
			VpcId:          redis.String(regionMap["vpc_id"].(string)),
			Databases:      createDatabases,
		}

		createRegions = append(createRegions, &createRegion)
	}

	return createRegions
}

func shouldRecreateRegion(existingRegion *regions.Region, resourceDataRegion *regions.Region, deleteRegionsFlag bool) bool {
	return (*existingRegion.DeploymentCIDR != *resourceDataRegion.DeploymentCIDR) && *resourceDataRegion.RecreateRegion && deleteRegionsFlag
}

func shouldUpdateDatabaseOnly(existingRegion *regions.Region, resourceDataRegion *regions.Region) bool {
	return *existingRegion.DeploymentCIDR == *resourceDataRegion.DeploymentCIDR && !cmp.Equal(existingRegion.Databases, resourceDataRegion.Databases) && !*resourceDataRegion.RecreateRegion
}
