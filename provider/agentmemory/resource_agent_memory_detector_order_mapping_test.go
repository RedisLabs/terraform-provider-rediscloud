package agentmemory

import (
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestOrderDetectorSelectionsLikeExisting(t *testing.T) {
	priorCard := AgentMemoryDetectorSelectionModel{ID: types.StringValue("credit-card"), Enabled: types.BoolValue(true), Action: types.StringValue("redact")}
	priorEmail := AgentMemoryDetectorSelectionModel{ID: types.StringValue("email"), Enabled: types.BoolValue(false), Action: types.StringValue("redact")}
	apiCard := AgentMemoryDetectorSelectionModel{ID: types.StringValue("credit-card"), Enabled: types.BoolValue(false), Action: types.StringValue("drop")}
	apiEmail := AgentMemoryDetectorSelectionModel{ID: types.StringValue("email"), Enabled: types.BoolValue(true), Action: types.StringValue("mask")}
	apiPhone := AgentMemoryDetectorSelectionModel{ID: types.StringValue("phone"), Enabled: types.BoolValue(true), Action: types.StringValue("drop")}
	unknownID, nullID, blankID := apiCard, apiCard, apiCard
	unknownID.ID = types.StringUnknown()
	nullID.ID = types.StringNull()
	blankID.ID = types.StringValue("")

	for _, tc := range []struct {
		name             string
		models, existing []AgentMemoryDetectorSelectionModel
		want             []AgentMemoryDetectorSelectionModel
	}{
		{
			name:     "prior order with current API values",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiCard, apiEmail},
		},
		{
			name:     "API-only selection is retained",
			models:   []AgentMemoryDetectorSelectionModel{apiPhone, apiEmail, apiCard},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiCard, apiEmail, apiPhone},
		},
		{
			name:     "remote removal is retained",
			models:   []AgentMemoryDetectorSelectionModel{apiPhone, apiEmail},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, apiPhone},
		},
		{
			name:   "no prior order during import",
			models: []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
			want:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
		},
		{
			name:     "nil API list does not restore prior selections",
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
		},
		{
			name:     "empty API list remains empty",
			models:   []AgentMemoryDetectorSelectionModel{},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{},
		},
		{
			name:     "duplicate API ID preserves all response entries",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard, priorCard},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, apiCard, priorCard},
		},
		{
			name:     "unknown API ID preserves response order",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, unknownID},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, unknownID},
		},
		{
			name:     "null API ID preserves response order",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, nullID},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, nullID},
		},
		{
			name:     "blank API ID preserves response order",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, blankID},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, blankID},
		},
		{
			name:     "duplicate prior ID is ambiguous",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
			existing: []AgentMemoryDetectorSelectionModel{priorCard, priorCard, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
		},
		{
			name:     "unknown prior ID is ambiguous",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
			existing: []AgentMemoryDetectorSelectionModel{unknownID, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
		},
		{
			name:     "null prior ID is ambiguous",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
			existing: []AgentMemoryDetectorSelectionModel{nullID, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
		},
		{
			name:     "blank prior ID is ambiguous",
			models:   []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
			existing: []AgentMemoryDetectorSelectionModel{blankID, priorEmail},
			want:     []AgentMemoryDetectorSelectionModel{apiEmail, apiCard},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalModels, originalExisting := slices.Clone(tc.models), slices.Clone(tc.existing)
			got := orderDetectorSelectionsLikeExisting(tc.models, tc.existing)
			require.Equal(t, tc.want, got)
			require.Equal(t, originalModels, tc.models, "response order must not be mutated")
			require.Equal(t, originalExisting, tc.existing, "prior state must not be mutated")
		})
	}
}

