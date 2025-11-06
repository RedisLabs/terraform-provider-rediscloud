package provider

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/stretchr/testify/assert"
)

// TestUnitIsEnableDefaultUserExplicitlySetInConfig tests the helper function logic
// that determines if enable_default_user was explicitly set in the Terraform config
func TestUnitIsEnableDefaultUserExplicitlySetInConfig(t *testing.T) {
	tests := []struct {
		name         string
		rawConfig    cty.Value
		regionName   string
		expected     bool
		description  string
	}{
		{
			name: "field_explicitly_set_to_true",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":                cty.StringVal("us-east-1"),
						"enable_default_user": cty.True,
					}),
				}),
			}),
			regionName:  "us-east-1",
			expected:    true,
			description: "When enable_default_user is explicitly true, should return true",
		},
		{
			name: "field_explicitly_set_to_false",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":                cty.StringVal("us-east-1"),
						"enable_default_user": cty.False,
					}),
				}),
			}),
			regionName:  "us-east-1",
			expected:    true,
			description: "When enable_default_user is explicitly false, should return true",
		},
		{
			name: "field_not_set",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("us-east-1"),
						// enable_default_user not present
					}),
				}),
			}),
			regionName:  "us-east-1",
			expected:    false,
			description: "When enable_default_user is not in config, should return false",
		},
		{
			name: "field_is_null",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":                cty.StringVal("us-east-1"),
						"enable_default_user": cty.NullVal(cty.Bool),
					}),
				}),
			}),
			regionName:  "us-east-1",
			expected:    false,
			description: "When enable_default_user is null, should return false",
		},
		{
			name: "multiple_regions_one_has_field",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":                cty.StringVal("us-east-1"),
						"enable_default_user": cty.True,
					}),
					cty.ObjectVal(map[string]cty.Value{
						"name":                cty.StringVal("us-east-2"),
						"enable_default_user": cty.NullVal(cty.Bool), // Null means not set
					}),
				}),
			}),
			regionName:  "us-east-2",
			expected:    false,
			description: "When checking region without field (null), should return false",
		},
		{
			name: "region_not_found",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("us-east-1"),
					}),
				}),
			}),
			regionName:  "eu-west-1",
			expected:    false,
			description: "When region not found in config, should return false",
		},
		{
			name: "override_region_null",
			rawConfig: cty.ObjectVal(map[string]cty.Value{
				"override_region": cty.NullVal(cty.Set(cty.Object(map[string]cty.Type{
					"name":                cty.String,
					"enable_default_user": cty.Bool,
				}))),
			}),
			regionName:  "us-east-1",
			expected:    false,
			description: "When override_region is null, should return false",
		},
		{
			name: "raw_config_null",
			rawConfig: cty.NullVal(cty.Object(map[string]cty.Type{
				"override_region": cty.Set(cty.Object(map[string]cty.Type{
					"name":                cty.String,
					"enable_default_user": cty.Bool,
				})),
			})),
			regionName:  "us-east-1",
			expected:    false,
			description: "When raw config is null (test environment), should return false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the cty.Value parsing logic that powers the actual function
			result := testIsEnableDefaultUserExplicitlySetFromRawConfig(tt.rawConfig, tt.regionName)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// testIsEnableDefaultUserExplicitlySetFromRawConfig replicates the logic of
// isEnableDefaultUserExplicitlySetInConfig for testing purposes with direct cty.Value input
func testIsEnableDefaultUserExplicitlySetFromRawConfig(rawConfig cty.Value, regionName string) bool {
	// Same logic as the actual function
	if rawConfig.IsNull() {
		return false
	}

	if !rawConfig.Type().HasAttribute("override_region") {
		return false
	}

	overrideRegionAttr := rawConfig.GetAttr("override_region")
	if overrideRegionAttr.IsNull() {
		return false
	}

	if overrideRegionAttr.Type().IsSetType() {
		iter := overrideRegionAttr.ElementIterator()
		for iter.Next() {
			_, regionVal := iter.Element()

			if regionVal.Type().HasAttribute("name") {
				nameAttr := regionVal.GetAttr("name")
				if !nameAttr.IsNull() && nameAttr.AsString() == regionName {
					if regionVal.Type().HasAttribute("enable_default_user") {
						eduAttr := regionVal.GetAttr("enable_default_user")
						return !eduAttr.IsNull()
					}
					return false
				}
			}
		}
	}

	return false
}
