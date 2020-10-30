package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rediscloud_api "github.com/RedisLabs/rediscloud-go-api"
)

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("REDISCLOUD_URL", ""),
				},
				"api_key": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"secret_key": {
					Type:     schema.TypeString,
					Optional: true,
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"rediscloud_payment_method": dataSourceRedisCloudPaymentMethod(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"rediscloud_subscription": resourceRedisCloudSubscription(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type apiClient struct {
	client *rediscloud_api.Client
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var config []rediscloud_api.Option
		config = append(config, rediscloud_api.AdditionalUserAgent(p.UserAgent("terraform-provider-rediscloud", version)))

		url := d.Get("url").(string)
		apiKey := d.Get("api_key").(string)
		secretKey := d.Get("secret_key").(string)

		if url != "" {
			config = append(config, rediscloud_api.BaseUrl(url))
		}

		if apiKey != "" && secretKey != "" {
			config = append(config, rediscloud_api.Auth(apiKey, secretKey))
		}

		client, err := rediscloud_api.NewClient(config...)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return &apiClient{
			client: client,
		}, nil
	}
}
