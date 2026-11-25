package iris

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

type memoryDatabase struct {
	DatabaseID string `json:"databaseId"`
}
type memoryTTL struct {
	TTLSeconds *int64 `json:"ttlSeconds,omitempty"`
}
type memoryCadence struct {
	ActiveIntervalSeconds *int64 `json:"activeIntervalSeconds,omitempty"`
}
type modelCredentials struct {
	Type   string `json:"type"`
	APIKey string `json:"apiKey"`
}
type modelConfig struct {
	Provider    *string           `json:"provider,omitempty"`
	Model       *string           `json:"model,omitempty"`
	Credentials *modelCredentials `json:"credentials,omitempty"`
}
type longTermMemoryConfig struct {
	TTLSeconds            *int64       `json:"ttlSeconds,omitempty"`
	SessionSummaryEnabled *bool        `json:"sessionSummaryEnabled,omitempty"`
	LLM                   *modelConfig `json:"llm,omitempty"`
	Embedding             *modelConfig `json:"embedding,omitempty"`
	Classification        *modelConfig `json:"classification,omitempty"`
}
type memoryOwnership struct {
	Mode    *string `json:"mode,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	Source  *string `json:"source,omitempty"`
}
type memorySummarization struct {
	Enabled         *bool             `json:"enabled,omitempty"`
	TriggerStrategy *string           `json:"triggerStrategy,omitempty"`
	EventCount      *memoryEventCount `json:"eventCount,omitempty"`
}
type memoryEventCount struct {
	Threshold   *int64 `json:"threshold,omitempty"`
	RetainCount *int64 `json:"retainCount,omitempty"`
}
type memoryExclusions struct {
	Enabled          *bool                     `json:"enabled,omitempty"`
	Semantic         *memorySemanticExclusions `json:"semantic,omitempty"`
	BuiltInDetectors *memoryBuiltInDetectors   `json:"builtInDetectors,omitempty"`
	CustomDetectors  *memoryCustomDetectors    `json:"customDetectors,omitempty"`
}
type memorySemanticExclusions struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Prompt  *string `json:"prompt,omitempty"`
}
type memoryBuiltInDetectors struct {
	Enabled   *bool                   `json:"enabled,omitempty"`
	Detectors []memoryBuiltInDetector `json:"detectors"`
}
type memoryCustomDetectors struct {
	Enabled   *bool                  `json:"enabled,omitempty"`
	Detectors []memoryCustomDetector `json:"detectors"`
}
type memoryBuiltInDetector struct {
	ID      string  `json:"id"`
	Enabled bool    `json:"enabled"`
	Action  *string `json:"action,omitempty"`
}
type memoryCustomDetector struct {
	Name    string        `json:"name"`
	Enabled bool          `json:"enabled"`
	Action  *string       `json:"action,omitempty"`
	Matcher memoryMatcher `json:"matcher"`
}
type memoryMatcher struct {
	Kind  string      `json:"kind"`
	Regex memoryRegex `json:"regex"`
}
type memoryRegex struct {
	Pattern string `json:"pattern"`
}
type memoryField struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}
type memoryExtraction struct {
	Prompt  *string `json:"prompt,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}
type memoryCustomType struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Fields             []memoryField     `json:"fields"`
	ExtractionStrategy *memoryExtraction `json:"extractionStrategy,omitempty"`
}
type memoryFailure struct {
	Message    *string `json:"message"`
	OccurredAt *string `json:"occurredAt"`
}
type memoryEndpoint struct {
	URL        string   `json:"url"`
	Provider   string   `json:"provider"`
	Region     string   `json:"region"`
	EgressIPs  []string `json:"egressIps"`
	Accessible bool     `json:"accessible"`
}
type EndpointModel struct {
	URL        types.String `tfsdk:"url"`
	Provider   types.String `tfsdk:"provider"`
	Region     types.String `tfsdk:"region"`
	EgressIPs  types.Set    `tfsdk:"egress_ips"`
	Accessible types.Bool   `tfsdk:"accessible"`
}

var endpointElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(EndpointModel{EgressIPs: types.SetNull(types.StringType)})}

func extractionToAPI(prompt types.String, enabled types.Bool) *memoryExtraction {
	if prompt.IsNull() && enabled.IsNull() {
		return nil
	}
	return &memoryExtraction{Prompt: optionalString(prompt), Enabled: optionalBool(enabled)}
}
func memoryFieldsToAPI(ctx context.Context, v types.Set) ([]memoryField, diag.Diagnostics) {
	fields := make([]memoryField, 0)
	if v.IsNull() || v.IsUnknown() {
		return fields, nil
	}
	var models []MemoryFieldModel
	d := v.ElementsAs(ctx, &models, false)
	if d.HasError() {
		return nil, d
	}
	for _, m := range models {
		fields = append(fields, memoryField{Name: m.Name.ValueString(), Description: m.Description.ValueString(), Type: m.Type.ValueString()})
	}
	return fields, d
}
func memoryFieldsFromAPI(ctx context.Context, fields []memoryField) (types.Set, diag.Diagnostics) {
	models := make([]MemoryFieldModel, 0, len(fields))
	for _, f := range fields {
		models = append(models, MemoryFieldModel{Name: types.StringValue(f.Name), Description: types.StringValue(f.Description), Type: types.StringValue(f.Type)})
	}
	v, d := types.SetValueFrom(ctx, memoryFieldElementType, models)
	return v, d
}
func memoryTypesToAPI(ctx context.Context, v types.Set) ([]memoryCustomType, diag.Diagnostics) {
	result := make([]memoryCustomType, 0)
	var diagnostics diag.Diagnostics
	if v.IsNull() || v.IsUnknown() {
		return result, nil
	}
	var models []CustomMemoryTypeModel
	d := v.ElementsAs(ctx, &models, false)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	for _, m := range models {
		fields, d := memoryFieldsToAPI(ctx, m.Fields)
		diagnostics.Append(d...)
		result = append(result, memoryCustomType{Name: m.Name.ValueString(), Description: m.Description.ValueString(), Fields: fields, ExtractionStrategy: extractionToAPI(m.ExtractionPrompt, m.ExtractionEnabled)})
	}
	return result, diagnostics
}
