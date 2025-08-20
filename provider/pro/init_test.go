package pro

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"testing"
)

var testProvider *schema.Provider
var providerFactories map[string]func() (*schema.Provider, error)

func init() {
	testProvider = New("dev")()
	providerFactories = map[string]func() (*schema.Provider, error){
		"rediscloud": func() (*schema.Provider, error) {
			return testProvider, nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
