package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
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

func resourceRedisCloudPrivateServiceConnectEndpoint() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Private Service Connect Endpoint to a Pro Subscription in your Redis Enterprise Cloud Account.",
		CreateContext: resourceRedisCloudPrivateServiceConnectEndpointCreate,
		ReadContext:   resourceRedisCloudPrivateServiceConnectEndpointRead,
		DeleteContext: resourceRedisCloudPrivateServiceConnectEndpointDelete,

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

func resourceRedisCloudPrivateServiceConnectEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	api := meta.(*client.ApiClient)

	subscriptionId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(subscriptionId)
	defer utils.SubscriptionMutex.Unlock(subscriptionId)

	pscServiceId := d.Get("private_service_connect_service_id").(int)
	gcpProjectId := d.Get("gcp_project_id").(string)
	gcpVpcName := d.Get("gcp_vpc_name").(string)
	gcpVpcSubnetName := d.Get("gcp_vpc_subnet_name").(string)
	endpointConnectionNamePrefix := d.Get("endpoint_connection_name").(string)

	endpointId, err := api.Client.PrivateServiceConnect.CreateEndpoint(ctx, subscriptionId, pscServiceId, psc.CreatePrivateServiceConnectEndpoint{
		GCPProjectID:           &gcpProjectId,
		GCPVPCName:             &gcpVpcName,
		GCPVPCSubnetName:       &gcpVpcSubnetName,
		EndpointConnectionName: &endpointConnectionNamePrefix,
	})

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectEndpointId(subscriptionId, pscServiceId, endpointId))

	err = utils.WaitForSubscriptionToBeActive(ctx, subscriptionId, api)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudPrivateServiceConnectEndpointRead(ctx, d, meta)
}

func resourceRedisCloudPrivateServiceConnectEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	resId, err := toPscEndpointId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, resId.subscriptionId, resId.pscServiceId)
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

	d.SetId(buildPrivateServiceConnectEndpointId(resId.subscriptionId, resId.pscServiceId, *endpoint.ID))

	if redis.StringValue(endpoint.Status) != psc.EndpointStatusRejected && redis.StringValue(endpoint.Status) != psc.EndpointStatusDeleted {
		creationScript, err := api.Client.PrivateServiceConnect.GetEndpointCreationScripts(ctx,
			resId.subscriptionId, resId.pscServiceId, *endpoint.ID, true)
		if err != nil {
			var notFound *psc.NotFound
			if errors.As(err, &notFound) {
				d.SetId("")
				return diags
			}
			return diag.FromErr(err)
		}

		if err := d.Set("service_attachments", flattenPrivateServiceConnectEndpointServiceAttachments(creationScript.Script.TerraformGcp.ServiceAttachments)); err != nil {
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

func resourceRedisCloudPrivateServiceConnectEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	resId, err := toPscEndpointId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	utils.SubscriptionMutex.Lock(resId.subscriptionId)
	defer utils.SubscriptionMutex.Unlock(resId.subscriptionId)

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, resId.subscriptionId, resId.pscServiceId)
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

	if redis.StringValue(endpoint.Status) == psc.EndpointStatusInitialized {
		// It's only possible to delete an endpoint in initialized status
		err = api.Client.PrivateServiceConnect.DeleteEndpoint(ctx, resId.subscriptionId, resId.pscServiceId, resId.endpointId)
		if err != nil {
			return diag.FromErr(err)
		}
		return diags
	}

	// Endpoints will be automatically removed once related GCP resources are removed. So we will wait for this
	// to happen, but we can't check the GCP resources from this provider
	err = waitForPrivateServiceConnectServiceEndpointDisappear(ctx, func() (result interface{}, state string, err error) {
		return refreshPrivateServiceConnectServiceEndpointDisappear(ctx, resId.subscriptionId, resId.pscServiceId, resId.endpointId, api)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}

func buildPrivateServiceConnectEndpointId(subId int, pscId int, endpointId int) string {
	return privateServiceConnectEndpointId{
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId}.String()
}

type privateServiceConnectEndpointId struct {
	subscriptionId int
	pscServiceId   int
	endpointId     int
}

func (p privateServiceConnectEndpointId) String() string {
	return fmt.Sprintf("%d/%d/%d", p.subscriptionId, p.pscServiceId, p.endpointId)
}

func toPscEndpointId(id string) (*privateServiceConnectEndpointId, error) {
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

	return &privateServiceConnectEndpointId{
		subscriptionId: subId,
		pscServiceId:   pscId,
		endpointId:     endpointId,
	}, nil
}

func refreshPrivateServiceConnectServiceEndpointDisappear(ctx context.Context, subscriptionId int,
	pscServiceId int, endpointId int, api *client.ApiClient) (result interface{}, state string, err error) {

	log.Printf("[DEBUG] Waiting for private service connect service endpoint %d/%d/%d to be deleted",
		subscriptionId, pscServiceId, endpointId)

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, subscriptionId, pscServiceId)
	if err != nil {
		return nil, "", err
	}

	endpoint := findPrivateServiceConnectEndpoints(endpointId, endpoints.Endpoints)
	if endpoint == nil {
		return placeholderStatusDisappear, placeholderStatusDisappear, nil
	}

	return redis.StringValue(endpoint.Status), redis.StringValue(endpoint.Status), nil
}

func findPrivateServiceConnectEndpoints(id int, endpoints []*psc.PrivateServiceConnectEndpoint) *psc.PrivateServiceConnectEndpoint {
	for _, endpoint := range endpoints {
		if redis.IntValue(endpoint.ID) == id {
			return endpoint
		}
	}
	return nil
}

func flattenPrivateServiceConnectEndpointServiceAttachments(serviceAttachments []psc.TerraformGCPServiceAttachment) []map[string]interface{} {
	var rl []map[string]interface{}
	for _, serviceAttachment := range serviceAttachments {

		serviceAttachmentMapString := map[string]interface{}{
			"name":                 serviceAttachment.Name,
			"dns_record":           serviceAttachment.DNSRecord,
			"ip_address_name":      serviceAttachment.IPAddressName,
			"forwarding_rule_name": serviceAttachment.ForwardingRuleName,
		}

		rl = append(rl, serviceAttachmentMapString)
	}

	return rl
}

func waitForPrivateServiceConnectServiceEndpointDisappear(ctx context.Context, refreshFunc func() (result interface{}, state string, err error)) error {
	wait := &retry.StateChangeConf{
		Pending: []string{
			psc.EndpointStatusProcessing,
			psc.EndpointStatusPending,
			psc.EndpointStatusAcceptPending,
			psc.EndpointStatusActive,
			psc.EndpointStatusDeleted,
			psc.EndpointStatusRejected,
			psc.EndpointStatusRejectPending,
			psc.EndpointStatusFailed,
		},
		Target:       []string{placeholderStatusDisappear},
		Timeout:      utils.SafetyTimeout,
		Delay:        10 * time.Second,
		PollInterval: 30 * time.Second,

		Refresh: refreshFunc,
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}
