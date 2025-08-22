package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/account"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/acl"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/active_active"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/essentials"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/misc"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/private_service_connect"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"url": {
					Type:        schema.TypeString,
					Description: fmt.Sprintf("This is the URL of Redis Cloud and will default to `https://api.redislabs.com/v1`. This can also be set by the `%s` environment variable.", utils.RedisCloudUrlEnvVar),
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc(utils.RedisCloudUrlEnvVar, ""),
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
				"rediscloud_cloud_account":    account.DataSourceRedisCloudCloudAccount(),
				"rediscloud_data_persistence": misc.DataSourceRedisCloudDataPersistence(),
				// Note the difference in public data-source name and the file/method name.
				// This is to help the developer relate their changes to what they would see happening in the Redis Console.
				// <default> == flexible == pro
				"rediscloud_subscription":                       pro.DataSourceRedisCloudProSubscription(),
				"rediscloud_database":                           pro.DataSourceRedisCloudProDatabase(),
				"rediscloud_database_modules":                   misc.DataSourceRedisCloudDatabaseModules(),
				"rediscloud_payment_method":                     account.DataSourceRedisCloudPaymentMethod(),
				"rediscloud_regions":                            misc.DataSourceRedisCloudRegions(),
				"rediscloud_essentials_plan":                    essentials.DataSourceRedisCloudEssentialsPlan(),
				"rediscloud_essentials_subscription":            essentials.DataSourceRedisCloudEssentialsSubscription(),
				"rediscloud_essentials_database":                essentials.DataSourceRedisCloudEssentialsDatabase(),
				"rediscloud_subscription_peerings":              private_service_connect.DataSourceRedisCloudSubscriptionPeerings(),
				"rediscloud_private_service_connect":            private_service_connect.DataSourcePrivateServiceConnect(),
				"rediscloud_private_service_connect_endpoints":  private_service_connect.DataSourcePrivateServiceConnectEndpoints(),
				"rediscloud_active_active_subscription":         active_active.DataSourceRedisCloudActiveActiveSubscription(),
				"rediscloud_active_active_subscription_regions": active_active.DataSourceRedisCloudActiveActiveSubscriptionRegions(),

				// Note the difference in public data-source name and the file/method name.
				// active_active_subscription_database == active_active_database
				"rediscloud_active_active_subscription_database":             active_active.DataSourceRedisCloudActiveActiveDatabase(),
				"rediscloud_active_active_private_service_connect":           active_active.DataSourceActiveActivePrivateServiceConnect(),
				"rediscloud_active_active_private_service_connect_endpoints": private_service_connect.DataSourceActiveActivePrivateServiceConnectEndpoints(),
				"rediscloud_transit_gateway":                                 transitgateway.DataSourceTransitGateway(),
				"rediscloud_active_active_transit_gateway":                   transitgateway.DataSourceActiveActiveTransitGateway(),
				"rediscloud_acl_rule":                                        acl.DataSourceRedisCloudAclRule(),
				"rediscloud_acl_role":                                        acl.DataSourceRedisCloudAclRole(),
				"rediscloud_acl_user":                                        acl.DataSourceRedisCloudAclUser(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"rediscloud_cloud_account":           account.ResourceRedisCloudCloudAccount(),
				"rediscloud_essentials_subscription": essentials.ResourceRedisCloudEssentialsSubscription(),
				"rediscloud_essentials_database":     essentials.ResourceRedisCloudEssentialsDatabase(),
				// Note the difference in public resource name and the file/method name.
				// <default> == flexible == pro
				"rediscloud_subscription":                              pro.ResourceRedisCloudProSubscription(),
				"rediscloud_subscription_database":                     pro.ResourceRedisCloudProDatabase(),
				"rediscloud_subscription_peering":                      private_service_connect.ResourceRedisCloudSubscriptionPeering(),
				"rediscloud_private_service_connect":                   private_service_connect.ResourceRedisCloudPrivateServiceConnect(),
				"rediscloud_private_service_connect_endpoint":          private_service_connect.ResourceRedisCloudPrivateServiceConnectEndpoint(),
				"rediscloud_private_service_connect_endpoint_accepter": private_service_connect.ResourceRedisCloudPrivateServiceConnectEndpointAccepter(),
				"rediscloud_active_active_subscription":                active_active.ResourceRedisCloudActiveActiveSubscription(),
				// Note the difference in public resource name and the file/method name.
				// active_active_subscription_database == active_active_database
				"rediscloud_active_active_subscription_database":                     active_active.ResourceRedisCloudActiveActiveDatabase(),
				"rediscloud_active_active_subscription_regions":                      active_active.ResourceRedisCloudActiveActiveSubscriptionRegions(),
				"rediscloud_active_active_subscription_peering":                      private_service_connect.ResourceRedisCloudActiveActiveSubscriptionPeering(),
				"rediscloud_active_active_private_service_connect":                   private_service_connect.ResourceRedisCloudActiveActivePrivateServiceConnect(),
				"rediscloud_active_active_private_service_connect_endpoint":          private_service_connect.ResourceRedisCloudActiveActivePrivateServiceConnectEndpoint(),
				"rediscloud_active_active_private_service_connect_endpoint_accepter": private_service_connect.ResourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepter(),
				"rediscloud_transit_gateway_attachment":                              transitgateway.ResourceRedisCloudTransitGatewayAttachment(),
				"rediscloud_acl_rule":                                                acl.ResourceRedisCloudAclRule(),
				"rediscloud_acl_role":                                                acl.ResourceRedisCloudAclRole(),
				"rediscloud_acl_user":                                                acl.ResourceRedisCloudAclUser(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
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

		return &utils.ApiClient{
			Client: client,
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
