package provider

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func resourceRedisCloudActiveActivePrivateServiceConnect() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Private Service Connect to an Active-Active Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActivePrivateServiceConnectCreate,
		ReadContext:   resourceRedisCloudActiveActivePrivateServiceConnectRead,
		DeleteContext: resourceRedisCloudActiveActivePrivateServiceConnectDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of the Pro subscription to attach",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"region_id": {
				Description: "The ID of the GCP region",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"private_service_connect_service_id": {
				Description: "The ID of the Private Service Connect",
				Type:        schema.TypeInt,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudActiveActivePrivateServiceConnectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	regionId := d.Get("region_id").(int)

	pscServiceId, err := api.Client.PrivateServiceConnect.CreateActiveActiveService(ctx, subscriptionId, regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectActiveActiveId(subscriptionId, regionId, pscServiceId))

	err = waitForPrivateServiceConnectServiceToBeActive(ctx, func() (result interface{}, state string, err error) {
		return refreshPrivateServiceConnectServiceActiveActiveStatus(ctx, subscriptionId, regionId, api)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	err = utils.WaitForSubscriptionToBeActive(ctx, subscriptionId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudActiveActivePrivateServiceConnectRead(ctx, d, meta)
}

func resourceRedisCloudActiveActivePrivateServiceConnectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	resId, err := toPscServiceActiveActiveId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	pscObj, err := api.Client.PrivateServiceConnect.GetActiveActiveService(ctx, resId.subscriptionId, resId.regionId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectActiveActiveId(resId.subscriptionId, resId.regionId, resId.pscServiceId))

	err = d.Set("subscription_id", strconv.Itoa(resId.subscriptionId))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("region_id", resId.regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_service_id", redis.IntValue(pscObj.ID))
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudActiveActivePrivateServiceConnectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	regionId := d.Get("region_id").(int)

	err = api.Client.PrivateServiceConnect.DeleteActiveActiveService(ctx, subscriptionId, regionId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId("")

	err = utils.WaitForSubscriptionToBeActive(ctx, subscriptionId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func buildPrivateServiceConnectActiveActiveId(subId int, regionId int, pscServiceId int) string {
	return fmt.Sprintf("%d/%d/%d", subId, regionId, pscServiceId)
}

type privateServiceConnectServiceActiveActiveId struct {
	subscriptionId int
	regionId       int
	pscServiceId   int
}

func toPscServiceActiveActiveId(id string) (*privateServiceConnectServiceActiveActiveId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	regionId, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	pscServiceId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectServiceActiveActiveId{
		subscriptionId: subId,
		regionId:       regionId,
		pscServiceId:   pscServiceId,
	}, nil
}

func refreshPrivateServiceConnectServiceActiveActiveStatus(ctx context.Context, subscriptionId int, regionId int, api *client.ApiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service status %d/%d to be active", subscriptionId, regionId)

	pscService, err := api.Client.PrivateServiceConnect.GetActiveActiveService(ctx, subscriptionId, regionId)
	if err != nil {
		return nil, "", err
	}

	return redis.StringValue(pscService.Status), redis.StringValue(pscService.Status), nil
}
