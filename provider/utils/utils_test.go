package utils

import (
	"errors"
	"fmt"
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

func TestIsTransientSubscriptionGetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"500 from go-api", errors.New("failed to get subscription 123: 500 - internal server error"), true},
		{"500 wrapped", fmt.Errorf("wrapped: %w", errors.New("failed to get subscription 123: 500 - boom")), true},
		{"502 bad gateway", errors.New("failed to get subscription 123: 502 - bad gateway"), true},
		{"503 unavailable", errors.New("failed to get subscription 123: 503 - unavailable"), true},
		{"504 gateway timeout", errors.New("failed to get subscription 123: 504 - gateway timeout"), true},
		{"404 not found", errors.New("failed to get subscription 123: 404 - not found"), false},
		{"429 rate limit", errors.New("failed to get subscription 123: 429 - too many requests"), false},
		{"501 not implemented", errors.New("failed to get subscription 123: 501 - not implemented"), false},
		{"unrelated error", errors.New("context deadline exceeded"), false},
		{"body contains 500 without code", errors.New("failed to get subscription 123: 400 - value 500 invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsTransientSubscriptionGetError(tt.err))
		})
	}
}
