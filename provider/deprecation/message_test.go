package deprecation

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func TestMessage(t *testing.T) {
	got := Message("memory_limit_in_gb", "`dataset_size_in_gb`")
	want := "`memory_limit_in_gb` is deprecated and will be removed in the next major version of the provider. Use `dataset_size_in_gb` instead."
	if got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}
}

func TestWhenNonEmpty(t *testing.T) {
	validate := WhenNonEmpty("client_ssl_certificate", "`client_tls_certificates`")

	if diags := validate("", cty.Path{}); len(diags) != 0 {
		t.Fatalf("empty value returned %d diagnostics, want 0", len(diags))
	}
	if diags := validate(0, cty.Path{}); len(diags) != 0 {
		t.Fatalf("non-string value returned %d diagnostics, want 0", len(diags))
	}

	diags := validate("-----BEGIN CERTIFICATE-----", cty.Path{})
	if len(diags) != 1 {
		t.Fatalf("set value returned %d diagnostics, want 1", len(diags))
	}
	if diags[0].Severity != diag.Warning {
		t.Fatalf("severity = %v, want warning", diags[0].Severity)
	}
	if diags[0].Detail != Message("client_ssl_certificate", "`client_tls_certificates`") {
		t.Fatalf("detail = %q", diags[0].Detail)
	}
}
