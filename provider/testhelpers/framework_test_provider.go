package testhelpers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// FrameworkTestProviderTypeName is the provider type name for the throwaway plugin-framework
// provider built by FrameworkProviderFactories. Resources it serves therefore use type names
// like "test_<suffix>" (set in their Metadata via req.ProviderTypeName).
const FrameworkTestProviderTypeName = "test"

type frameworkTestProvider struct {
	resources    []func() resource.Resource
	providerData any
}

var _ provider.Provider = &frameworkTestProvider{}

func (p *frameworkTestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = FrameworkTestProviderTypeName
}

func (p *frameworkTestProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *frameworkTestProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Real resources pull their API client out of ProviderData in Configure, so handing them one here is
	// what lets a test drive the real implementation. Pass nil for a resource that needs no client.
	resp.ResourceData = p.providerData
	resp.DataSourceData = p.providerData
}

func (p *frameworkTestProvider) Resources(_ context.Context) []func() resource.Resource {
	return p.resources
}

func (p *frameworkTestProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// FrameworkProviderFactories builds a ProtoV5ProviderFactories map serving a throwaway plugin-framework
// provider (type name "test") that exposes only the given resources, each handed providerData in its
// Configure. Use it to unit-test resource behaviour — plan modifiers, CRUD, guards — via
// resource.UnitTest, with no real provider wiring: pass a *client.ApiClient built by NewAPIClient to
// drive real resources against an in-memory API. A fresh map is returned on each call to avoid cross-test
// mutation.
func FrameworkProviderFactories(providerData any, resources ...func() resource.Resource) map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		FrameworkTestProviderTypeName: providerserver.NewProtocol5WithError(&frameworkTestProvider{
			resources:    resources,
			providerData: providerData,
		}),
	}
}
