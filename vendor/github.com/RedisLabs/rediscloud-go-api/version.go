package rediscloud_api

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	// Version is the release number of this SDK.
	Version = "0.1.0"

	// AccessKeyEnvVar is the environment variable that will be used for the access key by default.
	AccessKeyEnvVar = "REDISCLOUD_ACCESS_KEY"

	// SecretKeyEnvVar is the environment variable that will be used for the secret key by default.
	SecretKeyEnvVar = "REDISCLOUD_SECRET_KEY"
)

var userAgent = buildUserAgent("rediscloud-go-api", Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

func buildUserAgent(name string, version string, info ...string) string {
	product := fmt.Sprintf("%s/%s", name, version)
	systemInfo := strings.Join(info, "; ")
	return fmt.Sprintf("%s (%s)", product, systemInfo)
}
