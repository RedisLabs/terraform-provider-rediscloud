package private_service_connect

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
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceRedisCloudPrivateServiceConnectEndpointAccepter() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages the state of Private Service Connect Endpoint to a Pro Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudPrivateServiceConnectEndpointAccepterCreate,
		ReadContext:   resourceRedisCloudPrivateServiceConnectEndpointAccepterRead,
		UpdateContext: resourceRedisCloudPrivateServiceConnectEndpointAccepterUpdate,
		DeleteContext: resourceRedisCloudPrivateServiceConnectEndpointAccepterDelete,

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

func resourceRedisCloudPrivateServiceConnectEndpointAccepterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	pscServiceId := d.Get("private_service_connect_service_id").(int)
	endpointId := d.Get("private_service_connect_endpoint_id").(int)
	action := d.Get("action").(string)

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, subscriptionId, pscServiceId)
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
		d.SetId(buildPrivateServiceConnectEndpointAccepterId(subscriptionId, pscServiceId, endpointId))
		return diag.Diagnostics{}
	}

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusRejected && action == psc.EndpointActionReject {
		d.SetId(buildPrivateServiceConnectEndpointAccepterId(subscriptionId, pscServiceId, endpointId))
		return diag.Diagnostics{}
	}

	refreshFunc := func(targetStatus string) (result interface{}, state string, err error) {
		return refreshPrivateServiceConnectServiceEndpointStatus(ctx, subscriptionId, pscServiceId, endpointId, targetStatus, api)
	}

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusInitialized || redis.StringValue(endpoint.Status) == psc.EndpointStatusProcessing {
		err = utils.WaitForPrivateServiceConnectServiceEndpointToBePending(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(buildPrivateServiceConnectEndpointAccepterId(subscriptionId, pscServiceId, endpointId))

	err = api.Client.PrivateServiceConnect.UpdateEndpoint(ctx, subscriptionId, pscServiceId, endpointId, &psc.UpdatePrivateServiceConnectEndpoint{
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

	return resourceRedisCloudPrivateServiceConnectEndpointAccepterRead(ctx, d, meta)
}

func resourceRedisCloudPrivateServiceConnectEndpointAccepterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	resId, err := ToPscEndpointAccepterId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, resId.SubscriptionId, resId.PscServiceId)
	if err != nil {
		var notFound *psc.NotFound
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

	d.SetId(buildPrivateServiceConnectEndpointAccepterId(resId.SubscriptionId, resId.PscServiceId, redis.IntValue(endpoint.ID)))

	err = d.Set("subscription_id", strconv.Itoa(resId.SubscriptionId))
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

func buildPrivateServiceConnectEndpointAccepterId(subId int, pscId int, endpointId int) string {
	return privateServiceConnectEndpointId{
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId}.String()
}

type privateServiceConnectEndpointAccepterId struct {
	SubscriptionId int
	PscServiceId   int
	EndpointId     int
}

func (p privateServiceConnectEndpointAccepterId) String() string {
	return fmt.Sprintf("%d/%d/%d", p.SubscriptionId, p.PscServiceId, p.EndpointId)
}

func ToPscEndpointAccepterId(id string) (*privateServiceConnectEndpointAccepterId, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id: %s", id)
	}

	subId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	pscId, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}

	endpointId, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, err
	}

	return &privateServiceConnectEndpointAccepterId{
		SubscriptionId: subId,
		PscServiceId:   pscId,
		EndpointId:     endpointId,
	}, nil
}

func resourceRedisCloudPrivateServiceConnectEndpointAccepterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceRedisCloudPrivateServiceConnectEndpointAccepterCreate(ctx, d, meta)
}

func resourceRedisCloudPrivateServiceConnectEndpointAccepterDelete(_ context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	d.SetId("")
	return diags
}

func refreshPrivateServiceConnectServiceEndpointStatus(ctx context.Context, subscriptionId int,
	pscServiceId int, endpointId int, targetStatus string, api *utils.ApiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service endpoint status %d/%d/%d to be %s",
		subscriptionId, pscServiceId, endpointId, targetStatus)

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, subscriptionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := FindPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return nil, "", fmt.Errorf("endpoint with id %d not found", endpointId)
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}
