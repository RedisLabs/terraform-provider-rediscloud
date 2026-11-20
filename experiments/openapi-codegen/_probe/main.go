// This probe reports generator limitations; it does not exercise API operations.
package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"

	extractor "github.com/RedisLabs/terraform-provider-rediscloud/experiments/openapi-codegen/_generated/agent-memory/resource_agent_memory_extractor"
	memorytype "github.com/RedisLabs/terraform-provider-rediscloud/experiments/openapi-codegen/_generated/agent-memory/resource_agent_memory_type"
	apikey "github.com/RedisLabs/terraform-provider-rediscloud/experiments/openapi-codegen/_generated/identity-adapted/resource_identity_api_key"
	identityprovider "github.com/RedisLabs/terraform-provider-rediscloud/experiments/openapi-codegen/_generated/identity/resource_identity_provider"
	grants "github.com/RedisLabs/terraform-provider-rediscloud/experiments/openapi-codegen/_generated/identity/resource_identity_subject_grant_set"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

type probeResource struct {
	name   string
	schema schema.Schema
}

func (r *probeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.name
}
func (r *probeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = r.schema
}
func (*probeResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {}
func (*probeResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse)       {}
func (*probeResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {}
func (*probeResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {}

func probe(name string, build func(context.Context) schema.Schema) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("%s: PANIC: %v\n", name, p)
		}
	}()
	ctx := context.Background()
	s := build(ctx)
	fmt.Printf("%s: schema constructed (%d attributes)\n", name, len(s.Attributes))
	factories := testhelpers.FrameworkProviderFactories(nil, func() resource.Resource { return &probeResource{name: name, schema: s} })
	server, err := factories[testhelpers.FrameworkTestProviderTypeName]()
	if err != nil {
		fmt.Printf("%s: server error: %v\n", name, err)
		return
	}
	resp, err := server.GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
	if err != nil {
		fmt.Printf("%s: protocol 5 error: %v\n", name, err)
		return
	}
	errors := 0
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov5.DiagnosticSeverityError {
			errors++
			fmt.Printf("%s: protocol 5 diagnostic: %s: %s\n", name, d.Summary, d.Detail)
		}
	}
	if errors == 0 {
		fmt.Printf("%s: protocol 5 schema OK\n", name)
	}
}
func main() {
	probe("agent_memory_extractor", extractor.AgentMemoryExtractorResourceSchema)
	probe("agent_memory_type", memorytype.AgentMemoryTypeResourceSchema)
	probe("identity_api_key", apikey.IdentityApiKeyResourceSchema)
	probe("identity_provider", identityprovider.IdentityProviderResourceSchema)
	probe("identity_subject_grant_set", grants.IdentitySubjectGrantSetResourceSchema)
}
