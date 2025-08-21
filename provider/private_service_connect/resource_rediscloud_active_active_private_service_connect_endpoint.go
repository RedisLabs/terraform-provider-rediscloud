package private_service_connect

import (
	"context"
	"errors"
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceRedisCloudActiveActivePrivateServiceConnectEndpoint() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Private Service Connect Endpoint to an  Active-Active Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudActiveActivePrivateServiceConnectEndpointCreate,
		ReadContext:   resourceRedisCloudActiveActivePrivateServiceConnectEndpointRead,
		DeleteContext: resourceRedisCloudActiveActivePrivateServiceConnectEndpointDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
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
				Required:    true,
				ForceNew:    true,
			},
			"private_service_connect_endpoint_id": {
				Description: "The ID of the Private Service Connect Endpoint",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"gcp_project_id": {
				Description: "The Google Cloud Project ID",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"gcp_vpc_name": {
				Description: "The GCP VPC Network name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"gcp_vpc_subnet_name": {
				Description: "The GCP Subnet name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"endpoint_connection_name": {
				Description: "The endpoint connection name prefix. This prefix that will be used to create the Private Service Connect endpoint in your Google Cloud account",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"service_attachments": {
				Description: "The service attachments that were created for the Private Service Connect endpoint",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "Name of the service attachment",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"dns_record": {
							Description: "DNS record for the service attachment",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"ip_address_name": {
							Description: "IP address name for the service attachment",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"forwarding_rule_name": {
							Description: "Name of the forwarding rule for the service attachment",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*utils.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	regionId := d.Get("region_id").(int)
	pscServiceId := d.Get("private_service_connect_service_id").(int)
	gcpProjectId := d.Get("gcp_project_id").(string)
	gcpVpcName := d.Get("gcp_vpc_name").(string)
	gcpVpcSubnetName := d.Get("gcp_vpc_subnet_name").(string)
	endpointConnectionNamePrefix := d.Get("endpoint_connection_name").(string)

	endpointId, err := api.Client.PrivateServiceConnect.CreateActiveActiveEndpoint(ctx, subscriptionId, regionId, pscServiceId, psc.CreatePrivateServiceConnectEndpoint{
		GCPProjectID:           &gcpProjectId,
		GCPVPCName:             &gcpVpcName,
		GCPVPCSubnetName:       &gcpVpcSubnetName,
		EndpointConnectionName: &endpointConnectionNamePrefix,
	})

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectActiveActiveEndpointId(subscriptionId, regionId, pscServiceId, endpointId))

	err = utils.WaitForSubscriptionToBeActive(ctx, subscriptionId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudActiveActivePrivateServiceConnectEndpointRead(ctx, d, meta)
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	resId, err := toPscEndpointActiveActiveId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, resId.subscriptionId, resId.regionId, resId.pscServiceId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	endpoint := FindPrivateServiceConnectEndpoints(resId.endpointId, endpoints.Endpoints)
	if endpoint == nil {
		d.SetId("")
		return diags
	}

	d.SetId(buildPrivateServiceConnectActiveActiveEndpointId(resId.subscriptionId, resId.regionId, resId.pscServiceId, redis.IntValue(endpoint.ID)))

	if redis.StringValue(endpoint.Status) != psc.EndpointStatusRejected && redis.StringValue(endpoint.Status) != psc.EndpointStatusDeleted {
		creationScript, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpointCreationScripts(ctx,
			resId.subscriptionId, resId.regionId, resId.pscServiceId, redis.IntValue(endpoint.ID), true)
		if err != nil {
			var notFound *psc.NotFoundActiveActive
			if errors.As(err, &notFound) {
				d.SetId("")
				return diags
			}
			return diag.FromErr(err)
		}

		if err := d.Set("service_attachments", utils.FlattenPrivateServiceConnectEndpointServiceAttachments(creationScript.Script.TerraformGcp.ServiceAttachments)); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("service_attachments", []any{}); err != nil {
			return diag.FromErr(err)
		}
	}

	err = d.Set("subscription_id", strconv.Itoa(resId.subscriptionId))
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("region_id", resId.regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_service_id", resId.pscServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("private_service_connect_endpoint_id", endpoint.ID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("gcp_project_id", endpoint.GCPProjectID)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("gcp_vpc_name", endpoint.GCPVPCName)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("gcp_vpc_subnet_name", endpoint.GCPVPCSubnetName)
	if err != nil {
		return diag.FromErr(err)
	}

	err = d.Set("endpoint_connection_name", endpoint.EndpointConnectionName)
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudActiveActivePrivateServiceConnectEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	resId, err := toPscEndpointActiveActiveId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(resId.subscriptionId)
	defer utils.SubscriptionMutex.Unlock(resId.subscriptionId)

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, resId.subscriptionId, resId.regionId, resId.pscServiceId)
	if err != nil {
		var notFound *psc.NotFoundActiveActive
		if errors.As(err, &notFound) {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	endpoint := FindPrivateServiceConnectEndpoints(resId.endpointId, endpoints.Endpoints)
	if endpoint == nil {
		d.SetId("")
		return diags
	}

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusInitialized {
		// It's only possible to delete an endpoint in initialized status
		err = api.Client.PrivateServiceConnect.DeleteActiveActiveEndpoint(ctx, resId.subscriptionId, resId.regionId, resId.pscServiceId, resId.endpointId)
		if err != nil {
			return diag.FromErr(err)
		}
		return diags
	}

	// Endpoints will be automatically removed once related GCP resources are removed. So we will wait for this
	// to happen, but we can't check the GCP resources from this provider
	err = utils.WaitForPrivateServiceConnectServiceEndpointDisappear(ctx, func() (result interface{}, state string, err error) {
		return refreshPrivateServiceConnectServiceActiveActiveEndpointDisappear(ctx, resId.subscriptionId, resId.regionId, resId.pscServiceId, resId.endpointId, api)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func buildPrivateServiceConnectActiveActiveEndpointId(subId int, regionId int, pscId int, endpointId int) string {
	return privateServiceConnectActiveActiveEndpointId{
		subscriptionId: subId,
		regionId:       regionId,
		pscServiceId:   pscId,
		endpointId:     endpointId}.String()
}

type privateServiceConnectActiveActiveEndpointId struct {
	subscriptionId int
	regionId       int
	pscServiceId   int
	endpointId     int
}

func (p privateServiceConnectActiveActiveEndpointId) String() string {
	return fmt.Sprintf("%d/%d/%d/%d", p.subscriptionId, p.regionId, p.pscServiceId, p.endpointId)
}

func toPscEndpointActiveActiveId(id string) (*privateServiceConnectActiveActiveEndpointId, error) {
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

	return &privateServiceConnectActiveActiveEndpointId{
		subscriptionId: subId,
		regionId:       regionId,
		pscServiceId:   pscId,
		endpointId:     endpointId,
	}, nil
}

func refreshPrivateServiceConnectServiceActiveActiveEndpointDisappear(ctx context.Context, subscriptionId int,
	regionId int, pscServiceId int, endpointId int, api *utils.ApiClient) (result interface{}, state string, err error) {
	log.Printf("[DEBUG] Waiting for private service connect service endpoint %d/%d/%d to be deleted",
		subscriptionId, pscServiceId, endpointId)

	endpoints, err := api.Client.PrivateServiceConnect.GetActiveActiveEndpoints(ctx, subscriptionId, regionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := FindPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return utils.PlaceholderStatusDisappear, utils.PlaceholderStatusDisappear, nil
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}
