package agentmemory

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &agentMemoryResource{}
	_ resource.ResourceWithConfigure   = &agentMemoryResource{}
	_ resource.ResourceWithImportState = &agentMemoryResource{}
	_ resource.ResourceWithModifyPlan  = &agentMemoryResource{}
)

type agentMemoryResource struct {
	client *client.ApiClient
}

func NewAgentMemoryResource() resource.Resource {
	return &agentMemoryResource{}
}

func (r *agentMemoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_memory"
}

func (r *agentMemoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *agentMemoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	useStateForUnknown := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	useListStateForUnknown := []planmodifier.List{listplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		Description: "Creates a Redis Agent Memory service backed by an existing Redis Cloud database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The Agent Memory store ID.",
				Computed:      true,
				PlanModifiers: useStateForUnknown,
			},
			"name": schema.StringAttribute{
				Description: "The name of the Agent Memory service.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(regexp.MustCompile(`.*\S.*`), "must contain at least one non-whitespace character"),
				},
			},
			"database_id": schema.Int64Attribute{
				Description:   "The ID of the Redis Cloud database that backs the Agent Memory service.",
				Required:      true,
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
				PlanModifiers: requiresReplace,
			},
			"short_term_ttl_seconds": schema.Int64Attribute{
				Description: "Short-term memory TTL in seconds. The API accepts 1 second to 1 year.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 31536000),
				},
			},
			"long_term_ttl_seconds": schema.Int64Attribute{
				Description: "Long-term memory TTL in seconds. The API accepts 1 second to 1 year.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 31536000),
				},
			},
			"extraction_strategy": schema.StringAttribute{
				Description: "How long-term memories are extracted from sessions. Currently only INSTRUCT is supported by the Agent Memory API.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("INSTRUCT"),
				},
			},
			"extraction_cadence_seconds": schema.Int64Attribute{
				Description: "How often the extraction pipeline runs while a session is active. When configured, the API currently accepts 60-600 seconds.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(60, 600),
				},
			},
			"endpoint": schema.StringAttribute{
				Description:   "The primary data-plane API endpoint for the Agent Memory service.",
				Computed:      true,
				PlanModifiers: useStateForUnknown,
			},
			"endpoints": schema.ListAttribute{
				Description:   "Regional data-plane API endpoints for the Agent Memory service.",
				Computed:      true,
				ElementType:   agentMemoryEndpointObjectType(),
				PlanModifiers: useListStateForUnknown,
			},
			"status": schema.StringAttribute{
				Description:   "The current Agent Memory provisioning status.",
				Computed:      true,
				PlanModifiers: useStateForUnknown,
			},
			"created_at": schema.StringAttribute{
				Description:   "The Agent Memory service creation timestamp.",
				Computed:      true,
				PlanModifiers: useStateForUnknown,
			},
		},
		Blocks: map[string]schema.Block{
			"llm":                         modelConfigBlock("Customer-managed LLM configuration. Configure this together with embedding to opt into customer-managed models."),
			"embedding":                   modelConfigBlock("Customer-managed embedding model configuration for long-term memory. Configure this together with llm to opt into customer-managed models."),
			"summarization":               summarizationBlock(),
			"long_term_memory_exclusions": longTermMemoryExclusionsBlock(),
			"custom_memory_types":         customMemoryTypesBlock(),
		},
	}

}

func modelConfigBlock(description string) schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: description,
		Attributes: map[string]schema.Attribute{
			"provider": schema.StringAttribute{
				Description: "Model provider. The Agent Memory API validates this against the providers supported by the service.",
				Optional:    true,
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "Model name advertised by the provider.",
				Optional:    true,
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"credentials": schema.SingleNestedBlock{
				Description: "Customer-supplied model provider credentials. Credentials are write-only in the Agent Memory API and are preserved in Terraform state when configured.",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Credential type. Currently only apiKey is supported.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("apiKey"),
						},
					},
					"api_key": schema.StringAttribute{
						Description: "Provider API key.",
						Optional:    true,
						Sensitive:   true,
					},
				},
			},
		},
	}
}

func summarizationBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Session summarization configuration.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether automatic session summarization is enabled.",
				Optional:    true,
				Computed:    true,
			},
			"trigger_strategy": schema.StringAttribute{
				Description: "What triggers a summarization cycle.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("event_count"),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"event_count": schema.SingleNestedBlock{
				Description: "Event-count based summarization thresholds.",
				Attributes: map[string]schema.Attribute{
					"threshold": schema.Int64Attribute{
						Description: "Number of messages in a session before summarization runs.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.Between(0, 10000),
						},
					},
					"retain_count": schema.Int64Attribute{
						Description: "Number of most recent messages retained in full after summarization.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.Int64{
							int64validator.Between(0, 10000),
						},
					},
				},
			},
		},
	}
}

func longTermMemoryExclusionsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Rules defining content that must not be retained in long-term memory.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether long-term memory exclusions are enabled.",
				Optional:    true,
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"semantic":           semanticExclusionBlock(),
			"built_in_detectors": builtInDetectorsBlock(),
			"custom_detectors":   customDetectorsBlock(),
		},
	}
}

func semanticExclusionBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Semantic exclusion configuration.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether semantic exclusions are enabled.",
				Optional:    true,
				Computed:    true,
			},
			"prompt": schema.StringAttribute{
				Description: "Prompt describing concepts that should not be kept in long-term memory.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(2000),
				},
			},
		},
	}
}

func builtInDetectorsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Built-in detector selections for common sensitive identifiers.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether built-in detectors are enabled.",
				Optional:    true,
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"detectors": schema.ListNestedBlock{
				Description: "Built-in detectors selected for the store.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(5),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: detectorSelectionAttributes(),
				},
			},
		},
	}
}

func customDetectorsBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Custom regex detector configuration.",
		Attributes: map[string]schema.Attribute{
			"enabled": schema.BoolAttribute{
				Description: "Whether custom detectors are enabled.",
				Optional:    true,
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"detectors": schema.ListNestedBlock{
				Description: "Custom detectors for tenant-specific or application-specific sensitive data.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(32),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Custom detector name.",
							Required:    true,
							Validators:  customNameValidator(),
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether this detector is enabled.",
							Required:    true,
						},
						"action": schema.StringAttribute{
							Description: "Action applied when this detector matches.",
							Optional:    true,
							Computed:    true,
							Validators:  actionValidator(),
						},
					},
					Blocks: map[string]schema.Block{
						"matcher": schema.SingleNestedBlock{
							Description: "Custom detector matcher.",
							Attributes: map[string]schema.Attribute{
								"kind": schema.StringAttribute{
									Description: "Matcher kind.",
									Required:    true,
									Validators: []validator.String{
										stringvalidator.OneOf("regex"),
									},
								},
							},
							Blocks: map[string]schema.Block{
								"regex": schema.SingleNestedBlock{
									Description: "Regex matcher configuration.",
									Attributes: map[string]schema.Attribute{
										"pattern": schema.StringAttribute{
											Description: "Regex pattern used by the matcher.",
											Required:    true,
											Validators: []validator.String{
												stringvalidator.LengthBetween(1, 512),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func customMemoryTypesBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Custom long-term memory types registered on the store. New types and extraction strategy changes are updated in place; redefining or removing existing types is blocked because the API does not support it.",
		Validators: []validator.List{
			listvalidator.SizeAtMost(3),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Description: "Custom memory type name.",
					Required:    true,
					Validators:  customNameValidator(),
				},
				"description": schema.StringAttribute{
					Description: "Custom memory type description.",
					Required:    true,
					Validators: []validator.String{
						stringvalidator.LengthBetween(1, 200),
					},
				},
			},
			Blocks: map[string]schema.Block{
				"fields":              customFieldsBlock(),
				"extraction_strategy": customExtractionStrategyBlock(),
			},
		},
	}
}

func customFieldsBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Structured fields for this custom memory type.",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Description: "Field name.",
					Required:    true,
					Validators:  customNameValidator(),
				},
				"description": schema.StringAttribute{
					Description: "Field description.",
					Required:    true,
					Validators: []validator.String{
						stringvalidator.LengthBetween(1, 200),
					},
				},
				"type": schema.StringAttribute{
					Description: "Field type.",
					Required:    true,
					Validators: []validator.String{
						stringvalidator.OneOf("str", "int", "float", "bool", "list[str]", "list[float]", "object"),
					},
				},
			},
		},
	}
}

func customExtractionStrategyBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		Description: "Extraction strategy for this custom memory type.",
		Attributes: map[string]schema.Attribute{
			"prompt": schema.StringAttribute{
				Description: "Prompt used to extract this custom memory type.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 10000),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether extraction is enabled for this custom memory type.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func detectorSelectionAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Description: "Built-in detector ID.",
			Required:    true,
			Validators: []validator.String{
				stringvalidator.OneOf("credit-card", "email", "ip-address", "phone", "us-ssn"),
			},
		},
		"enabled": schema.BoolAttribute{
			Description: "Whether this detector is enabled.",
			Required:    true,
		},
		"action": schema.StringAttribute{
			Description: "Action applied when this detector matches.",
			Optional:    true,
			Computed:    true,
			Validators:  actionValidator(),
		},
	}
}

