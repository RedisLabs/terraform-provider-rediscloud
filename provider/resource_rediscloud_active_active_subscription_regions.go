package provider

import (
	"context"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/regions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudActiveActiveSubscriptionRegions() *schema.Resource {
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
			Create: schema.DefaultTimeout(60 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description:      "ID of the subscription that the regions belong to",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
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
							Description: "Defines whether the regions should be re-created",
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
									"database_id": {
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
	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(strconv.Itoa(subId))

	return resourceRedisCloudActiveActiveRegionUpdate(ctx, d, meta)
}

func resourceRedisCloudActiveActiveRegionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	deleteRegionsFlag := d.Get("delete_regions").(bool)

	// Get existing regions, so we can do a manual diff
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

	desiredRegions := buildRegionsFromResourceData(d.Get("region").(*schema.Set))

	// Determine which regions currently exist but aren't in the config
	// These will need to be deleted
	regionsToDelete := make([]*regions.Region, 0)
	for _, r := range existingRegions.Regions {
		if _, ok := desiredRegions[*r.Region]; !ok {
			regionsToDelete = append(regionsToDelete, r)
		}
	}

	// Of the regions that are in the config, determine which are brand new and should be created, which already exist
	// but have changed and require recreating (if update not supported), and which have changed and require updates
	// (updating a region's DBs is supported)
	regionsToCreate := make([]*regions.Region, 0)
	regionsToRecreate := make([]*regions.Region, 0)
	regionsToUpdateDatabases := make([]*regions.Region, 0)
	for _, r := range desiredRegions {
		existingRegion, ok := existingRegionMap[*r.Region]
		if !ok {
			regionsToCreate = append(regionsToCreate, r)
		} else {
			if shouldRecreateRegion(r, existingRegion) {
				if !*r.RecreateRegion || !deleteRegionsFlag {
					return diag.Errorf("Region %s needs to be recreated but recreate_region flag was not set!", *r.Region)
				}
				regionsToRecreate = append(regionsToRecreate, r)
			} else if shouldUpdateRegionDatabases(r, existingRegion) {
				regionsToUpdateDatabases = append(regionsToUpdateDatabases, r)
			}
		}
	}

	if len(regionsToCreate) > 0 {
		err := regionsCreate(ctx, subId, regionsToCreate, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(regionsToRecreate) > 0 {
		err := regionsDelete(ctx, subId, regionsToRecreate, api)
		if err != nil {
			return diag.FromErr(err)
		}
		err = regionsCreate(ctx, subId, regionsToRecreate, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(regionsToUpdateDatabases) > 0 {
		err = regionsUpdateDatabases(ctx, subId, api, regionsToUpdateDatabases, existingRegionMap)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if len(regionsToDelete) > 0 {
		if !deleteRegionsFlag {
			return diag.Errorf("Region has been removed, but delete_regions flag was not set!")
		}

		err := regionsDelete(ctx, subId, regionsToDelete, api)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudActiveActiveRegionRead(ctx, d, meta)
}

func resourceRedisCloudActiveActiveRegionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subId, err := strconv.Atoi(d.Id())
	existingRegions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		if _, ok := err.(*subscriptions.NotFound); ok {
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if err := d.Set("subscription_id", strconv.Itoa(*existingRegions.SubscriptionId)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("region", buildResourceDataFromAPIRegions(existingRegions.Regions, d.Get("region").(*schema.Set))); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// Does nothing
func resourceRedisCloudActiveActiveRegionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceRedisCloudActiveActiveRegionRead(ctx, d, meta)
}

func regionsCreate(ctx context.Context, subId int, regionsToCreate []*regions.Region, api *apiClient) error {
	// If no new regions were defined return
	if len(regionsToCreate) == 0 {
		return nil
	}

	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	// Call GO API createRegion for all non-existing regions
	for _, currentRegion := range regionsToCreate {
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
	}

	return nil
}

func regionsUpdateDatabases(ctx context.Context, subId int, api *apiClient, regionsToUpdateDatabases []*regions.Region, existingRegionMap map[string]*regions.Region) error {
	databaseUpdates := make(map[int][]*databases.LocalRegionProperties)
	for _, desiredRegion := range regionsToUpdateDatabases {
		// Collect existing databases to a map <dbId, db>
		existingDBMap := make(map[int]*regions.Database)
		for _, db := range existingRegionMap[*desiredRegion.Region].Databases {
			existingDBMap[*db.DatabaseId] = db
		}

		for _, db := range desiredRegion.Databases {
			if !reflect.DeepEqual(db, existingDBMap[*db.DatabaseId]) {
				localThroughput := databases.LocalThroughput{
					Region:                   desiredRegion.Region,
					WriteOperationsPerSecond: db.WriteOperationsPerSecond,
					ReadOperationsPerSecond:  db.ReadOperationsPerSecond,
				}
				localRegionProperty := databases.LocalRegionProperties{
					Region:                     desiredRegion.Region,
					LocalThroughputMeasurement: &localThroughput,
				}
				databaseUpdates[*db.DatabaseId] = append(databaseUpdates[*db.DatabaseId], &localRegionProperty)
			}
		}
	}

	if len(databaseUpdates) > 0 {
		subscriptionMutex.Lock(subId)
		defer subscriptionMutex.Unlock(subId)

		for dbId, localRegionProperties := range databaseUpdates {
			dbUpdate := databases.UpdateActiveActiveDatabase{
				Regions: localRegionProperties,
			}
			err := api.client.Database.ActiveActiveUpdate(ctx, subId, dbId, dbUpdate)
			if err != nil {
				return err
			}

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
		}
	}

	return nil
}

func regionsDelete(ctx context.Context, subId int, regionsToDelete []*regions.Region, api *apiClient) error {
	subscriptionMutex.Lock(subId)
	defer subscriptionMutex.Unlock(subId)

	deleteRegions := regions.DeleteRegions{}
	for _, region := range regionsToDelete {
		deleteRegion := regions.DeleteRegion{
			Region: region.Region,
		}
		deleteRegions.Regions = append(deleteRegions.Regions, &deleteRegion)
	}

	err := api.client.Regions.DeleteWithQuery(ctx, subId, deleteRegions)
	if err != nil {
		return err
	}

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

func buildResourceDataFromAPIRegions(regionsFromAPI []*regions.Region, regionsFromConfig *schema.Set) []map[string]interface{} {
	var result []map[string]interface{}

	recreateRegions := make(map[string]bool)
	for _, element := range regionsFromConfig.List() {
		r := element.(map[string]interface{})
		recreateRegions[r["region"].(string)] = r["recreate_region"].(bool)
	}

	for _, region := range regionsFromAPI {
		var dbs []interface{}
		for _, database := range region.Databases {
			databaseMapString := map[string]interface{}{
				"database_id":                       database.DatabaseId,
				"database_name":                     database.DatabaseName,
				"local_read_operations_per_second":  database.ReadOperationsPerSecond,
				"local_write_operations_per_second": database.WriteOperationsPerSecond,
			}
			dbs = append(dbs, databaseMapString)
		}

		regionMapString := map[string]interface{}{
			"region_id":                  region.RegionId,
			"region":                     region.Region,
			"recreate_region":            recreateRegions[*region.Region],
			"networking_deployment_cidr": region.DeploymentCIDR,
			"vpc_id":                     region.VpcId,
			"database":                   dbs,
		}
		result = append(result, regionMapString)
	}

	return result
}

func buildRegionsFromResourceData(rd *schema.Set) map[string]*regions.Region {
	result := make(map[string]*regions.Region)
	for _, r := range rd.List() {
		regionMap := r.(map[string]interface{})

		dbs := make([]*regions.Database, 0)
		for _, database := range regionMap["database"].(*schema.Set).List() {
			databaseMap := database.(map[string]interface{})
			db := regions.Database{
				DatabaseId:               redis.Int(databaseMap["database_id"].(int)),
				DatabaseName:             redis.String(databaseMap["database_name"].(string)),
				ReadOperationsPerSecond:  redis.Int(databaseMap["local_read_operations_per_second"].(int)),
				WriteOperationsPerSecond: redis.Int(databaseMap["local_write_operations_per_second"].(int)),
			}
			dbs = append(dbs, &db)
		}

		region := regions.Region{
			Region:         redis.String(regionMap["region"].(string)),
			RecreateRegion: redis.Bool(regionMap["recreate_region"].(bool)),
			DeploymentCIDR: redis.String(regionMap["networking_deployment_cidr"].(string)),
			VpcId:          redis.String(regionMap["vpc_id"].(string)),
			Databases:      dbs,
		}

		result[*region.Region] = &region
	}

	return result
}

func shouldRecreateRegion(existingRegion *regions.Region, desiredRegion *regions.Region) bool {
	return *existingRegion.DeploymentCIDR != *desiredRegion.DeploymentCIDR
}

func shouldUpdateRegionDatabases(existingRegion *regions.Region, desiredRegion *regions.Region) bool {
	return !shouldRecreateRegion(existingRegion,
		desiredRegion) && !reflect.DeepEqual(existingRegion.Databases, desiredRegion.Databases)
}
