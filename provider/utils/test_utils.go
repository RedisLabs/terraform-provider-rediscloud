package utils

import (
	"os"
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
