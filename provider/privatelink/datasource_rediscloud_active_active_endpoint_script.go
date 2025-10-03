package privatelink

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"strconv"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceActiveActivePrivateLinkEndpointScript() *schema.Resource {
	return &schema.Resource{
		Description: "The Active Active PrivateLink Endpoint Script data source allows users to request an endpoint script for an Active Active Subscription PrivateLink",
		ReadContext: dataSourceActiveActivePrivateLinkScriptRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of an Active Active subscription",
				Type:        schema.TypeString,
				Required:    true,
			},
			"region_id": {
				Description: "The ID of an Active Active subscription region",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"endpoint_script": {
				Description: "The endpoint script for the PrivateLink",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceActiveActivePrivateLinkScriptRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	regionId := d.Get("region_id").(int)

	endpointScript, err := api.Client.PrivateLink.GetActiveActivePrivateLinkEndpointScript(ctx, subId, regionId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	err = d.Set("endpoint_script", redis.StringValue(endpointScript))
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
