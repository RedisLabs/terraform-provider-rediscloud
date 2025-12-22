package provider

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/service/account"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func dataSourceRedisCloudDatabaseModules() *schema.Resource {
	return &schema.Resource{
		Description: "The Database modules data source allows access to the details of existing database modules within your Redis Enterprise Cloud account.",
		ReadContext: dataSourceRedisCloudDatabaseModulesRead,

		Schema: map[string]*schema.Schema{
			"modules": {
				Description: "A list of database modules",
				Type:        schema.TypeSet,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The identifier assigned by the database module",
							Computed:    true,
							Type:        schema.TypeString,
						},
						"description": {
							Description: "A meaningful description of the database module",
							Computed:    true,
							Type:        schema.TypeString,
						},
					},
				},
			},
		},
	}
}

func dataSourceRedisCloudDatabaseModulesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	api := meta.(*client.ApiClient)

	modules, err := api.Client.Account.ListDatabaseModules(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("ALL")
	if err := d.Set("modules", flattenDatabaseModules(modules)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func flattenDatabaseModules(moduleList []*account.DatabaseModule) []map[string]interface{} {

	var dbml []map[string]interface{}
	for _, currentModule := range moduleList {

		moduleMapString := map[string]interface{}{
			"name":        currentModule.Name,
			"description": currentModule.Description,
		}

		dbml = append(dbml, moduleMapString)
	}

	return dbml
}