func customNameValidator() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(1, 64),
		stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`), "must start with a letter and contain only letters, numbers, underscores, and hyphens"),
	}
}

func actionValidator() []validator.String {
	return []validator.String{stringvalidator.OneOf("redact", "drop")}
}

func (r *agentMemoryResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan AgentMemoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AgentMemoryResourceModel
	hasState := !req.State.Raw.IsNull()
	if hasState {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	validateModelConfigPlan(plan, state, hasState, resp)
	validateCustomMemoryTypesPlan(plan, state, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Summarization == nil {
		validateExclusionsPlan(plan, resp)
		return
	}

	if plan.Summarization.Enabled.IsNull() || plan.Summarization.Enabled.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("summarization").AtName("enabled"),
			"Missing summarization enabled flag",
			"summarization.enabled must be configured when the summarization block is present.",
		)
	}
	if plan.Summarization.EventCount == nil {
		validateExclusionsPlan(plan, resp)
		return
	}

	threshold := plan.Summarization.EventCount.Threshold
	retainCount := plan.Summarization.EventCount.RetainCount
	if threshold.IsNull() || threshold.IsUnknown() || retainCount.IsNull() || retainCount.IsUnknown() {
		return
	}

	if threshold.ValueInt64() <= retainCount.ValueInt64() {
		resp.Diagnostics.AddAttributeError(
			path.Root("summarization").AtName("event_count").AtName("threshold"),
			"Invalid summarization event count",
			"summarization.event_count.threshold must be greater than summarization.event_count.retain_count.",
		)
	}

	validateExclusionsPlan(plan, resp)
}

func validateExclusionsPlan(plan AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if plan.LongTermMemoryExclusions == nil {
		return
	}

	if plan.LongTermMemoryExclusions.Enabled.IsNull() || plan.LongTermMemoryExclusions.Enabled.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("long_term_memory_exclusions").AtName("enabled"),
			"Missing long-term memory exclusions enabled flag",
			"long_term_memory_exclusions.enabled must be configured when the long_term_memory_exclusions block is present.",
		)
	}
	if semantic := plan.LongTermMemoryExclusions.Semantic; semantic != nil && (semantic.Enabled.IsNull() || semantic.Enabled.IsUnknown()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("long_term_memory_exclusions").AtName("semantic").AtName("enabled"),
			"Missing semantic exclusion enabled flag",
			"long_term_memory_exclusions.semantic.enabled must be configured when the semantic block is present.",
		)
	}
	if builtIn := plan.LongTermMemoryExclusions.BuiltInDetectors; builtIn != nil && (builtIn.Enabled.IsNull() || builtIn.Enabled.IsUnknown()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("long_term_memory_exclusions").AtName("built_in_detectors").AtName("enabled"),
			"Missing built-in detectors enabled flag",
			"long_term_memory_exclusions.built_in_detectors.enabled must be configured when the built_in_detectors block is present.",
		)
	}
	custom := plan.LongTermMemoryExclusions.CustomDetectors
	if custom == nil {
		return
	}
	if custom.Enabled.IsNull() || custom.Enabled.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("long_term_memory_exclusions").AtName("custom_detectors").AtName("enabled"),
			"Missing custom detectors enabled flag",
			"long_term_memory_exclusions.custom_detectors.enabled must be configured when the custom_detectors block is present.",
		)
	}
	for i, detector := range custom.Detectors {
		if detector.Matcher == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("long_term_memory_exclusions").AtName("custom_detectors").AtName("detectors").AtListIndex(i).AtName("matcher"),
				"Missing custom detector matcher",
				"custom detector matcher must be configured.",
			)
			continue
		}
		if detector.Matcher.Regex == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("long_term_memory_exclusions").AtName("custom_detectors").AtName("detectors").AtListIndex(i).AtName("matcher").AtName("regex"),
				"Missing custom detector regex",
				"custom detector matcher.regex must be configured when matcher.kind is regex.",
			)
		}
	}
}

func (r *agentMemoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Missing Redis Cloud client", "The Redis Cloud client was not configured before import.")
		return
	}

	store, err := r.client.Client.AgentMemory.Get(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Agent Memory service", err.Error())
		return
	}

	var state AgentMemoryResourceModel
	readImportedAgentMemoryIntoModel(ctx, store, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
