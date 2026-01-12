package provider

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	client2 "github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
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
			"console_password": {
				Description: "Cloud provider management console password",
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
			},
			"console_username": {
				Description: "Cloud provider management console username",
				Type:        schema.TypeString,
				Required:    true,
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
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(cloud_accounts.ProviderValues(), false)),
				ForceNew:         true,
			},
			"sign_in_login_url": {
				Description: "Cloud provider management console login URL",
				Type:        schema.TypeString,
				Required:    true,
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
	client := meta.(*client2.ApiClient)

	accessKey := d.Get("access_key_id").(string)
	secretKey := d.Get("access_secret_key").(string)
	consolePassword := d.Get("console_password").(string)
	consoleUsername := d.Get("console_username").(string)
	name := d.Get("name").(string)
	provider := d.Get("provider_type").(string)
	signInLoginUrl := d.Get("sign_in_login_url").(string)

	id, err := client.Client.CloudAccount.Create(ctx, cloud_accounts.CreateCloudAccount{
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

func waitForCloudAccountToBeActive(ctx context.Context, id int, client *client2.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:   10 * time.Second,
		Pending: []string{cloud_accounts.StatusDraft, cloud_accounts.StatusChangeDraft},
		Target:  []string{cloud_accounts.StatusActive},
		Timeout: 1 * time.Minute,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for cloud account %d to be active", id)

			account, err := client.Client.CloudAccount.Get(ctx, id)
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
	client := meta.(*client2.ApiClient)

	var diags diag.Diagnostics

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	account, err := client.Client.CloudAccount.Get(ctx, id)
	if err != nil {
		notFound := &cloud_accounts.NotFound{}
		if errors.As(err, &notFound) {
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
	client := meta.(*client2.ApiClient)

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

	err = client.Client.CloudAccount.Update(ctx, id, cloud_accounts.UpdateCloudAccount{
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
	client := meta.(*client2.ApiClient)

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Client.CloudAccount.Delete(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
