package provider

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/regions"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudActiveActiveRegion() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates an Active Active Region and within your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActiveSubscriptionCreate,
		ReadContext:   resourceRedisCloudActiveActiveRegionRead,
		//UpdateContext: resourceRedisCloudSubscriptionUpdate,
		//DeleteContext: resourceRedisCloudSubscriptionDelete,
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, i interface{}) error {
			_, cPlanExists := diff.GetOk("creation_plan")
			if cPlanExists {
				return nil
			}

			// The resource hasn't been created yet, but the creation plan is missing.
			if diff.Id() == "" {
				return fmt.Errorf(`the "creation_plan" block is required`)
			}
			return nil
		},

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
				Description:      "A meaningful name to identify the subscription",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
				ForceNew:         true,
			},
			"delete_regions": {
				Description: "TODO",
				Type:        schema.TypeBool,
				Optional:    true,
			},
			"regions": {
				Description: "Cloud networking details, per region (multiple regions for Active-Active cluster)",
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["region"].(string)))
					return schema.HashString(buf.String())
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region_id": {
							Description:      "The region id",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
							ForceNew:         true,
						},
						"region": {
							Description: "Deployment region as defined by cloud provider",
							Type:        schema.TypeString,
							Required:    true,
						},
						"deployment_cidr": {
							Description:      "Deployment CIDR mask",
							Type:             schema.TypeString,
							ForceNew:         true,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
						},
						"vpc_id": {
							Description: "Identifier of the VPC to be peered",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"recreate_region": {
							Description: "Defines wheter the regions should be re-created",
							Type:        schema.TypeBool,
							Required:    false,
						},
						"networking_deployment_cidr": {
							Description:      "Deployment CIDR mask",
							Type:             schema.TypeString,
							Required:         true,
							ValidateDiagFunc: validateDiagFunc(validation.IsCIDR),
						},

						"databases": {
							Description: "TODO",
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Set: func(v interface{}) int {
								var buf bytes.Buffer
								m := v.(map[string]interface{})
								buf.WriteString(fmt.Sprintf("%s-", m["region"].(string)))
								return schema.HashString(buf.String())
							},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Description:      "A numeric id for the database",
										Type:             schema.TypeString,
										Required:         true,
										ValidateDiagFunc: validateDiagFunc(validation.StringMatch(regexp.MustCompile("^\\d+$"), "must be a number")),
										ForceNew:         true,
									},
									"database_name": {
										Description: "A meaningful name to identify the database",
										Type:        schema.TypeString,
										Required:    true,
									},
									"write_operations_per_second": {
										Description: "Write operations per second for creation plan databases",
										Type:        schema.TypeInt,
										Required:    true,
									},
									"read_operations_per_second": {
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

func resourceRedisCloudActiveActiveSubscriptionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Query API for existing Regions for a given Subscription
	existingRegions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create an existingRegionMap<regionId, region>
	existingRegionMap := make(map[int]*regions.Region)
	for _, existingRegion := range existingRegions.Regions {
		existingRegionMap[*existingRegion.RegionId] = existingRegion
	}

	createRegionsFromResourceData := buildCreateActiveActiveRegions(d.Get("regions"))
	// Filter non-existing regions
	createRegions := make([]*regions.Region, 0)
	for _, currentRegion := range createRegionsFromResourceData {
		if nonExistingRegion, ok := existingRegionMap[*currentRegion.RegionId]; ok {
			createRegions = append(createRegions, nonExistingRegion)
		}
	}

	// Call GO API createRegion for all non-existing regions
	for _, currentRegion := range createRegions {
		api.client.Regions.Create(ctx, currentRegion)
	}

	return diags
}

func resourceRedisCloudActiveActiveRegionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	var diags diag.Diagnostics

	subId, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	regions, err := api.client.Regions.List(ctx, subId)
	if err != nil {
		if _, ok := err.(*subscriptions.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("subscription_id", redis.IntValue(regions.SubscriptionId)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("regions", buildActiveActiveRegionsResourceData(regions.Regions, true)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func buildActiveActiveRegionsResourceData(regions []*regions.Region, isResource bool) []map[string]interface{} {
	var resourceDataMap []map[string]interface{}

	for _, currentRegion := range regions {
		var databases []interface{}
		for _, database := range currentRegion.Databases {
			datbaseMapString := map[string]interface{}{
				"database_id":                 database.DatabaseId,
				"database_name":               database.DatabaseName,
				"read_operations_per_second":  database.ReadOperationsPerSecond,
				"write_operations_per_second": database.WriteOperationsPerSecond,
			}
			databases = append(databases, datbaseMapString)
		}

		regionMapString := map[string]interface{}{
			"region_id":       currentRegion.RegionId,
			"region":          currentRegion.Region,
			"deployment_cidr": currentRegion.DeploymentCIDR,
			"vpc_id":          currentRegion.VpcId,
			"databases":       databases,
		}
		resourceDataMap = append(resourceDataMap, regionMapString)
	}

	return resourceDataMap
}

func buildCreateActiveActiveRegions(r interface{}) []*regions.Region {
	createRegions := make([]*regions.Region, 0)
	for _, region := range r.([]interface{}) {
		regionMap := region.(map[string]interface{})

		// CreateDatabases
		createDatabases := make([]*regions.Database, 0)
		if databases := regionMap["databases"].(*schema.Set).List(); len(databases) != 0 {
			for _, database := range databases {
				databaseMap := database.(map[string]interface{})
				createDatabase := regions.Database{
					DatabaseId:               redis.Int(databaseMap["id"].(int)),
					DatabaseName:             redis.String(databaseMap["database_name"].(string)),
					ReadOperationsPerSecond:  redis.Int(databaseMap["read_operations_per_second"].(int)),
					WriteOperationsPerSecond: redis.Int(databaseMap["write_operations_per_second"].(int)),
				}
				createDatabases = append(createDatabases, &createDatabase)
			}
		}

		createRegion := regions.Region{
			RegionId:       redis.Int(regionMap["region_id"].(int)),
			Region:         redis.String(regionMap["region"].(string)),
			DeploymentCIDR: redis.String(regionMap["deployment_cidr"].(string)),
			VpcId:          redis.String(regionMap["vpc_id"].(string)),
			Databases:      createDatabases,
		}

		createRegions = append(createRegions, &createRegion)
	}

	return createRegions
}
