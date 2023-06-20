package provider

import (
	"context"
	"fmt"
	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure RedisCloudProvider satisfies provider interface.
var _ provider.Provider = &RedisCloudProvider{}

// RedisCloudProvider defines the provider implementation.
type RedisCloudProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// RedisCloudProviderModel describes the provider data model.
type RedisCloudProviderModel struct {
	Url       types.String `tfsdk:"url"`
	ApiKey    types.String `tfsdk:"api_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

func (p *RedisCloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, _ *provider.MetadataResponse) {
}

func (p *RedisCloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("This is the URL of Redis Cloud and will default to `https://api.redislabs.com/v1`. This can also be set by the `%s` environment variable.", RedisCloudUrlEnvVar),
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("This is the Redis Cloud API key. It must be provided but can also be set by the `%s` environment variable.", rediscloudApi.AccessKeyEnvVar),
				Optional:            true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("This is the Redis Cloud API secret key. It must be provided but can also be set by the `%s` environment variable.", rediscloudApi.SecretKeyEnvVar),
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{},
	}
}

func (p *RedisCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Rediscloud client")

	var config []rediscloudApi.Option
	ua := fmt.Sprintf("Terraform/%s (+https://www.terraform.io) Terraform-Plugin-Framework terraform-provider-rediscloud/%s", req.TerraformVersion, p.version)
	config = append(config, rediscloudApi.AdditionalUserAgent(ua))

	// Retrieve provider data from configuration
	var data RedisCloudProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// All three values should be known before any other resource is applied
	if data.Url.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown Rediscloud url",
			"The provider cannot create the Rediscloud API client as there is an unknown configuration value for the Rediscloud API url. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the REDISCLOUD_URL environment variable.",
		)
	}

	if data.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Rediscloud access key",
			"The provider cannot create the Rediscloud API client as there is an unknown configuration value for the Rediscloud access key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the REDISCLOUD_ACCESS_KEY environment variable.",
		)
	}

	if data.SecretKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Unknown Rediscloud secret key",
			"The provider cannot create the Rediscloud API client as there is an unknown configuration value for the Rediscloud secret key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the REDISCLOUD_ACCESS_KEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override with Terraform configuration value if set.
	url := os.Getenv(RedisCloudUrlEnvVar)
	apiKey := os.Getenv(rediscloudApi.AccessKeyEnvVar)
	secretKey := os.Getenv(rediscloudApi.SecretKeyEnvVar)

	if !data.Url.IsNull() {
		url = data.Url.ValueString()
	}

	if !data.ApiKey.IsNull() {
		apiKey = data.ApiKey.ValueString()
	}

	if !data.SecretKey.IsNull() {
		secretKey = data.SecretKey.ValueString()
	}

	// If any of the expected configurations are missing, return errors with provider-specific guidance.
	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing Rediscloud API url",
			"The provider cannot create the Rediscloud API client as there is a missing or empty value for the Rediscloud API url. "+
				"Set the url value in the configuration or use the REDISCLOUD_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Rediscloud API key",
			"The provider cannot create the Rediscloud API client as there is a missing or empty value for the Rediscloud access key. "+
				"Set the api_key value in the configuration or use the REDISCLOUD_ACCESS_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if secretKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret_key"),
			"Missing Rediscloud API Password",
			"The provider cannot create the Rediscloud API client as there is a missing or empty value for the Rediscloud secret key. "+
				"Set the secret_key value in the configuration or use the REDISCLOUD_SECRET_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	config = append(config, rediscloudApi.BaseURL(url))
	config = append(config, rediscloudApi.Auth(apiKey, secretKey))

	// Analogue for sdkv2's logging.IsDebugOrHigher
	logLevel := strings.ToUpper(os.Getenv("TF_LOG"))
	if logLevel == "DEBUG" || logLevel == "TRACE" {
		config = append(config, rediscloudApi.LogRequests(true))
	}

	config = append(config, rediscloudApi.Logger(&DebugLogger{}))

	// TODO This block might not be necessary
	ctx = tflog.SetField(ctx, "rediscloud_url", url)
	ctx = tflog.SetField(ctx, "rediscloud_access_key", apiKey)
	ctx = tflog.SetField(ctx, "rediscloud_secret_key", secretKey)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "rediscloud_secret_key")

	tflog.Debug(ctx, "Creating Rediscloud client")

	client, err := rediscloudApi.NewClient(config...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Rediscloud API client",
			"An unexpected error occurred when creating the Rediscloud API client:"+err.Error(),
		)
		return
	}

	tflog.Info(ctx, "Configured Rediscloud client", map[string]any{"success": true})

	// Make the Rediscloud client available during DataSource and Resource type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RedisCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
	//return []func() resource.Resource{
	//	func() resource.Resource {
	//		return &resourceRedisCloudCloudAccount{}
	//	},
	//}
}

func (p *RedisCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func NewFrameworkProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RedisCloudProvider{
			version: version,
		}
	}
}
