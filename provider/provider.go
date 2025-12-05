package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

	client2 "github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/privatelink"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/transitgateway"
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
				"rediscloud_cloud_account":    dataSourceRedisCloudCloudAccount(),
				"rediscloud_data_persistence": dataSourceRedisCloudDataPersistence(),
				// Note the difference in public data-source name and the file/method name.
				// This is to help the developer relate their changes to what they would see happening in the Redis Console.
				// <default> == flexible == pro
				"rediscloud_subscription":                               pro.DataSourceRedisCloudProSubscription(),
				"rediscloud_database":                                   pro.DataSourceRedisCloudProDatabase(),
				"rediscloud_database_modules":                           dataSourceRedisCloudDatabaseModules(),
				"rediscloud_payment_method":                             dataSourceRedisCloudPaymentMethod(),
				"rediscloud_regions":                                    dataSourceRedisCloudRegions(),
				"rediscloud_essentials_plan":                            dataSourceRedisCloudEssentialsPlan(),
				"rediscloud_essentials_subscription":                    dataSourceRedisCloudEssentialsSubscription(),
				"rediscloud_essentials_database":                        dataSourceRedisCloudEssentialsDatabase(),
				"rediscloud_subscription_peerings":                      dataSourceRedisCloudSubscriptionPeerings(),
				"rediscloud_private_service_connect":                    dataSourcePrivateServiceConnect(),
				"rediscloud_private_service_connect_endpoints":          dataSourcePrivateServiceConnectEndpoints(),
				"rediscloud_active_active_subscription":                 dataSourceRedisCloudActiveActiveSubscription(),
				"rediscloud_active_active_subscription_regions":         dataSourceRedisCloudActiveActiveSubscriptionRegions(),
				"rediscloud_private_link":                               privatelink.DataSourcePrivateLink(),
				"rediscloud_private_link_endpoint_script":               privatelink.DataSourcePrivateLinkEndpointScript(),
				"rediscloud_active_active_private_link":                 privatelink.DataSourceActiveActivePrivateLink(),
				"rediscloud_active_active_private_link_endpoint_script": privatelink.DataSourceActiveActivePrivateLinkEndpointScript(),

				// Note the difference in public data-source name and the file/method name.
				// active_active_subscription_database == active_active_database
				"rediscloud_active_active_subscription_database":             dataSourceRedisCloudActiveActiveDatabase(),
				"rediscloud_active_active_private_service_connect":           dataSourceActiveActivePrivateServiceConnect(),
				"rediscloud_active_active_private_service_connect_endpoints": dataSourceActiveActivePrivateServiceConnectEndpoints(),
				"rediscloud_transit_gateway":                                 dataSourceTransitGateway(),
				"rediscloud_active_active_transit_gateway":                   dataSourceActiveActiveTransitGateway(),
				"rediscloud_transit_gateway_invitations":                     transitgateway.DataSourceRedisCloudTransitGatewayInvitations(),
				"rediscloud_active_active_transit_gateway_invitations":       transitgateway.DataSourceRedisCloudActiveActiveTransitGatewayInvitations(),
				"rediscloud_acl_rule":                                        dataSourceRedisCloudAclRule(),
				"rediscloud_acl_role":                                        dataSourceRedisCloudAclRole(),
				"rediscloud_acl_user":                                        dataSourceRedisCloudAclUser(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"rediscloud_cloud_account":           resourceRedisCloudCloudAccount(),
				"rediscloud_essentials_subscription": resourceRedisCloudEssentialsSubscription(),
				"rediscloud_essentials_database":     resourceRedisCloudEssentialsDatabase(),
				// Note the difference in public resource name and the file/method name.
				// <default> == flexible == pro
				"rediscloud_subscription":                              pro.ResourceRedisCloudProSubscription(),
				"rediscloud_subscription_database":                     pro.ResourceRedisCloudProDatabase(),
				"rediscloud_subscription_peering":                      resourceRedisCloudSubscriptionPeering(),
				"rediscloud_private_service_connect":                   resourceRedisCloudPrivateServiceConnect(),
				"rediscloud_private_service_connect_endpoint":          resourceRedisCloudPrivateServiceConnectEndpoint(),
				"rediscloud_private_service_connect_endpoint_accepter": resourceRedisCloudPrivateServiceConnectEndpointAccepter(),
				"rediscloud_private_link":                              privatelink.ResourceRedisCloudPrivateLink(),
				"rediscloud_active_active_private_link":                privatelink.ResourceRedisCloudActiveActivePrivateLink(),

				"rediscloud_active_active_subscription": resourceRedisCloudActiveActiveSubscription(),
				// Note the difference in public resource name and the file/method name.
				// active_active_subscription_database == active_active_database
				"rediscloud_active_active_subscription_database":                     resourceRedisCloudActiveActiveDatabase(),
				"rediscloud_active_active_subscription_regions":                      resourceRedisCloudActiveActiveSubscriptionRegions(),
				"rediscloud_active_active_subscription_peering":                      resourceRedisCloudActiveActiveSubscriptionPeering(),
				"rediscloud_active_active_private_service_connect":                   resourceRedisCloudActiveActivePrivateServiceConnect(),
				"rediscloud_active_active_private_service_connect_endpoint":          resourceRedisCloudActiveActivePrivateServiceConnectEndpoint(),
				"rediscloud_active_active_private_service_connect_endpoint_accepter": resourceRedisCloudActiveActivePrivateServiceConnectEndpointAccepter(),
				"rediscloud_transit_gateway_attachment":                              resourceRedisCloudTransitGatewayAttachment(),
				"rediscloud_active_active_transit_gateway_attachment":                resourceRedisCloudActiveActiveTransitGatewayAttachment(),
				"rediscloud_transit_gateway_invitation_acceptor":                     transitgateway.ResourceRedisCloudTransitGatewayInvitationAcceptor(),
				//"rediscloud_active_active_transit_gateway_invitation_acceptor":       transitgateway.ResourceRedisCloudActiveActiveTransitGatewayInvitationAcceptor(),
				"rediscloud_acl_rule": resourceRedisCloudAclRule(),
				"rediscloud_acl_role": resourceRedisCloudAclRole(),
				"rediscloud_acl_user": resourceRedisCloudAclUser(),
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

		return &client2.ApiClient{
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
