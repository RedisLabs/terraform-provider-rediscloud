package utils

import (
	"os"
	"strings"
	"testing"
)

func AccRequiresEnvVar(t *testing.T, envVarName string) string {
	envVarValue := os.Getenv(envVarName)
	if envVarValue == "" || envVarValue == "false" {
		t.Skipf("Skipping test because %s is not set.", envVarName)
	}
	return envVarValue
}

func GetTestConfig(t *testing.T, testFile string) string {
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	return string(content)
}

// RenderTestConfig loads a test fixture and replaces named placeholders with values.
// Placeholders in fixtures use the format __PLACEHOLDER_NAME__ (e.g., __CLOUD_ACCOUNT__).
// This allows fixtures to be valid HCL that can be formatted with terraform fmt.
//
// For booleans, use "__PLACEHOLDER__" == "true" pattern in fixtures and pass "true"/"false" strings.
// For numbers, use tonumber("__PLACEHOLDER__") pattern in fixtures and pass number as string.
func RenderTestConfig(t *testing.T, testFile string, vars map[string]string) string {
	config := GetTestConfig(t, testFile)
	for placeholder, value := range vars {
		config = strings.ReplaceAll(config, placeholder, value)
	}
	return config
}
