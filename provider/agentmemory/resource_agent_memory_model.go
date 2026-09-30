package agentmemory

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

type AgentMemoryResourceModel struct {
	ID                       types.String                       `tfsdk:"id"`
	Name                     types.String                       `tfsdk:"name"`
	DatabaseID               types.Int64                        `tfsdk:"database_id"`
	ShortTermTTLSeconds      types.Int64                        `tfsdk:"short_term_ttl_seconds"`
	LongTermTTLSeconds       types.Int64                        `tfsdk:"long_term_ttl_seconds"`
	ExtractionCadenceSeconds types.Int64                        `tfsdk:"extraction_cadence_seconds"`
	Summarization            *AgentMemorySummarizationModel     `tfsdk:"summarization"`
	LongTermMemoryExclusions *AgentMemoryExclusionsModel        `tfsdk:"long_term_memory_exclusions"`
	CustomMemoryTypes        []AgentMemoryCustomMemoryTypeModel `tfsdk:"custom_memory_types"`
	Endpoint                 types.String                       `tfsdk:"endpoint"`
	Endpoints                types.List                         `tfsdk:"endpoints"`
	Status                   types.String                       `tfsdk:"status"`
	CreatedAt                types.String                       `tfsdk:"created_at"`
}

type AgentMemoryEndpointModel struct {
	URL          types.String `tfsdk:"url"`
	Provider     types.String `tfsdk:"provider"`
	Region       types.String `tfsdk:"region"`
	EgressIPs    types.List   `tfsdk:"egress_ips"`
	IsAccessible types.Bool   `tfsdk:"is_accessible"`
}

var agentMemoryEndpointAttrTypes = customtypes.AttrTypesOf(AgentMemoryEndpointModel{
	EgressIPs: types.ListNull(types.StringType),
})

func agentMemoryEndpointObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: agentMemoryEndpointAttrTypes}
}

type AgentMemorySummarizationModel struct {
	Enabled         types.Bool                  `tfsdk:"enabled"`
	TriggerStrategy types.String                `tfsdk:"trigger_strategy"`
	EventCount      *AgentMemoryEventCountModel `tfsdk:"event_count"`
}

type AgentMemoryEventCountModel struct {
	Threshold   types.Int64 `tfsdk:"threshold"`
	RetainCount types.Int64 `tfsdk:"retain_count"`
}

type AgentMemoryExclusionsModel struct {
	Enabled          types.Bool                         `tfsdk:"enabled"`
	Semantic         *AgentMemorySemanticExclusionModel `tfsdk:"semantic"`
	BuiltInDetectors *AgentMemoryBuiltInDetectorsModel  `tfsdk:"built_in_detectors"`
	CustomDetectors  *AgentMemoryCustomDetectorsModel   `tfsdk:"custom_detectors"`
}

type AgentMemorySemanticExclusionModel struct {
	Enabled types.Bool   `tfsdk:"enabled"`
	Prompt  types.String `tfsdk:"prompt"`
}

type AgentMemoryBuiltInDetectorsModel struct {
	Enabled   types.Bool                          `tfsdk:"enabled"`
	Detectors []AgentMemoryDetectorSelectionModel `tfsdk:"detectors"`
}

type AgentMemoryDetectorSelectionModel struct {
	ID      types.String `tfsdk:"id"`
	Enabled types.Bool   `tfsdk:"enabled"`
	Action  types.String `tfsdk:"action"`
}

type AgentMemoryCustomDetectorsModel struct {
	Enabled   types.Bool                       `tfsdk:"enabled"`
	Detectors []AgentMemoryCustomDetectorModel `tfsdk:"detectors"`
}

type AgentMemoryCustomDetectorModel struct {
	Name    types.String                 `tfsdk:"name"`
	Enabled types.Bool                   `tfsdk:"enabled"`
	Action  types.String                 `tfsdk:"action"`
	Matcher *AgentMemoryMatcherSpecModel `tfsdk:"matcher"`
}

type AgentMemoryMatcherSpecModel struct {
	Kind  types.String               `tfsdk:"kind"`
	Regex *AgentMemoryRegexSpecModel `tfsdk:"regex"`
}

type AgentMemoryRegexSpecModel struct {
	Pattern types.String `tfsdk:"pattern"`
}

type AgentMemoryCustomMemoryTypeModel struct {
	Name               types.String                         `tfsdk:"name"`
	Description        types.String                         `tfsdk:"description"`
	Fields             []AgentMemoryCustomFieldModel        `tfsdk:"fields"`
	ExtractionStrategy *AgentMemoryCustomExtractionStrategy `tfsdk:"extraction_strategy"`
}

type AgentMemoryCustomFieldModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

type AgentMemoryCustomExtractionStrategy struct {
	Prompt  types.String `tfsdk:"prompt"`
	Enabled types.Bool   `tfsdk:"enabled"`
}
