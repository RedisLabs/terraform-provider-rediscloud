package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strconv"
)

func dataSourceRedisCloudCloudAccount() *schema.Resource {
	return &schema.Resource{
		Description: "The Cloud Account data source allows access to the ID of a Cloud Account configuration.  This ID can be used when creating Subscription resources.",
		ReadContext: dataSourceRedisCloudCloudAccountRead,

		Schema: map[string]*schema.Schema{
			"exclude_internal_account": {
				Type:        schema.TypeBool,
				Description: "Whether to exclude the Redis Labs internal cloud account.",
				Optional:    true,
				Default:     false,
			},
			"provider_type": {
				Type:             schema.TypeString,
				Description:      "The cloud provider of the cloud account, (either `AWS` or `GCP`)",
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
			},
			"name": {
				Type:        schema.TypeString,
				Description: "A meaningful name to identify the cloud account",
				Optional:    true,
				Computed:    true,
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Description: "The access key ID associated with the cloud account",
				Computed:    true,
			},
		},
	}
}

func dataSourceRedisCloudCloudAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := meta.(*apiClient)

	accounts, err := client.client.CloudAccount.List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var filters []func(account *cloud_accounts.CloudAccount) bool

	if v, ok := d.GetOk("exclude_internal_account"); ok && v.(bool) {
		filters = append(filters, func(account *cloud_accounts.CloudAccount) bool {
			return redis.IntValue(account.ID) != 1
		})
	}
	if v, ok := d.GetOk("provider_type"); ok {
		filters = append(filters, func(account *cloud_accounts.CloudAccount) bool {
			return redis.StringValue(account.Provider) == v.(string)
		})
	}
	if v, ok := d.GetOk("name"); ok {
		filters = append(filters, func(account *cloud_accounts.CloudAccount) bool {
			return redis.StringValue(account.Name) == v.(string)
		})
	}

	accounts = filterCloudAccounts(accounts, filters)

	if len(accounts) == 0 {
		return diag.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(accounts) > 1 {
		return diag.Errorf("Your query returned more than one result. Please change try a more specific search criteria and try again.")
	}

	account := accounts[0]

	d.SetId(strconv.Itoa(redis.IntValue(account.ID)))
	if err := d.Set("name", redis.StringValue(account.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("provider_type", redis.StringValue(account.Provider)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("access_key_id", redis.StringValue(account.AccessKeyID)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func filterCloudAccounts(accounts []*cloud_accounts.CloudAccount, filters []func(account *cloud_accounts.CloudAccount) bool) []*cloud_accounts.CloudAccount {
	var filtered []*cloud_accounts.CloudAccount
	for _, cloudAccount := range accounts {
		if filterCloudAccount(cloudAccount, filters) {
			filtered = append(filtered, cloudAccount)
		}
	}

	return filtered
}

func filterCloudAccount(account *cloud_accounts.CloudAccount, filters []func(account *cloud_accounts.CloudAccount) bool) bool {
	for _, f := range filters {
		if !f(account) {
			return false
		}
	}
	return true
}
