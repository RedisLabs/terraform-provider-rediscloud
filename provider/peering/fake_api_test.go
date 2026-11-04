package peering

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func newFakeAPIClient(t *testing.T, routes map[string]string) *client.ApiClient {
	t.Helper()
	return peeringAPIClient(t, routes, "")
}

type peeringTestTransport func(*http.Request) (*http.Response, error)

func (f peeringTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func peeringAPIClient(t *testing.T, routes map[string]string, failAfterFirst string) *client.ApiClient {
	t.Helper()

	first := true
	transport := peeringTestTransport(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			defer func() { _ = req.Body.Close() }()
		}
		route := req.Method + " " + req.URL.Path
		body, ok := routes[route]
		recorder := httptest.NewRecorder()
		recorder.Header().Set("Content-Type", "application/json")
		switch {
		case !ok:
			t.Errorf("unexpected request: %s", route)
			recorder.WriteHeader(http.StatusNotFound)
		case route == failAfterFirst && !first:
			recorder.WriteHeader(http.StatusInternalServerError)
			_, _ = recorder.WriteString(`{"description":"final refresh failed"}`)
		default:
			if route == failAfterFirst {
				first = false
			}
			_, _ = recorder.WriteString(body)
		}
		return recorder.Result(), nil
	})

	sdk, err := rediscloudapi.NewClient(
		rediscloudapi.BaseURL("https://peering.example"),
		rediscloudapi.Auth("fake-key", "fake-secret"),
		rediscloudapi.Transporter(transport),
	)
	require.NoError(t, err)
	return &client.ApiClient{Client: sdk, WaitDelayOverride: time.Millisecond, WaitPollIntervalOverride: time.Millisecond}
}

func assertAcknowledgedPeeringState(t *testing.T, ctx context.Context, state tfsdk.State, status string) {
	t.Helper()
	var id, gotStatus types.String
	require.False(t, state.GetAttribute(ctx, path.Root("id"), &id).HasError())
	require.False(t, state.GetAttribute(ctx, path.Root("status"), &gotStatus).HasError())
	assert.Equal(t, types.StringValue("1/2"), id)
	assert.Equal(t, types.StringValue(status), gotStatus)
	assert.True(t, state.Raw.IsFullyKnown())
}

func nullTimeouts(s resource.SchemaResponse) timeouts.Value {
	return timeouts.Value{Object: types.ObjectNull(s.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)}
}

// assertStateKeepsPlannedValues mirrors the Terraform core check that rejects
// an apply whose state differs from a known planned value.
func assertStateKeepsPlannedValues(t *testing.T, plan tfsdk.Plan, state tfsdk.State) {
	t.Helper()
	var planned, applied map[string]tftypes.Value
	require.NoError(t, plan.Raw.As(&planned))
	require.NoError(t, state.Raw.As(&applied))
	for name, value := range planned {
		if value.IsFullyKnown() {
			assert.True(t, value.Equal(applied[name]), "%s: planned %s, applied %s", name, value, applied[name])
		}
	}
}

// assertNullAttributes checks which attributes are null, as SDKv2 2.20.0 stored them.
func assertNullAttributes(t *testing.T, state tfsdk.State, null []string) {
	t.Helper()
	var applied map[string]tftypes.Value
	require.NoError(t, state.Raw.As(&applied))
	want := map[string]bool{"timeouts": true}
	for _, name := range null {
		want[name] = true
	}
	for name, value := range applied {
		assert.Equal(t, want[name], value.IsNull(), "%s null", name)
	}
}

func importAndRead(t *testing.T, ctx context.Context, r resource.ResourceWithImportState, s resource.SchemaResponse, id string) tfsdk.State {
	t.Helper()
	imported := resource.ImportStateResponse{State: tfsdk.State{Schema: s.Schema, Raw: tftypes.NewValue(s.Schema.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: id}, &imported)
	require.False(t, imported.Diagnostics.HasError(), imported.Diagnostics)

	read := resource.ReadResponse{State: imported.State}
	r.Read(ctx, resource.ReadRequest{State: imported.State}, &read)
	require.False(t, read.Diagnostics.HasError(), read.Diagnostics)
	return read.State
}
