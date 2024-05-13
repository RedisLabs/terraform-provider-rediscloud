package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

const RedisCloudUrlEnvVar = "REDISCLOUD_URL"

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:        schema.TypeString,
					Description: fmt.Sprintf("This is the URL of Redis Cloud and will default to `https://api.redislabs.com/v1`. This can also be set by the `%s` environment variable.", RedisCloudUrlEnvVar),
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc(RedisCloudUrlEnvVar, ""),
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
				"rediscloud_cloud_account":              dataSourceRedisCloudCloudAccount(),
				"rediscloud_data_persistence":           dataSourceRedisCloudDataPersistence(),
				"rediscloud_database":                   dataSourceRedisCloudDatabase(), // Deprecated
				"rediscloud_database_modules":           dataSourceRedisCloudDatabaseModules(),
				"rediscloud_payment_method":             dataSourceRedisCloudPaymentMethod(),
				"rediscloud_regions":                    dataSourceRedisCloudRegions(),
				"rediscloud_essentials_plan":            dataSourceRedisCloudEssentialsPlan(),
				"rediscloud_essentials_subscription":    dataSourceRedisCloudEssentialsSubscription(),
				"rediscloud_flexible_subscription":      dataSourceRedisCloudFlexibleSubscription(),
				"rediscloud_flexible_database":          dataSourceRedisCloudFlexibleDatabase(),
				"rediscloud_subscription":               dataSourceRedisCloudSubscription(), // Deprecated
				"rediscloud_subscription_peerings":      dataSourceRedisCloudSubscriptionPeerings(),
				"rediscloud_active_active_subscription": dataSourceRedisCloudActiveActiveSubscription(),
				"rediscloud_active_active_database":     dataSourceRedisCloudActiveActiveDatabase(),
				"rediscloud_acl_rule":                   dataSourceRedisCloudAclRule(),
				"rediscloud_acl_role":                   dataSourceRedisCloudAclRole(),
				"rediscloud_acl_user":                   dataSourceRedisCloudAclUser(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"rediscloud_cloud_account":                       resourceRedisCloudCloudAccount(),
				"rediscloud_essentials_subscription":             resourceRedisCloudEssentialsSubscription(),
				"rediscloud_flexible_subscription":               resourceRedisCloudFlexibleSubscription(),
				"rediscloud_flexible_database":                   resourceRedisCloudFlexibleDatabase(),
				"rediscloud_subscription":                        resourceRedisCloudSubscription(),         // Deprecated
				"rediscloud_subscription_database":               resourceRedisCloudSubscriptionDatabase(), // Deprecated
				"rediscloud_subscription_peering":                resourceRedisCloudSubscriptionPeering(),
				"rediscloud_active_active_subscription_database": resourceRedisCloudActiveActiveSubscriptionDatabase(), // Deprecated
				"rediscloud_active_active_subscription":          resourceRedisCloudActiveActiveSubscription(),
				"rediscloud_active_active_database":              resourceRedisCloudActiveActiveDatabase(),
				"rediscloud_active_active_subscription_regions":  resourceRedisCloudActiveActiveSubscriptionRegions(),
				"rediscloud_active_active_subscription_peering":  resourceRedisCloudActiveActiveSubscriptionPeering(),
				"rediscloud_acl_rule":                            resourceRedisCloudAclRule(),
				"rediscloud_acl_role":                            resourceRedisCloudAclRole(),
				"rediscloud_acl_user":                            resourceRedisCloudAclUser(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

// Lock that must be acquired when modifying something related to a subscription as only one _thing_ can modify a subscription and all sub-resources at any time
var subscriptionMutex = newPerIdLock()

type apiClient struct {
	client *rediscloudApi.Client
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var config []rediscloudApi.Option
		config = append(config, rediscloudApi.AdditionalUserAgent(p.UserAgent("terraform-provider-rediscloud", version)))

		url := d.Get("url").(string)
		apiKey := d.Get("api_key").(string)
		secretKey := d.Get("secret_key").(string)

		if url != "" {
			config = append(config, rediscloudApi.BaseURL(url))
		}

		if apiKey != "" && secretKey != "" {
			config = append(config, rediscloudApi.Auth(apiKey, secretKey))
		}

		if logging.IsDebugOrHigher() {
			config = append(config, rediscloudApi.LogRequests(true))
		}

		config = append(config, rediscloudApi.Logger(&debugLogger{}))

		client, err := rediscloudApi.NewClient(config...)
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
