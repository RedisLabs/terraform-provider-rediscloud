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
		CreateContext: resourceRedisCloudCloudAccountCreate,
		ReadContext:   resourceRedisCloudCloudAccountRead,
		UpdateContext: resourceRedisCloudCloudAccountUpdate,
		DeleteContext: resourceRedisCloudCloudAccountDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext, // TODO validate that this is in the right format
		},

		Schema: map[string]*schema.Schema{
			"access_key_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"access_secret_key": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"console_password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"console_username": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"provider_type": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validateDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
				ForceNew:         true,
			},
			"sign_in_login_url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceRedisCloudCloudAccountCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*apiClient)

	accessKey := d.Get("access_key_id").(string)
	secretKey := d.Get("access_secret_key").(string)
	consolePassword := d.Get("console_password").(string)
	consoleUsername := d.Get("console_username").(string)
	name := d.Get("name").(string)
	provider := d.Get("provider_type").(string)
	signInLoginUrl := d.Get("sign_in_login_url").(string)

	id, err := client.client.CloudAccount.Create(ctx, cloud_accounts.CreateCloudAccount{
		AccessKeyID:     redis.String(accessKey),
		AccessSecretKey: redis.String(secretKey),
		ConsoleUsername: redis.String(consoleUsername),
		ConsolePassword: redis.String(consolePassword),
		Name:            redis.String(name),
		Provider:        redis.String(provider),
		SignInLoginURL:  redis.String(signInLoginUrl),
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
	consolePassword := d.Get("console_password").(string)
	consoleUsername := d.Get("console_username").(string)
	name := d.Get("name").(string)
	signInLoginUrl := d.Get("sign_in_login_url").(string)

	err = client.client.CloudAccount.Update(ctx, id, cloud_accounts.UpdateCloudAccount{
		AccessKeyID:     redis.String(accessKey),
		AccessSecretKey: redis.String(secretKey),
		ConsoleUsername: redis.String(consoleUsername),
		ConsolePassword: redis.String(consolePassword),
		Name:            redis.String(name),
		SignInLoginURL:  redis.String(signInLoginUrl),
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
