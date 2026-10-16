package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func sdkLoggerForTest(t *testing.T, kind string, output *bytes.Buffer) rediscloudapi.Log {
	t.Helper()
	if kind == "framework" {
		return &frameworkDebugLogger{ctx: tflogtest.RootLogger(context.Background(), output)}
	}
	previous := log.Writer()
	log.SetOutput(output)
	t.Cleanup(func() { log.SetOutput(previous) })
	return &debugLogger{}
}

func TestSDKLoggingPreservesSafeContext(t *testing.T) {
	const secret = "FAKE_LOG_SECRET_NEVER_ECHO"
	for _, kind := range []string{"legacy", "framework"} {
		for _, tc := range []struct{ name, message, want string }{
			{"request", "DEBUG: Request /" + secret + ":\nPOST /memory-stores HTTP/1.1\r\nAuthorization: " + secret + "\r\n\r\n{\"apiKey\":\"" + secret + "\"}", "HTTP POST request"},
			{"response", "DEBUG: Response /" + secret + ":\nHTTP/1.1 403 " + secret + "\r\nSet-Cookie: " + secret + "\r\n\r\n" + secret, "403 Forbidden"},
			{"wait", "Waiting for Agent Memory store " + secret + " to finish being updated", "operation to complete"},
			{"progress", "task " + secret + " not processed yet: " + secret, "task is still processing"},
			{"rate-limit", "status code 429 received, request will be retried " + secret, "rate limit"},
			{"unknown", "unexpected SDK event " + secret, "details were omitted"},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				var output bytes.Buffer
				logger := sdkLoggerForTest(t, kind, &output)
				logger.Printf("%s", tc.message)
				logger.Println(tc.message)
				if strings.Contains(output.String(), secret) {
					t.Error("SDK logger exposed sensitive payload content")
				}
				if !strings.Contains(output.String(), tc.want) {
					t.Errorf("SDK logger lost safe context %q", tc.want)
				}
			})
		}
	}
}

func TestSDKLoggingActualPayloads(t *testing.T) {
	const secret = "FAKE_ACTUAL_MODEL_OR_RUNTIME_SECRET"
	for _, kind := range []string{"legacy", "framework"} {
		for _, scenario := range []string{"llm-request", "generated-key-response", "error-response"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				var output bytes.Buffer
				logger := sdkLoggerForTest(t, kind, &output)
				var mu sync.Mutex
				created := false
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					switch {
					case scenario == "error-response":
						w.WriteHeader(http.StatusBadRequest)
						_, _ = fmt.Fprintf(w, `{"message":"backend rejected %s"}`, secret)
					case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
						var body agentmemoryapi.CreateStore
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.LLM == nil || body.LLM.Credentials == nil || body.LLM.Credentials.APIKey != secret {
							t.Error("secret did not reach its intended API request")
						}
						_, _ = w.Write([]byte(`{"taskId":"task-1"}`))
					case r.Method == http.MethodGet && r.URL.Path == "/tasks/task-1":
						_, _ = w.Write([]byte(`{"status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
					case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1/api-keys":
						if created {
							_, _ = w.Write([]byte(`{"apiKeys":[{"apiKeyId":"key-1","keyName":"runtime"}]}`))
						} else {
							_, _ = w.Write([]byte(`{"apiKeys":[]}`))
						}
					case r.Method == http.MethodPost && r.URL.Path == "/memory-stores/store-1/api-keys":
						created = true
						_, _ = fmt.Fprintf(w, `{"apiKey":%q}`, secret)
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				api, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"), rediscloudapi.LogRequests(true), rediscloudapi.Logger(logger))
				if err != nil {
					t.Fatal("failed to initialise local SDK client")
				}
				if scenario == "generated-key-response" {
					key, err := api.AgentMemory.CreateAPIKey(context.Background(), "store-1", "runtime")
					if err != nil || key == nil || key.APIKey != secret {
						t.Fatal("log protection changed the returned key secret")
					}
				} else {
					id, err := api.AgentMemory.Create(context.Background(), agentmemoryapi.CreateStore{Name: "logger-test", DatabaseID: 123, LLM: &agentmemoryapi.ModelConfig{Provider: "fake-provider", Model: "fake-model", Credentials: &agentmemoryapi.ModelCredentials{Type: "apiKey", APIKey: secret}}})
					if scenario == "error-response" {
						if err == nil {
							t.Fatal("log protection swallowed an API error")
						}
					} else if err != nil || id != "store-1" {
						t.Fatal("log protection changed store creation")
					}
				}
				if strings.Contains(output.String(), secret) {
					t.Error("real SDK logging exposed model credentials or a returned secret")
				}
				wantStatus := "200 OK"
				if scenario == "error-response" {
					wantStatus = "400 Bad Request"
				}
				if !strings.Contains(output.String(), "HTTP POST request") || !strings.Contains(output.String(), wantStatus) {
					t.Error("real SDK logging lost HTTP method or response status")
				}
			})
		}
	}
}
