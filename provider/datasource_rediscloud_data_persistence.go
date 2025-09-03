package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRedisCloudDataPersistence() *schema.Resource {
	return &schema.Resource{
		Description: "The data persistence data source allows access to a list of supported data persistence options.  Each option represents the rate at which a database will persist its data to storage.",
		ReadContext: dataSourceRedisCloudDataPersistenceRead,

		Schema: map[string]*schema.Schema{
			"data_persistence": {
				Type:        schema.TypeSet,
				Description: "A list of data persistence option that can be applied to subscription databases",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Computed:    true,
							Description: "The identifier of the data persistence option",
							Type:        schema.TypeString,
						},
						"description": {
							Computed:    true,
							Description: "A meaningful description of the data persistence option",
							Type:        schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudDataPersistenceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	dataPersistence, err := api.Client.Account.ListDataPersistence(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("ALL")
	if err := d.Set("data_persistence", flattenDataPersistence(dataPersistence)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func flattenDataPersistence(dataPersistenceList []*account.DataPersistence) []map[string]interface{} {

	var dpl []map[string]interface{}
	for _, currentDataPersistence := range dataPersistenceList {

		dpMapString := map[string]interface{}{
			"name":        currentDataPersistence.Name,
			"description": currentDataPersistence.Description,
		}

		dpl = append(dpl, dpMapString)
	}

	return dpl
}
