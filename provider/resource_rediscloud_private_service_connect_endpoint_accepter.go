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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRedisCloudPrivateServiceConnectEndpointAccepter() *schema.Resource {
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
	api := meta.(*apiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	subscriptionMutex.Lock(subscriptionId)
	defer subscriptionMutex.Unlock(subscriptionId)

	pscServiceId := d.Get("private_service_connect_service_id").(int)
	endpointId := d.Get("private_service_connect_endpoint_id").(int)
	action := d.Get("action").(string)

	endpoints, err := api.client.PrivateServiceConnect.GetEndpoints(ctx, subscriptionId, pscServiceId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	endpoint := findPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
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
		err = waitForPrivateServiceConnectServiceEndpointToBePending(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(buildPrivateServiceConnectEndpointAccepterId(subscriptionId, pscServiceId, endpointId))

	err = api.client.PrivateServiceConnect.UpdateEndpoint(ctx, subscriptionId, pscServiceId, endpointId, &psc.UpdatePrivateServiceConnectEndpoint{
		Action: redis.String(action),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if action == psc.EndpointActionAccept {
		err = waitForPrivateServiceConnectServiceEndpointToBeActive(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	} else {
		err = waitForPrivateServiceConnectServiceEndpointToBeRejected(ctx, refreshFunc)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRedisCloudPrivateServiceConnectEndpointAccepterRead(ctx, d, meta)
}

func resourceRedisCloudPrivateServiceConnectEndpointAccepterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	resId, err := toPscEndpointAccepterId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	endpoints, err := api.client.PrivateServiceConnect.GetEndpoints(ctx, resId.subscriptionId, resId.pscServiceId)
	if err != nil {
		var notFound *psc.NotFound
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	endpoint := findPrivateServiceConnectEndpoints(resId.endpointId, endpoints.Endpoints)
	if endpoint == nil {
		d.SetId("")
		return diags
	}

	d.SetId(buildPrivateServiceConnectEndpointAccepterId(resId.subscriptionId, resId.pscServiceId, redis.IntValue(endpoint.ID)))

	err = d.Set("subscription_id", strconv.Itoa(resId.subscriptionId))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_service_id", resId.pscServiceId)
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
	subscriptionId int
	pscServiceId   int
	endpointId     int
}

func (p privateServiceConnectEndpointAccepterId) String() string {
	return fmt.Sprintf("%d/%d/%d", p.subscriptionId, p.pscServiceId, p.endpointId)
}

func toPscEndpointAccepterId(id string) (*privateServiceConnectEndpointAccepterId, error) {
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
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId,
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
	pscServiceId int, endpointId int, targetStatus string, api *apiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service endpoint status %d/%d/%d to be %s",
		subscriptionId, pscServiceId, endpointId, targetStatus)

	endpoints, err := api.client.PrivateServiceConnect.GetEndpoints(ctx, subscriptionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := findPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return nil, "", fmt.Errorf("endpoint with id %d not found", endpointId)
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}

func waitForPrivateServiceConnectServiceEndpointToBePending(ctx context.Context, refreshFunc func(targetStatus string) (result interface{}, state string, err error)) error {
	targetStatus := psc.EndpointStatusPending
	return waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx, func() (result interface{}, state string, err error) {
		return refreshFunc(targetStatus)
	}, targetStatus, []string{
		psc.EndpointStatusInitialized,
		psc.EndpointStatusProcessing})
}

func waitForPrivateServiceConnectServiceEndpointToBeActive(ctx context.Context, refreshFunc func(targetStatus string) (result interface{}, state string, err error)) error {
	targetStatus := psc.EndpointStatusActive
	return waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx, func() (result interface{}, state string, err error) {
		return refreshFunc(targetStatus)
	}, targetStatus, []string{
		psc.EndpointStatusPending,
		psc.EndpointStatusAcceptPending})
}

func waitForPrivateServiceConnectServiceEndpointToBeRejected(ctx context.Context, refreshFunc func(targetStatus string) (result interface{}, state string, err error)) error {
	targetStatus := psc.EndpointStatusRejected
	return waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx, func() (result interface{}, state string, err error) {
		return refreshFunc(targetStatus)
	}, targetStatus, []string{
		psc.EndpointStatusPending,
		psc.EndpointStatusRejectPending})
}

func waitForPrivateServiceConnectServiceEndpointToBeInStatus(ctx context.Context,
	refreshFunc func() (result interface{}, state string, err error), status string, pendingStatus []string) error {
	wait := &retry.StateChangeConf{
		Pending:      pendingStatus,
		Target:       []string{status},
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
