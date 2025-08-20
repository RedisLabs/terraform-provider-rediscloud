package psc

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourcePrivateServiceConnect() *schema.Resource {
	return &schema.Resource{
		Description: "The Private Service Connect data source allows access to an available Private Service Connect Service within your Redis Enterprise Cloud Account.",
		ReadContext: dataSourcePrivateServiceConnectRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of a Pro subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"private_service_connect_service_id": {
				Description: "The ID of the Private Service Connect Service relative to the associated subscription",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"connection_host_name": {
				Description: "The connection host name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"service_attachment_name": {
				Description: "The service attachment name",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "The Private Service Connect status",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourcePrivateServiceConnectRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*utils.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	pscService, err := api.Client.PrivateServiceConnect.GetService(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildPrivateServiceConnectId(subId, *pscService.ID))
	if err := d.Set("private_service_connect_service_id", pscService.ID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("connection_host_name", pscService.ConnectionHostName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("service_attachment_name", pscService.ServiceAttachmentName); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", pscService.Status); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
