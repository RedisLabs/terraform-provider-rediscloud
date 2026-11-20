package main

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPatternConversion(t *testing.T) {
	for _, tc := range []struct {
		pattern, value string
		valid          bool
	}{
		{`^[^|\\u0000-\\u001f\\u007f]+$`, "tenant-1", true},
		{`^[^|\\u0000-\\u001f\\u007f]+$`, "tenant\x00", false},
		{`^[^|\\u0000-\\u001f\\u007f]+$`, "tenant|1", false},
		{`^(?:[\\w+=@-][\\w+=.@-]*|\\.[\\w+=@-][\\w+=.@-]*|\\.\\.[\\w+=.@-]+)$`, "user-1", true},
	} {
		p, err := pattern(tc.pattern)
		require.NoError(t, err)
		require.Equal(t, tc.valid, regexp.MustCompile(p).MatchString(tc.value))
	}
	_, err := pattern(`(?<=unsupported)lookbehind`)
	require.Error(t, err)
}
func TestPreparationPreservesDynamicJSON(t *testing.T) {
	shape := object{"type": "array", "items": object{"type": "object", "properties": object{"constraints": object{"description": "arbitrary JSON"}}}}
	adapted, err := safeSchema(shape)
	require.NoError(t, err)
	require.Equal(t, "string", adapted["type"])
	require.Equal(t, "array", shape["type"])
	secret, err := safeSchema(object{"type": "object", "properties": object{"key": object{"type": "string", "format": "password"}}})
	require.NoError(t, err)
	require.Equal(t, "password", secret["format"])
}
func TestResolveRejectsMissingAndRecursiveReferences(t *testing.T) {
	_, err := resolve(object{}, object{"$ref": "#/components/schemas/Missing"}, 0)
	require.ErrorContains(t, err, "missing ref")
	doc := object{"components": object{"schemas": object{"Loop": object{"$ref": "#/components/schemas/Loop"}}}}
	_, err = resolve(doc, object{"$ref": "#/components/schemas/Loop"}, 0)
	require.ErrorContains(t, err, "recursive schema")
}
