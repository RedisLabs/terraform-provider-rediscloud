package testhelpers

import (
	"os"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
)

// RequireEnvVars skips or fails the test if any of the named environment
// variables are not set.
func RequireEnvVars(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

// BasicPreCheck checks that the minimum provider configuration (URL, access key
// and secret key) are present. Use RequireEnvVars directly when additional
// variables are needed.
func BasicPreCheck(t *testing.T) {
	t.Helper()
	RequireEnvVars(t, rediscloudApi.RedisCloudUrlEnvVar, rediscloudApi.AccessKeyEnvVar, rediscloudApi.SecretKeyEnvVar)
}
