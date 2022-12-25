package provider

import (
	"context"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	"strconv"
	"time"
)

func resourceRedisCloudCloudAccount() *schema.Resource {
	return &schema.Resource{
		Description:   "Creates a Cloud Account resource representing the access credentials to a cloud provider account, (`AWS` or `GCP`). Your Redis Enterprise Cloud account uses these credentials to provision databases within your infrastructure. ",
		CreateContext: resourceRedisCloudCloudAccountCreate,
		ReadContext:   resourceRedisCloudCloudAccountRead,
		UpdateContext: resourceRedisCloudCloudAccountUpdate,
		DeleteContext: resourceRedisCloudCloudAccountDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				_, err := strconv.Atoi(d.Id())
				if err != nil {
					return nil, err
				}
				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"access_key_id": {
				Description: "Cloud provider access key",
				Type:        schema.TypeString,
				Required:    true,
			},
			"access_secret_key": {
				Description: "Cloud provider secret key",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"name": {
				Description: "Display name of the account",
				Type:        schema.TypeString,
				Required:    true,
			},
			"provider_type": {
				Description:      "Cloud provider type - either `AWS` or `GCP`",
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
				ForceNew:         true,
			},
			"status": {
				Description: "The current status of the account - `draft`, `pending` or `active`",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceRedisCloudCloudAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	accessKey := d.Get("access_key_id").(string)
	secretKey := d.Get("access_secret_key").(string)
	name := d.Get("name").(string)
	provider := d.Get("provider_type").(string)

	id, err := client.client.CloudAccount.Create(ctx, cloud_accounts.CreateCloudAccount{
		AccessKeyID:     redis.String(accessKey),
		AccessSecretKey: redis.String(secretKey),
		Name:            redis.String(name),
		Provider:        redis.String(provider),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(id))

	err = waitForCloudAccountToBeActive(ctx, id, client)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudCloudAccountRead(ctx, d, meta)
}

func waitForCloudAccountToBeActive(ctx context.Context, id int, client *apiClient) error {
	wait := &resource.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{cloud_accounts.StatusDraft, cloud_accounts.StatusChangeDraft},
		Target:  []string{cloud_accounts.StatusActive},
		Timeout: 1 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for cloud account %d to be active", id)

			account, err := client.client.CloudAccount.Get(ctx, id)
			if err != nil {
				return nil, "", err
			}

			status := redis.StringValue(account.Status)
			return status, status, nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

func resourceRedisCloudCloudAccountRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := client.client.CloudAccount.Get(ctx, id)
	if err != nil {
		if _, ok := err.(*cloud_accounts.NotFound); ok {
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	if err := d.Set("access_key_id", redis.StringValue(account.AccessKeyID)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("name", redis.StringValue(account.Name)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("provider_type", redis.StringValue(account.Provider)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("status", redis.StringValue(account.Status)); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceRedisCloudCloudAccountUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	accessKey := d.Get("access_key_id").(string)
	secretKey := d.Get("access_secret_key").(string)
	name := d.Get("name").(string)

	err = client.client.CloudAccount.Update(ctx, id, cloud_accounts.UpdateCloudAccount{
		AccessKeyID:     redis.String(accessKey),
		AccessSecretKey: redis.String(secretKey),
		Name:            redis.String(name),
	})

	if err != nil {
		return diag.FromErr(err)
	}

	return resourceRedisCloudCloudAccountRead(ctx, d, meta)
}

func resourceRedisCloudCloudAccountDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.client.CloudAccount.Delete(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
