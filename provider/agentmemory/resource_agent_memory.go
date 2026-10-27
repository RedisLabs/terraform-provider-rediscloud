package agentmemory

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                     = &agentMemoryResource{}
	_ resource.ResourceWithConfigure        = &agentMemoryResource{}
	_ resource.ResourceWithImportState      = &agentMemoryResource{}
	_ resource.ResourceWithModifyPlan       = &agentMemoryResource{}
	_ resource.ResourceWithConfigValidators = &agentMemoryResource{}
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

func (r *agentMemoryResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Redis Agent Memory service backed by an existing Redis Cloud database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Agent Memory store ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the Agent Memory service.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`.*\S.*`), "must contain at least one non-whitespace character"),
				},
			},
			"database_id": schema.Int64Attribute{
				Description: "The ID of the Redis Cloud database that backs the Agent Memory service. Changing it requires replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"short_term_ttl_seconds": schema.Int64Attribute{
				Description: "Short-term memory TTL in seconds.",
				Optional:    true,
				Computed:    true,
			},
			"long_term_ttl_seconds": schema.Int64Attribute{
				Description: "Long-term memory TTL in seconds.",
				Optional:    true,
				Computed:    true,
			},
			"extraction_strategy": schema.StringAttribute{
				Description: "How long-term memories are extracted from sessions. Changing an explicitly configured strategy requires replacement.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseNonNullStateForUnknown(),
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"extraction_cadence_seconds": schema.Int64Attribute{
				Description: "How often the extraction pipeline runs while a session is active, in seconds.",
				Optional:    true,
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "The primary data-plane API endpoint for the Agent Memory service.",
				Computed:    true,
			},
			"endpoints": schema.ListAttribute{
				Description: "Regional data-plane API endpoints for the Agent Memory service.",
				Computed:    true,
				ElementType: agentMemoryEndpointObjectType(),
			},
			"status": schema.StringAttribute{
				Description: "The current Agent Memory provisioning status.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The Agent Memory service creation timestamp.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
			"llm": schema.SingleNestedBlock{
				Description: "Customer-managed LLM configuration. Configure this together with embedding to opt into customer-managed models.",
				// Separate checks keep unknown values from hiding missing siblings.
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(path.MatchRelative().AtName("provider")),
					objectvalidator.AlsoRequires(path.MatchRelative().AtName("model")),
				},
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						Description: "Model provider. The Agent Memory API validates this against the providers supported by the service. Changing an explicitly configured provider requires replacement.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseNonNullStateForUnknown(),
							stringplanmodifier.RequiresReplaceIfConfigured(),
						},
					},
					"model": schema.StringAttribute{
						Description: "Model name advertised by the provider.",
						Optional:    true,
						Computed:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"credentials": schema.SingleNestedBlock{
						Description: "Customer-supplied model provider credentials. The API does not return secrets. api_key is write-only and is not persisted in Terraform plan or state.",
						// Separate checks keep unknown values from hiding missing siblings.
						Validators: []validator.Object{
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("type")),
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("api_key")),
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("api_key_version")),
						},
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Credential type validated by the Agent Memory API.",
								Optional:    true,
							},
							"api_key": schema.StringAttribute{
								Description: "Provider API key omitted from plan and state. Requires Terraform 1.16.7 or later. Increment api_key_version to rotate it.",
								Optional:    true,
								Sensitive:   true,
								WriteOnly:   true,
							},
							"api_key_version": schema.Int64Attribute{
								Description: "Positive version used to trigger an in-place write-only credential update. Change this value whenever api_key changes.",
								Optional:    true,
								Validators: []validator.Int64{
									int64validator.AtLeast(1),
								},
							},
						},
					},
				},
			},
			"embedding": schema.SingleNestedBlock{
				Description: "Customer-managed embedding model configuration for long-term memory. Configure this together with llm to opt into customer-managed models.",
				// Separate checks keep unknown values from hiding missing siblings.
				Validators: []validator.Object{
					objectvalidator.AlsoRequires(path.MatchRelative().AtName("provider")),
					objectvalidator.AlsoRequires(path.MatchRelative().AtName("model")),
				},
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{
						Description: "Model provider. The Agent Memory API validates this against the providers supported by the service. Changing an explicitly configured provider requires replacement.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseNonNullStateForUnknown(),
							stringplanmodifier.RequiresReplaceIfConfigured(),
						},
					},
					"model": schema.StringAttribute{
						Description: "Model name advertised by the provider. Changing an explicitly configured model requires replacement.",
						Optional:    true,
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseNonNullStateForUnknown(),
							stringplanmodifier.RequiresReplaceIfConfigured(),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"credentials": schema.SingleNestedBlock{
						Description: "Customer-supplied model provider credentials. The API does not return secrets. api_key is write-only and is not persisted in Terraform plan or state.",
						// Separate checks keep unknown values from hiding missing siblings.
						Validators: []validator.Object{
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("type")),
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("api_key")),
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("api_key_version")),
						},
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Description: "Credential type validated by the Agent Memory API.",
								Optional:    true,
							},
							"api_key": schema.StringAttribute{
								Description: "Provider API key omitted from plan and state. Requires Terraform 1.16.7 or later. Increment api_key_version to rotate it.",
								Optional:    true,
								Sensitive:   true,
								WriteOnly:   true,
							},
							"api_key_version": schema.Int64Attribute{
								Description: "Positive version used to trigger an in-place write-only credential update. Change this value whenever api_key changes.",
								Optional:    true,
								Validators: []validator.Int64{
									int64validator.AtLeast(1),
								},
							},
						},
					},
				},
			},
			"summarization": schema.SingleNestedBlock{
				Description: "Session summarization configuration.",
				// Required child attributes would also require an omitted single nested block.
				Validators: []validator.Object{objectvalidator.AlsoRequires(path.MatchRelative().AtName("enabled"))},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Whether automatic session summarization is enabled.",
						Optional:    true,
						Computed:    true,
					},
					"trigger_strategy": schema.StringAttribute{
						Description: "What triggers a summarization cycle. Accepted values and omitted defaults are determined by the Agent Memory API.",
						Optional:    true,
						Computed:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"event_count": schema.SingleNestedBlock{
						Description: "Event-count based summarization thresholds.",
						// Check configuration so computed or prior-state values cannot hide missing fields.
						// Separate checks keep unknown values from hiding missing siblings.
						Validators: []validator.Object{
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("threshold")),
							objectvalidator.AlsoRequires(path.MatchRelative().AtName("retain_count")),
						},
						Attributes: map[string]schema.Attribute{
							"threshold": schema.Int64Attribute{
								Description: "Number of messages in a session before summarization runs.",
								Optional:    true,
								Computed:    true,
							},
							"retain_count": schema.Int64Attribute{
								Description: "Number of most recent messages retained in full after summarization.",
								Optional:    true,
								Computed:    true,
							},
						},
					},
				},
			},
			"long_term_memory_exclusions": schema.SingleNestedBlock{
				Description: "Exclusion rules applied to newly extracted long-term memories. They do not filter session events, summaries, or memories written directly by the application, and do not re-evaluate existing memories. Semantic exclusions guide the extraction model but do not guarantee exclusion.",
				Validators:  []validator.Object{objectvalidator.AlsoRequires(path.MatchRelative().AtName("enabled"))},
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Whether long-term memory exclusions are enabled.",
						Optional:    true,
						Computed:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"semantic": schema.SingleNestedBlock{
						Description: "Semantic exclusion configuration.",
						Validators:  []validator.Object{objectvalidator.AlsoRequires(path.MatchRelative().AtName("enabled"))},
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
							},
						},
					},
					"built_in_detectors": schema.SingleNestedBlock{
						Description: "Built-in detector selections for common sensitive identifiers.",
						Validators:  []validator.Object{objectvalidator.AlsoRequires(path.MatchRelative().AtName("enabled"))},
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether built-in detectors are enabled.",
								Optional:    true,
								Computed:    true,
							},
						},
						Blocks: map[string]schema.Block{
							"detectors": schema.ListNestedBlock{
								CustomType:  NewDetectorSelectionListType(),
								Description: "Built-in detectors selected for the store.",
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Description: "Built-in detector ID.",
											Required:    true,
											Validators: []validator.String{
												// The API owns the detector catalogue; blank IDs are never valid.
												stringvalidator.RegexMatches(regexp.MustCompile(`.*\S.*`), "must contain at least one non-whitespace character"),
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
											Validators: []validator.String{
												stringvalidator.LengthAtLeast(1),
											},
										},
									},
								},
							},
						},
					},
					"custom_detectors": schema.SingleNestedBlock{
						Description: "Custom regex detector configuration.",
						Validators:  []validator.Object{objectvalidator.AlsoRequires(path.MatchRelative().AtName("enabled"))},
						Attributes: map[string]schema.Attribute{
							"enabled": schema.BoolAttribute{
								Description: "Whether custom detectors are enabled.",
								Optional:    true,
								Computed:    true,
							},
						},
						Blocks: map[string]schema.Block{
							"detectors": schema.ListNestedBlock{
								CustomType:  NewCustomDetectorListType(),
								Description: "Custom detectors for tenant-specific or application-specific sensitive data.",
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{
											Description: "Custom detector name.",
											Required:    true,
											// Name format and reserved names are validated by the API.
											Validators: []validator.String{
												// Empty names cannot round-trip: API strings are read as null when empty.
												stringvalidator.LengthAtLeast(1),
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
											Validators: []validator.String{
												stringvalidator.LengthAtLeast(1),
											},
										},
									},
									Blocks: map[string]schema.Block{
										"matcher": schema.SingleNestedBlock{
											Description: "Custom detector matcher.",
											Validators:  []validator.Object{objectvalidator.IsRequired()},
											Attributes: map[string]schema.Attribute{
												"kind": schema.StringAttribute{
													Description: "Matcher kind.",
													Required:    true,
													Validators: []validator.String{
														// The API validates supported kinds; empty values cannot round-trip.
														stringvalidator.LengthAtLeast(1),
													},
												},
											},
											Blocks: map[string]schema.Block{
												"regex": schema.SingleNestedBlock{
													Description: "Regex matcher configuration.",
													// Pattern is required only when this optional block is configured.
													Validators: []validator.Object{objectvalidator.AlsoRequires(path.MatchRelative().AtName("pattern"))},
													Attributes: map[string]schema.Attribute{
														"pattern": schema.StringAttribute{
															Description: "Regex pattern used by the matcher.",
															Optional:    true,
															Validators: []validator.String{
																stringvalidator.LengthAtLeast(1),
																nonEmptyRegexMatchValidator{},
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
					},
				},
			},
			"custom_memory_types": schema.ListNestedBlock{
				CustomType:  NewCustomMemoryTypeListType(),
				Description: "Custom long-term memory types registered on the store. Adding types and adding or updating their extraction strategies are supported in place. Removing or redefining existing types, or removing their extraction strategies, requires replacement of the entire Agent Memory service.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Custom memory type name.",
							Required:    true,
							// The API reserves these names for built-in memory types.
							Validators: []validator.String{
								// Empty names cannot round-trip: API strings are read as null when empty.
								stringvalidator.LengthAtLeast(1),
								stringvalidator.NoneOf("semantic", "episodic", "message", "session_summary_view"),
							},
						},
						"description": schema.StringAttribute{
							Description: "Custom memory type description.",
							Required:    true,
							Validators: []validator.String{
								// Empty descriptions become null when the API response is read into state.
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
					Blocks: map[string]schema.Block{
						"fields": schema.ListNestedBlock{
							CustomType:  NewCustomFieldListType(),
							Description: "Structured fields for this custom memory type.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "Field name.",
										Required:    true,
										Validators: []validator.String{
											// Empty names cannot round-trip: API strings are read as null when empty.
											stringvalidator.LengthAtLeast(1),
										},
									},
									"description": schema.StringAttribute{
										Description: "Field description.",
										Required:    true,
										Validators: []validator.String{
											// Empty descriptions become null when the API response is read into state.
											stringvalidator.LengthAtLeast(1),
										},
									},
									"type": schema.StringAttribute{
										Description: "Field type.",
										Required:    true,
										Validators: []validator.String{
											// Empty field types become null when the API response is read into state.
											stringvalidator.LengthAtLeast(1),
										},
									},
								},
							},
						},
						"extraction_strategy": schema.SingleNestedBlock{
							Description: "Extraction strategy for this custom memory type.",
							// Required child attributes would also make the absent block invalid.
							Validators: []validator.Object{
								objectvalidator.AlsoRequires(path.MatchRelative().AtName("prompt")),
							},
							Attributes: map[string]schema.Attribute{
								"prompt": schema.StringAttribute{
									Description: "Prompt used to extract this custom memory type.",
									Optional:    true,
									Validators: []validator.String{
										// The create API requires a non-blank custom extraction prompt.
										stringvalidator.LengthAtLeast(1),
									},
								},
								"enabled": schema.BoolAttribute{
									Description: "Whether extraction is enabled for this custom memory type.",
									Optional:    true,
									Computed:    true,
								},
							},
						},
					},
				},
			},
		},
	}

}

func (r *agentMemoryResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.RequiredTogether(path.MatchRoot("llm"), path.MatchRoot("embedding")),
	}
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

	// Configuration distinguishes an omitted strategy from its computed plan value.
	var config AgentMemoryResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
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

	modelConfig := plan
	if !hasState || state.LLM == nil || state.Embedding == nil {
		// New model blocks have no prior selectors to reuse. Configuration keeps
		// omitted selectors distinct from expressions that are still unknown.
		resp.Diagnostics.Append(req.Config.Get(ctx, &modelConfig)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	// Write-only values exist only in configuration, even when the rest comes from the plan.
	modelConfig.LLM = modelConfigWithWriteOnlyValue(modelConfig.LLM, config.LLM)
	modelConfig.Embedding = modelConfigWithWriteOnlyValue(modelConfig.Embedding, config.Embedding)
	validateModelConfigPlan(modelConfig, state, hasState, resp)
	validateCustomMemoryTypesPlan(ctx, plan, state, resp)
	validateCustomDetectorNamesPlan(ctx, plan, resp)
	validateCustomDetectorMatchersConfig(ctx, config, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	validateSummarizationPlan(plan, resp)
}

func validateSummarizationPlan(plan AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if plan.Summarization == nil || plan.Summarization.EventCount == nil {
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
}

func (r *agentMemoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import has no timeout configuration to read until the resource enters state.
	ctx, cancel := context.WithTimeout(ctx, agentMemoryReadDefaultTimeout)
	defer cancel()

	if r.client == nil {
		resp.Diagnostics.AddError("Missing Redis Cloud client", "The Redis Cloud client was not configured before import.")
		return
	}

	store, err := r.client.Client.AgentMemory.Get(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Agent Memory service", err.Error())
		return
	}

	// Import must validate the returned identity against the ID supplied by the user.
	// Import has no configuration; initialise the absent block with its custom type.
	timeoutType := timeouts.BlockAll(ctx).Type().(timeouts.Type)
	state := AgentMemoryResourceModel{
		ID:       types.StringValue(req.ID),
		Timeouts: timeouts.Value{Object: types.ObjectNull(timeoutType.AttrTypes)},
	}
	readImportedAgentMemoryIntoModel(ctx, store, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
