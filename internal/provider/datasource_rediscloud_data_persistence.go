package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

func dataSourceRedisCloudDataPersistence() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRedisCloudDataPersistenceRead,

		Schema: map[string]*schema.Schema{
			"data_persistence": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Computed: true,
							Type:     schema.TypeString,
						},
						"description": {
							Computed: true,
							Type:     schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudDataPersistenceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	dataPersistence, err := api.client.Account.ListDataPersistence(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(time.Now().UTC().String())
	d.Set("data_persistence", flattenDataPersistence(dataPersistence))

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
