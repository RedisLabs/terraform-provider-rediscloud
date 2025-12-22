package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTime(t *testing.T) {
	tests := []struct {
		input  string
		errors bool
	}{
		{"0:00", false},
		{"09:00", false},
		{"12:00", false},
		{"24:00", true},    // '24' isn't a valid hour
		{"12:00:00", true}, // seconds are invalid
		{"blah", true},     // Not a valid time
		{"", true},         // Nothing
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			actual := IsTime()(test.input, nil)
			assert.Equal(t, test.errors, actual.HasError(), "%+v", actual)
		})
	}
}
