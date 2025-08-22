package private_service_connect

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourcePrivateServiceConnectEndpoints() *schema.Resource {
	return &schema.Resource{
		Description: "The Private Service Connect Endpoints data source allows access to an available endpoints on a Private Service Connect Service within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourcePrivateServiceConnectEndpointsRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of a Pro subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"private_service_connect_service_id": {
				Description: "The ID of the Private Service Connect Service relative to the associated subscription",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"endpoints": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"private_service_connect_endpoint_id": {
							Description: "The ID of the Private Service Connect endpoint",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"gcp_project_id": {
							Description: "The Google Cloud Project ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_vpc_name": {
							Description: "The GCP VPC name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"gcp_vpc_subnet_name": {
							Description: "The GCP Subnet name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"endpoint_connection_name": {
							Description: "The endpoint connection name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status": {
							Description: "The endpoint status",
							Type:        schema.TypeString,
							Computed:    true,
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
				},
			},
		},
	}
}

func dataSourcePrivateServiceConnectEndpointsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	pscServiceId := d.Get("private_service_connect_service_id").(int)

	endpoints, err := api.Client.PrivateServiceConnect.GetEndpoints(ctx, subId, pscServiceId)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceAttachments := map[int][]psc.TerraformGCPServiceAttachment{}
	for _, endpoint := range endpoints.Endpoints {
		serviceAttachments[*endpoint.ID] = []psc.TerraformGCPServiceAttachment{}
		if redis.StringValue(endpoint.Status) != psc.EndpointStatusRejected && redis.StringValue(endpoint.Status) != psc.EndpointStatusDeleted {
			script, err := api.Client.PrivateServiceConnect.GetEndpointCreationScripts(ctx, subId, pscServiceId, redis.IntValue(endpoint.ID), true)
			if err != nil {
				return diag.FromErr(err)
			}
			serviceAttachments[*endpoint.ID] = script.Script.TerraformGcp.ServiceAttachments
		}
	}

	if err := d.Set("endpoints", utils.FlattenPrivateServiceConnectEndpoints(endpoints.Endpoints, serviceAttachments)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectId(subId, pscServiceId))

	return diags
}