func TestOrderCustomDetectorsLikeExisting(t *testing.T) {
	priorAccount := detectorOrderCustomModel(t, "account", true, "redact", "ACCT-[0-9]+")
	priorTenant := detectorOrderCustomModel(t, "tenant", false, "redact", "TENANT-[0-9]+")
	apiAccount := detectorOrderCustomModel(t, "account", false, "drop", "ACCOUNT-[0-9]+")
	apiTenant := detectorOrderCustomModel(t, "tenant", true, "mask", "TENANT-[A-Z]+")
	apiProject := detectorOrderCustomModel(t, "project", true, "drop", "PROJECT-[0-9]+")
	unknownName, nullName, blankName := apiAccount, apiAccount, apiAccount
	unknownName.Name = types.StringUnknown()
	nullName.Name = types.StringNull()
	blankName.Name = types.StringValue("")

	for _, tc := range []struct {
		name             string
		models, existing []AgentMemoryCustomDetectorModel
		want             []AgentMemoryCustomDetectorModel
	}{
		{
			name:     "prior order with current API values and matchers",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiAccount, apiTenant},
		},
		{
			name:     "API-only detector is retained",
			models:   []AgentMemoryCustomDetectorModel{apiProject, apiTenant, apiAccount},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiAccount, apiTenant, apiProject},
		},
		{
			name:     "remote removal is retained",
			models:   []AgentMemoryCustomDetectorModel{apiProject, apiTenant},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, apiProject},
		},
		{
			name:   "no prior order during import",
			models: []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
			want:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
		},
		{
			name:     "nil API list does not restore prior detectors",
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
		},
		{
			name:     "empty API list remains empty",
			models:   []AgentMemoryCustomDetectorModel{},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{},
		},
		{
			name:     "duplicate API name preserves all response entries",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount, priorAccount},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, apiAccount, priorAccount},
		},
		{
			name:     "unknown API name preserves response order",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, unknownName},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, unknownName},
		},
		{
			name:     "null API name preserves response order",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, nullName},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, nullName},
		},
		{
			name:     "new empty API name is retained",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, blankName},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, blankName},
		},

		{
			name:     "known empty name retains configured order",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, blankName},
			existing: []AgentMemoryCustomDetectorModel{blankName, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{blankName, apiTenant},
		},
		{
			name:     "API-only empty name follows configured detectors",
			models:   []AgentMemoryCustomDetectorModel{blankName, apiAccount, apiTenant},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiAccount, apiTenant, blankName},
		},
		{
			name:     "duplicate empty API names remain visible",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, blankName, blankName},
			existing: []AgentMemoryCustomDetectorModel{blankName, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, blankName, blankName},
		},
		{
			name:     "duplicate empty prior names are ambiguous",
			models:   []AgentMemoryCustomDetectorModel{blankName, apiTenant},
			existing: []AgentMemoryCustomDetectorModel{blankName, blankName, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{blankName, apiTenant},
		},
		{
			name:     "duplicate prior name is ambiguous",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
			existing: []AgentMemoryCustomDetectorModel{priorAccount, priorAccount, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
		},
		{
			name:     "unknown prior name is ambiguous",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
			existing: []AgentMemoryCustomDetectorModel{unknownName, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
		},
		{
			name:     "null prior name is ambiguous",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
			existing: []AgentMemoryCustomDetectorModel{nullName, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
		},
		{
			name:     "removed empty prior name is not restored",
			models:   []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
			existing: []AgentMemoryCustomDetectorModel{blankName, priorTenant},
			want:     []AgentMemoryCustomDetectorModel{apiTenant, apiAccount},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalModels, originalExisting := slices.Clone(tc.models), slices.Clone(tc.existing)
			got := orderCustomDetectorsLikeExisting(tc.models, tc.existing)
			require.Equal(t, tc.want, got)
			require.Equal(t, originalModels, tc.models, "response order must not be mutated")
			require.Equal(t, originalExisting, tc.existing, "prior state must not be mutated")
		})
	}
}

func detectorOrderCustomModel(t *testing.T, name string, enabled bool, action, pattern string) AgentMemoryCustomDetectorModel {
	return AgentMemoryCustomDetectorModel{
		Name:    types.StringValue(name),
		Enabled: types.BoolValue(enabled),
		Action:  types.StringValue(action),
		Matcher: mustMatcherSpecList(t, &AgentMemoryMatcherSpecModel{
			Kind: types.StringValue("regex"),
			Regex: mustRegexSpecList(t, &AgentMemoryRegexSpecModel{
				Pattern: types.StringValue(pattern),
			}),
		}),
	}
}
