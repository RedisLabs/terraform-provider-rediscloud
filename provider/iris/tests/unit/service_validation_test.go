package unit_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestIrisDefersServiceOwnedValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("planning must not call the service") }))
	defer server.Close()
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{{
		ProtoV5ProviderFactories: factories(t),
		ConfigFile:               config.StaticFile("testdata/service_values.tf"),
		ConfigVariables:          config.Variables{"memory_url": config.StringVariable(server.URL), "identity_url": config.StringVariable(server.URL), "name": config.StringVariable(strings.Repeat("future.", 100) + ":id")},
		PlanOnly:                 true, ExpectNonEmptyPlan: true,
	}}})
}
