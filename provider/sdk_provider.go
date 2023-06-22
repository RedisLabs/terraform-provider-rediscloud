package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
	"os"
	"strings"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

const RedisCloudUrlEnvVar = "REDISCLOUD_URL"

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func NewSdkProvider(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:        schema.TypeString,
					Description: fmt.Sprintf("This is the URL of Redis Cloud and will default to `https://api.redislabs.com/v1`. This can also be set by the `%s` environment variable.", RedisCloudUrlEnvVar),
					Optional:    true,
				},
				"api_key": {
					Type:        schema.TypeString,
					Description: fmt.Sprintf("This is the Redis Cloud API key. It must be provided but can also be set by the `%s` environment variable.", rediscloudApi.AccessKeyEnvVar),
					Optional:    true,
				},
				"secret_key": {
					Type:        schema.TypeString,
					Description: fmt.Sprintf("This is the Redis Cloud API secret key. It must be provided but can also be set by the `%s` environment variable.", rediscloudApi.SecretKeyEnvVar),
					Optional:    true,
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"rediscloud_cloud_account":         dataSourceRedisCloudCloudAccount(),
				"rediscloud_data_persistence":      dataSourceRedisCloudDataPersistence(),
				"rediscloud_database":              dataSourceRedisCloudDatabase(),
				"rediscloud_database_modules":      dataSourceRedisCloudDatabaseModules(),
				"rediscloud_payment_method":        dataSourceRedisCloudPaymentMethod(),
				"rediscloud_regions":               dataSourceRedisCloudRegions(),
				"rediscloud_subscription":          dataSourceRedisCloudSubscription(),
				"rediscloud_subscription_peerings": dataSourceRedisCloudSubscriptionPeerings(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"rediscloud_cloud_account":                       resourceRedisCloudCloudAccount(),
				"rediscloud_subscription":                        resourceRedisCloudSubscription(),
				"rediscloud_subscription_database":               resourceRedisCloudSubscriptionDatabase(),
				"rediscloud_subscription_peering":                resourceRedisCloudSubscriptionPeering(),
				"rediscloud_active_active_subscription_database": resourceRedisCloudActiveActiveSubscriptionDatabase(),
				"rediscloud_active_active_subscription":          resourceRedisCloudActiveActiveSubscription(),
				"rediscloud_active_active_subscription_regions":  resourceRedisCloudActiveActiveSubscriptionRegions(),
				"rediscloud_active_active_subscription_peering":  resourceRedisCloudActiveActiveSubscriptionPeering(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var config []rediscloudApi.Option
		config = append(config, rediscloudApi.AdditionalUserAgent(p.UserAgent("terraform-provider-rediscloud", version)))

		url := d.Get("url").(string)
		apiKey := d.Get("api_key").(string)
		secretKey := d.Get("secret_key").(string)

		// Replacement for DefaultFunc in schema, which muxing doesn't allow
		if url == "" {
			url = os.Getenv(RedisCloudUrlEnvVar)
		}
		config = append(config, rediscloudApi.BaseURL(url))

		if apiKey != "" && secretKey != "" {
			config = append(config, rediscloudApi.Auth(apiKey, secretKey))
		}

		if logging.IsDebugOrHigher() {
			config = append(config, rediscloudApi.LogRequests(true))
		}

		config = append(config, rediscloudApi.Logger(&sdkDebugLogger{}))

		client, err := rediscloudApi.NewClient(config...)
		if err != nil {
			return nil, diag.FromErr(err)
		}

		return &apiClient{
			client: client,
		}, nil
	}
}

type sdkDebugLogger struct{}

func (d *sdkDebugLogger) Printf(format string, v ...interface{}) {
	log.Printf("[DEBUG] [rediscloud-go-api] "+format, v...)
}

func (d *sdkDebugLogger) Println(v ...interface{}) {
	var items []string
	for _, i := range v {
		items = append(items, fmt.Sprintf("%s", i))
	}
	log.Printf("[DEBUG] [rediscloud-go-api] %s", strings.Join(items, " "))
}
