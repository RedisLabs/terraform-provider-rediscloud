package provider

import (
	rediscloud_api "github.com/RedisLabs/rediscloud-go-api"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactories = map[string]func() (*schema.Provider, error){
	"rediscloud": func() (*schema.Provider, error) {
		return New("dev")(), nil
	},
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {
	if _, ok := os.LookupEnv("REDISCLOUD_URL"); !ok {
		t.Fatal("Missing `REDISCLOUD_URL` environment variable")
	}
	if _, ok := os.LookupEnv(rediscloud_api.AccessKeyEnvVar); !ok {
		t.Fatalf("Missing `%s` environment variable", rediscloud_api.AccessKeyEnvVar)
	}
	if _, ok := os.LookupEnv(rediscloud_api.SecretKeyEnvVar); !ok {
		t.Fatalf("Missing `%s` environment variable", rediscloud_api.SecretKeyEnvVar)
	}
}
