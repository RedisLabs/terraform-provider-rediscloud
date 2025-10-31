package privatelink

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"strconv"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourcePrivateLinkEndpointScript() *schema.Resource {
	return &schema.Resource{
		Description: "The PrivateLink Endpoint Script data source allows users to request an endpoint script for a pro subscription PrivateLink",
		ReadContext: dataSourcePrivateLinkScriptRead,

		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Description: "The ID of a Pro subscription",
				Type:        schema.TypeString,
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

func dataSourcePrivateLinkScriptRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	subId, err := strconv.Atoi(d.Get("subscription_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	// check if sub and privatelink exist first
	subscription, err := api.Client.Subscription.Get(ctx, subId)

	if err != nil || subscription == nil {
		return diag.FromErr(err)
	}

	privateLink, err := api.Client.PrivateLink.GetPrivateLink(ctx, subId)

	if err != nil || privateLink == nil {
		return diag.FromErr(err)
	}

	endpointScript, err := api.Client.PrivateLink.GetPrivateLinkEndpointScript(ctx, subId)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(subId))

	err = d.Set("endpoint_script", redis.StringValue(endpointScript.ResourceEndpointScript))
	if err != nil {
		return diag.FromErr(err)
	}

	return diags
}
