package iris

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func AgentMemoryResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "An Iris Agent Memory store backed by an existing Redis Cloud database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Resource identifier used for import.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the memory store.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"database_id": schema.StringAttribute{
				Required:      true,
				Description:   "Redis Cloud database ID. Changing the database replaces the store.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"created_at": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
				Description:   "UTC creation timestamp.",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "Provisioning state; READY means the store is usable.",
			},
			"subscription_id": schema.StringAttribute{
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
				Description:   "Database subscription ID.",
			},
			"failure_message": schema.StringAttribute{
				Computed:    true,
				Description: "Failure message when the store is unavailable.",
			},
			"failure_occurred_at": schema.StringAttribute{
				Computed:    true,
				Description: "UTC timestamp of the provisioning failure.",
			},
			"ownership_source": schema.StringAttribute{
				Computed:    true,
				Description: "Source of the effective ownership policy: STORE, TENANT or GLOBAL.",
			},
			"ownership_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether ownership checks are effective.",
			},
			"flush_on_delete": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Delete short-term sessions on destroy. Long-term memories are retained.",
				Default:     booldefault.StaticBool(false),
			},
			"extraction_strategy": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Memory extraction strategy.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"short_memory_ttl_seconds": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Short-term memory TTL in seconds.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"long_term_memory_ttl_seconds": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Long-term memory TTL in seconds.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"session_summary_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Extract session summaries into long-term memory.",
			},
			"extraction_interval_seconds": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Interval between extraction runs in seconds.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"ownership_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Ownership policy. INHERIT uses the tenant or global default.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"summarization_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Condense long sessions into a running summary.",
			},
			"summarization_trigger_strategy": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Session summarization trigger.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"summarization_event_threshold": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Event count for session summarization.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"summarization_retain_count": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of recent events to retain after summarization.",
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"exclusions_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable sensitive-data exclusions.",
			},
			"semantic_exclusions_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable semantic sensitive-data exclusions.",
			},
			"built_in_detectors_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable the built-in detector group.",
			},
			"custom_detectors_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable the custom detector group.",
			},
			"semantic_exclusions_prompt": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Instructions describing what semantic exclusion should remove.",
			},
			"llm_provider": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "LLM provider.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"llm_model": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "LLM model.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"llm_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "LLM API key. Write-only; omitted from plan and state. Increment llm_api_key_version to rotate. Omit the key and its version for providers that require no credentials.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"llm_api_key_version": schema.Int64Attribute{Optional: true, Description: "Positive credential rotation version. Increment whenever llm_api_key changes.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
			"embedding_provider": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Embedding provider.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"embedding_model": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Embedding model.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},
			"embedding_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Embedding API key. Write-only; omitted from plan and state. Increment embedding_api_key_version to rotate. Omit the key and its version for providers that require no credentials.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"embedding_api_key_version": schema.Int64Attribute{Optional: true, Description: "Positive credential rotation version. Increment whenever embedding_api_key changes.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
			"classification_provider": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Classification provider.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(classificationProviderChanged, "Changing the classification provider replaces the store.", "Changing the classification provider replaces the store.")},
			},
			"classification_model": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Classification model.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"classification_api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Classification API key. Write-only; omitted from plan and state. Increment classification_api_key_version to rotate. Omit the key and its version for providers that require no credentials.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"classification_api_key_version": schema.Int64Attribute{Optional: true, Description: "Positive credential rotation version. Increment whenever classification_api_key changes.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
			"endpoints":                      schema.SetAttribute{Computed: true, ElementType: endpointElementType, Description: "API endpoints for accessing the store."}}, Blocks: map[string]schema.Block{
			"timeouts":           timeouts.BlockAll(ctx),
			"custom_memory_type": customMemoryTypeBlock(),
			"built_in_detector":  builtInDetectorBlock(),
			"custom_detector":    customDetectorBlock(),
		}}
}

