package provider

import (
	"context"
	"fmt"
	"os"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/activeactive"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &redisCloudFrameworkProvider{}

// redisCloudFrameworkProvider is the Plugin Framework implementation of the provider.
type redisCloudFrameworkProvider struct {
	version string
}

// redisCloudProviderModel describes the provider data model.
type redisCloudProviderModel struct {
	Url       types.String `tfsdk:"url"`
	ApiKey    types.String `tfsdk:"api_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

// NewFrameworkProvider returns a new Plugin Framework provider instance.
func NewFrameworkProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &redisCloudFrameworkProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name.
func (p *redisCloudFrameworkProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "rediscloud"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *redisCloudFrameworkProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Redis Enterprise Cloud.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: fmt.Sprintf("This is the URL of Redis Cloud and will default to `https://api.redislabs.com/v1`. This can also be set by the `%s` environment variable.", RedisCloudUrlEnvVar),
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: fmt.Sprintf("This is the Redis Cloud API key. It must be provided but can also be set by the `%s` environment variable.", rediscloudApi.AccessKeyEnvVar),
				Optional:    true,
			},
			"secret_key": schema.StringAttribute{
				Description: fmt.Sprintf("This is the Redis Cloud API secret key. It must be provided but can also be set by the `%s` environment variable.", rediscloudApi.SecretKeyEnvVar),
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares the Redis Cloud API client for data sources and resources.
func (p *redisCloudFrameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Redis Cloud client")

	var config redisCloudProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Default values from environment variables
	url := os.Getenv(RedisCloudUrlEnvVar)
	apiKey := os.Getenv(rediscloudApi.AccessKeyEnvVar)
	secretKey := os.Getenv(rediscloudApi.SecretKeyEnvVar)

	// Override with config values if provided
	if !config.Url.IsNull() {
		url = config.Url.ValueString()
	}
	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}
	if !config.SecretKey.IsNull() {
		secretKey = config.SecretKey.ValueString()
	}

	// Validate required credentials
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Redis Cloud API Key",
			fmt.Sprintf("The provider cannot create the Redis Cloud API client as there is a missing or empty value for the Redis Cloud API key. "+
				"Set the api_key value in the configuration or use the %s environment variable.", rediscloudApi.AccessKeyEnvVar),
		)
	}

	if secretKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Missing Redis Cloud API Secret Key",
			fmt.Sprintf("The provider cannot create the Redis Cloud API client as there is a missing or empty value for the Redis Cloud API secret key. "+
				"Set the secret_key value in the configuration or use the %s environment variable.", rediscloudApi.SecretKeyEnvVar),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Build API client configuration
	var clientConfig []rediscloudApi.Option
	clientConfig = append(clientConfig, rediscloudApi.AdditionalUserAgent(fmt.Sprintf("terraform-provider-rediscloud/%s", p.version)))

	if url != "" {
		clientConfig = append(clientConfig, rediscloudApi.BaseURL(url))
	}

	clientConfig = append(clientConfig, rediscloudApi.Auth(apiKey, secretKey))

	// Enable request logging in debug mode
	clientConfig = append(clientConfig, rediscloudApi.LogRequests(true))
	clientConfig = append(clientConfig, rediscloudApi.Logger(&frameworkDebugLogger{}))

	// Create the API client
	apiClient, err := rediscloudApi.NewClient(clientConfig...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Redis Cloud API Client",
			"An unexpected error occurred when creating the Redis Cloud API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	// Wrap in ApiClient for compatibility with existing code
	wrappedClient := &client.ApiClient{
		Client: apiClient,
	}

	// Make the client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = wrappedClient
	resp.ResourceData = wrappedClient

	tflog.Info(ctx, "Configured Redis Cloud client", map[string]any{"success": true})
}

// Resources defines the resources implemented in the provider.
func (p *redisCloudFrameworkProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		activeactive.NewActiveActiveDatabaseResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *redisCloudFrameworkProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Data sources will be migrated in future PRs
	}
}

// frameworkDebugLogger implements the rediscloud-go-api Logger interface for Plugin Framework.
type frameworkDebugLogger struct{}

func (l *frameworkDebugLogger) Printf(format string, v ...interface{}) {
	tflog.Debug(context.Background(), fmt.Sprintf("[rediscloud-go-api] "+format, v...))
}

func (l *frameworkDebugLogger) Println(v ...interface{}) {
	tflog.Debug(context.Background(), fmt.Sprintf("[rediscloud-go-api] %v", v))
}
