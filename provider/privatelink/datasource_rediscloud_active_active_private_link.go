package privatelink

import (
	"context"
	"strconv"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceActiveActivePrivateLink() *schema.Resource {
	return &schema.Resource{
		Description: "The Private Link data source allows access to an available Active-Active Private Link within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourceActiveActivePrivateLinkRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of an active-active subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"region_id": {
				Description: "The RedisCloud ID of the active-active subscription region",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"principals": {
				Description: "List of principals attached to this PrivateLink",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"principal": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"principal_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"principal_alias": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"resource_configuration_id": {
				Description: "ID of the resource configuration to attach to this PrivateLink",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"resource_configuration_arn": {
				Description: "ARN of the resource configuration attached to this PrivateLink",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"share_arn": {
				Description: "ARN of the share to attach to this Private Link",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"share_name": {
				Description: "ARN of the share to attach to this PrivateLink",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"connections": {
				Description: "Connections attached to this PrivateLink",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"association_id": {
							Description: "Association ID of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"connection_id": {
							Description: "Connection ID of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"connection_type": {
							Description: "Connection type of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"owner_id": {
							Description: "Owner ID of the PrivateLink connection",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"association_date": {
							Description: "Date the connection was associated",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			"databases": {
				Description: "",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"resource_link_endpoint": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceActiveActivePrivateLinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	regionId := d.Get("region_id").(int)

	privateLink, err := api.Client.PrivateLink.GetActiveActivePrivateLink(ctx, subId, regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	privateLinkId := makeActiveActivePrivateLinkId(subId, regionId)
	d.SetId(privateLinkId)

	if err := d.Set("resource_configuration_id", privateLink.ResourceConfigurationId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("resource_configuration_arn", privateLink.ResourceConfigurationArn); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("share_arn", privateLink.ShareArn); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("share_name", privateLink.ShareName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("subscription_id", privateLink.SubscriptionId); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("region_id", privateLink.RegionId); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("connections", flattenConnections(privateLink.Connections)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("databases", flattenDatabases(privateLink.Databases)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
