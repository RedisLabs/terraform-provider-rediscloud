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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRedisCloudPrivateServiceConnect() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Private Service Connect to an Active-Active Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudPrivateServiceConnectCreate,
		ReadContext:   resourceRedisCloudPrivateServiceConnectRead,
		DeleteContext: resourceRedisCloudPrivateServiceConnectDelete,

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
			"private_service_connect_service_id": {
				Description: "The ID of the Private Service Connect",
				Type:        schema.TypeInt,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudPrivateServiceConnectCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*apiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	subscriptionMutex.Lock(subscriptionId)

	pscServiceId, err := api.client.PrivateServiceConnect.CreateService(ctx, subscriptionId)
	if err != nil {
		subscriptionMutex.Unlock(subscriptionId)
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectId(subscriptionId, pscServiceId))

	err = waitForPrivateServiceConnectServiceToBeActive(ctx, func() (result interface{}, state string, err error) {
		return refreshPrivateServiceConnectServiceStatus(ctx, subscriptionId, api)
	})
	if err != nil {
		subscriptionMutex.Unlock(subscriptionId)
		return diag.FromErr(err)
	}

	err = waitForSubscriptionToBeActive(ctx, subscriptionId, api)
	if err != nil {
		subscriptionMutex.Unlock(subscriptionId)
		return diag.FromErr(err)
	}

	subscriptionMutex.Unlock(subscriptionId)
	return resourceRedisCloudPrivateServiceConnectRead(ctx, d, meta)
}

func resourceRedisCloudPrivateServiceConnectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	resId, err := toPscServiceId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	pscObj, err := api.client.PrivateServiceConnect.GetService(ctx, resId.subscriptionId)
	if err != nil {
		var notFound *psc.NotFound
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectId(resId.subscriptionId, resId.pscServiceId))

	err = d.Set("subscription_id", strconv.Itoa(resId.subscriptionId))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_service_id", redis.IntValue(pscObj.ID))
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudPrivateServiceConnectDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	subscriptionMutex.Lock(subscriptionId)
	defer subscriptionMutex.Unlock(subscriptionId)

	err = api.client.PrivateServiceConnect.DeleteService(ctx, subscriptionId)
	if err != nil {
		var notFound *psc.NotFound
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.SetId("")

	err = waitForSubscriptionToBeActive(ctx, subscriptionId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func buildPrivateServiceConnectId(subId int, pscServiceId int) string {
	return fmt.Sprintf("%d/%d", subId, pscServiceId)
}

type privateServiceConnectServiceId struct {
	subscriptionId int
	pscServiceId   int
}

func toPscServiceId(id string) (*privateServiceConnectServiceId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	pscServiceId, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectServiceId{
		subscriptionId: subId,
		pscServiceId:   pscServiceId,
	}, nil
}

func refreshPrivateServiceConnectServiceStatus(ctx context.Context, subscriptionId int, api *apiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service status %d to be active", subscriptionId)

	pscService, err := api.client.PrivateServiceConnect.GetService(ctx, subscriptionId)
	if err != nil {
		return nil, "", err
	}

	return redis.StringValue(pscService.Status), redis.StringValue(pscService.Status), nil
}

func waitForPrivateServiceConnectServiceToBeActive(ctx context.Context, refreshFunc func() (result interface{}, state string, err error)) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			psc.ServiceStatusCreateQueued,
			psc.ServiceStatusInitialized,
			psc.ServiceStatusCreatePending},
		Target:       []string{psc.ServiceStatusActive},
		Timeout:      safetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: refreshFunc,
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
