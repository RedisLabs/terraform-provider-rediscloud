package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"time"
)

func dataSourceRedisCloudDatabaseModules() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRedisCloudDatabaseModulesRead,

		Schema: map[string]*schema.Schema{
			"modules": {
				Type: schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Computed: true,
							Type: schema.TypeString,
						},
						"description": {
							Computed: true,
							Type: schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudDatabaseModulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*apiClient)

	modules, err := api.client.Account.ListDatabaseModules(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(time.Now().UTC().String())
	d.Set("modules", flattenDatabaseModules(modules))

	return diags
}

func flattenDatabaseModules(moduleList []*account.DatabaseModule) []map[string]interface{} {

	var dbml []map[string]interface{}
	for _, currentModule := range moduleList {

		moduleMapString := map[string]interface{}{
			"name":          currentModule.Name,
			"description": currentModule.Description,
		}

		dbml = append(dbml, moduleMapString)
	}

	return dbml
}
