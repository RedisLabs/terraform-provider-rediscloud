package private_service_connect

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepter() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages the state of Private Service Connect Endpoint to an Active-Active Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterCreate,
		ReadContext:   resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterRead,
		UpdateContext: resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterUpdate,
		DeleteContext: resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
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
				Required:    true,
				ForceNew:    true,
			},
			"private_service_connect_endpoint_id": {
				Description: "The ID of the Private Service Connect Endpoint",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"action": {
				Description:      "Accept or reject the endpoint",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{psc.EndpointActionAccept, psc.EndpointActionReject}, false)),
			},
		},
	}
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	regionId := d.Get("region_id").(int)
	pscServiceId := d.Get("private_service_connect_service_id").(int)
	endpointId := d.Get("private_service_connect_endpoint_id").(int)
	action := d.Get("action").(string)

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, subscriptionId, regionId, pscServiceId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	endpoint := FindPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return diag.FromErr(fmt.Errorf("endpoint with id %d not found", endpointId))
	}

	if endpoint.Status == nil {
		return diag.FromErr(fmt.Errorf("endpoint with id %d has no status", endpointId))
	}

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusActive && action == psc.EndpointActionAccept {
		d.SetId(buildPrivateServiceConnectActiveActiveEndpointAccepterId(subscriptionId, regionId, pscServiceId, endpointId))
		return diag.Diagnostics{}
	}

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusRejected && action == psc.EndpointActionReject {
		d.SetId(buildPrivateServiceConnectActiveActiveEndpointAccepterId(subscriptionId, regionId, pscServiceId, endpointId))
		return diag.Diagnostics{}
	}

	refreshFunc := func(targetStatus string) (result interface{}, state string, err error) {
		return refreshPrivateServiceConnectServiceEndpointActiveActiveStatus(ctx, subscriptionId, regionId, pscServiceId, endpointId, targetStatus, api)
	}

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusInitialized || redis.StringValue(endpoint.Status) == psc.EndpointStatusProcessing {
		err = utils.WaitForPrivateServiceConnectServiceEndpointToBePending(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(buildPrivateServiceConnectActiveActiveEndpointAccepterId(subscriptionId, regionId, pscServiceId, endpointId))

	err = api.Client.PrivateServiceConnect.UpdateActiveActiveEndpoint(ctx, subscriptionId, regionId, pscServiceId, endpointId, &psc.UpdatePrivateServiceConnectEndpoint{
		Action: redis.String(action),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if action == psc.EndpointActionAccept {
		err = utils.WaitForPrivateServiceConnectServiceEndpointToBeActive(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err = utils.WaitForPrivateServiceConnectServiceEndpointToBeRejected(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterRead(ctx, d, meta)
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	resId, err := ToPscEndpointActiveActiveAccepterId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, resId.SubscriptionId, resId.RegionId, resId.PscServiceId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	endpoint := FindPrivateServiceConnectEndpoints(resId.EndpointId, endpoints.Endpoints)
	if endpoint == nil {
		d.SetId("")
		return diags
	}

	d.SetId(buildPrivateServiceConnectActiveActiveEndpointAccepterId(resId.SubscriptionId, resId.RegionId, resId.PscServiceId, redis.IntValue(endpoint.ID)))

	err = d.Set("subscription_id", strconv.Itoa(resId.SubscriptionId))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("region_id", resId.RegionId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_service_id", resId.PscServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_endpoint_id", redis.IntValue(endpoint.ID))
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func buildPrivateServiceConnectActiveActiveEndpointAccepterId(subId int, regionId int, pscId int, endpointId int) string {
	return privateServiceConnectActiveActiveEndpointId{
		subscriptionId: subId,
		regionId:       regionId,
		pscServiceId:   pscId,
		endpointId:     endpointId}.String()
}

type privateServiceConnectActiveActiveEndpointAccepterId struct {
	SubscriptionId int
	RegionId       int
	PscServiceId   int
	EndpointId     int
}

func (p privateServiceConnectActiveActiveEndpointAccepterId) String() string {
	return fmt.Sprintf("%d/%d/%d/%d", p.SubscriptionId, p.RegionId, p.PscServiceId, p.EndpointId)
}

func ToPscEndpointActiveActiveAccepterId(id string) (*privateServiceConnectActiveActiveEndpointAccepterId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
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

	pscId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	endpointId, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectActiveActiveEndpointAccepterId{
		SubscriptionId: subId,
		RegionId:       regionId,
		PscServiceId:   pscId,
		EndpointId:     endpointId,
	}, nil
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterCreate(ctx, d, meta)
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepterDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	d.SetId("")
	return diags
}
