package agentmemory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestReviewAPIKeyDeleteFailureHandling(t *testing.T) {
	for _, code := range []int{404, 403, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" || r.URL.Path != "/memory-stores/store-1/api-keys/key-1" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"message":"parent store missing or unavailable","error":"review"}`))
			}))
			defer server.Close()
			t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
			t.Setenv(rediscloudapi.AccessKeyEnvVar, "test-api-key")
			t.Setenv(rediscloudapi.SecretKeyEnvVar, "test-secret-key")
			c, err := client.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			r := &agentMemoryAPIKeyResource{client: c}
			ctx := context.Background()
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			state := tfsdk.State{Schema: schema.Schema}
			diags := state.Set(ctx, &AgentMemoryAPIKeyResourceModel{ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime"), APIKey: types.StringNull(), ObfuscatedToken: types.StringNull(), CreatedAt: types.Int64Null()})
			if diags.HasError() {
				t.Fatal(diags)
			}
			response := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
			if got := response.Diagnostics.HasError(); got != (code != 404) {
				t.Fatalf("status=%d diagnostics=%v", code, response.Diagnostics)
			}
		})
	}
}

func TestReviewAPIErrorSecretRedaction(t *testing.T) {
	const sentinel = "FAKE_REVIEW_SECRET_MUST_NOT_APPEAR"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"message":"invalid credential ` + sentinel + `","error":"credential rejected"}`))
	}))
	defer server.Close()
	t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
	t.Setenv(rediscloudapi.AccessKeyEnvVar, "test-api-key")
	t.Setenv(rediscloudapi.SecretKeyEnvVar, sentinel)
	c, err := client.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	r := &agentMemoryAPIKeyResource{client: c}
	ctx := context.Background()
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	state := tfsdk.State{Schema: schema.Schema}
	diags := state.Set(ctx, &AgentMemoryAPIKeyResourceModel{ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime"), APIKey: types.StringValue(sentinel), ObfuscatedToken: types.StringNull(), CreatedAt: types.Int64Null()})
	if diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("expected actionable API error")
	}
	for _, d := range response.Diagnostics {
		if strings.Contains(d.Summary()+d.Detail(), sentinel) {
			t.Error("CONFIRMED: fake credential echoed by API reaches Terraform diagnostic without redaction")
		}
	}
}