type AgentMemoryModel struct {
	Timeouts                     timeouts.Value `tfsdk:"timeouts"`
	ID                           types.String   `tfsdk:"id"`
	Name                         types.String   `tfsdk:"name"`
	DatabaseID                   types.String   `tfsdk:"database_id"`
	CreatedAt                    types.String   `tfsdk:"created_at"`
	State                        types.String   `tfsdk:"state"`
	SubscriptionID               types.String   `tfsdk:"subscription_id"`
	FailureMessage               types.String   `tfsdk:"failure_message"`
	FailureOccurredAt            types.String   `tfsdk:"failure_occurred_at"`
	OwnershipSource              types.String   `tfsdk:"ownership_source"`
	OwnershipEnabled             types.Bool     `tfsdk:"ownership_enabled"`
	FlushOnDelete                types.Bool     `tfsdk:"flush_on_delete"`
	ExtractionStrategy           types.String   `tfsdk:"extraction_strategy"`
	ShortMemoryTTLSeconds        types.Int64    `tfsdk:"short_memory_ttl_seconds"`
	LongTermMemoryTTLSeconds     types.Int64    `tfsdk:"long_term_memory_ttl_seconds"`
	SessionSummaryEnabled        types.Bool     `tfsdk:"session_summary_enabled"`
	ExtractionIntervalSeconds    types.Int64    `tfsdk:"extraction_interval_seconds"`
	OwnershipMode                types.String   `tfsdk:"ownership_mode"`
	SummarizationEnabled         types.Bool     `tfsdk:"summarization_enabled"`
	SummarizationTriggerStrategy types.String   `tfsdk:"summarization_trigger_strategy"`
	SummarizationEventThreshold  types.Int64    `tfsdk:"summarization_event_threshold"`
	SummarizationRetainCount     types.Int64    `tfsdk:"summarization_retain_count"`
	ExclusionsEnabled            types.Bool     `tfsdk:"exclusions_enabled"`
	SemanticExclusionsEnabled    types.Bool     `tfsdk:"semantic_exclusions_enabled"`
	BuiltInDetectorsEnabled      types.Bool     `tfsdk:"built_in_detectors_enabled"`
	CustomDetectorsEnabled       types.Bool     `tfsdk:"custom_detectors_enabled"`
	SemanticExclusionsPrompt     types.String   `tfsdk:"semantic_exclusions_prompt"`
	LLMProvider                  types.String   `tfsdk:"llm_provider"`
	LLMModel                     types.String   `tfsdk:"llm_model"`
	LLMAPIKey                    types.String   `tfsdk:"llm_api_key"`
	LLMAPIKeyVersion             types.Int64    `tfsdk:"llm_api_key_version"`
	EmbeddingProvider            types.String   `tfsdk:"embedding_provider"`
	EmbeddingModel               types.String   `tfsdk:"embedding_model"`
	EmbeddingAPIKey              types.String   `tfsdk:"embedding_api_key"`
	EmbeddingAPIKeyVersion       types.Int64    `tfsdk:"embedding_api_key_version"`
	ClassificationProvider       types.String   `tfsdk:"classification_provider"`
	ClassificationModel          types.String   `tfsdk:"classification_model"`
	ClassificationAPIKey         types.String   `tfsdk:"classification_api_key"`
	ClassificationAPIKeyVersion  types.Int64    `tfsdk:"classification_api_key_version"`
	Endpoints                    types.Set      `tfsdk:"endpoints"`
	CustomMemoryTypes            types.Set      `tfsdk:"custom_memory_type"`
	BuiltInDetectors             types.Set      `tfsdk:"built_in_detector"`
	CustomDetectors              types.Set      `tfsdk:"custom_detector"`
}

var _ resource.ResourceWithConfigValidators = (*agentMemoryResource)(nil)

func (*agentMemoryResource) ConfigValidators(context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.RequiredTogether(path.MatchRoot("llm_api_key"), path.MatchRoot("llm_api_key_version")),
		resourcevalidator.RequiredTogether(path.MatchRoot("embedding_api_key"), path.MatchRoot("embedding_api_key_version")),
		resourcevalidator.RequiredTogether(path.MatchRoot("classification_api_key"), path.MatchRoot("classification_api_key_version")),
	}
}

// Adding classification is supported; changing an existing provider is not.
func classificationProviderChanged(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() && !req.StateValue.IsNull() && !req.StateValue.Equal(req.PlanValue)
}
