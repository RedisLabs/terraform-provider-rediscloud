package provider

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
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
			actual := isTime()(test.input, nil)
			assert.Equal(t, test.errors, actual.HasError(), "%+v", actual)
		})
	}
}

func testAccRequiresEnvVar(t *testing.T, envVarName string) string {
	envVarValue := os.Getenv(envVarName)
	if envVarValue == "" || envVarValue == "false" {
		t.Skipf("Skipping test because %s is not set.", envVarName)
	}
	return envVarValue
}

func getTestConfig(t *testing.T, testFile string) string {
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return string(content)
}
