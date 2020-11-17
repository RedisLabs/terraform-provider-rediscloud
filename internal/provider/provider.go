package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
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
				"rediscloud_cloud_account":    dataSourceRedisCloudCloudAccount(),
				"rediscloud_data_persistence": dataSourceRedisCloudDataPersistence(),
				"rediscloud_database_modules": dataSourceRedisCloudDatabaseModules(),
				"rediscloud_payment_method":   dataSourceRedisCloudPaymentMethod(),
				"rediscloud_regions":          dataSourceRedisCloudRegions(),
        "rediscloud_subscription":     dataSourceRedisCloudSubscription(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"rediscloud_cloud_account": resourceRedisCloudCloudAccount(),
				"rediscloud_subscription":  resourceRedisCloudSubscription(),
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
			config = append(config, rediscloud_api.BaseURL(url))
		}

		if apiKey != "" && secretKey != "" {
			config = append(config, rediscloud_api.Auth(apiKey, secretKey))
		}

		if logging.IsDebugOrHigher() {
			config = append(config, rediscloud_api.LogRequests(true))
		}

		config = append(config, rediscloud_api.Logger(&debugLogger{}))

		client, err := rediscloud_api.NewClient(config...)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return &apiClient{
			client: client,
		}, nil
	}
}

type debugLogger struct{}

func (d *debugLogger) Printf(format string, v ...interface{}) {
	log.Printf("[DEBUG] [rediscloud-go-api] "+format, v...)
}

func (d *debugLogger) Println(v ...interface{}) {
	var items []string
	for _, i := range v {
		items = append(items, fmt.Sprintf("%s", i))
	}
	log.Printf("[DEBUG] [rediscloud-go-api] %s", strings.Join(items, " "))
}
