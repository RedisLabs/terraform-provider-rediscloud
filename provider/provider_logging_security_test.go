package provider

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"

	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestSDKLoggersNeverEmitPayloadCredentials(t *testing.T) {
	const secret = "FAKE_RUNTIME_OR_MODEL_KEY_MUST_NOT_ESCAPE"
	var sdk, framework bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&sdk)
	defer log.SetOutput(previous)
	legacy := &debugLogger{}
	modern := &frameworkDebugLogger{ctx: tflogtest.RootLogger(context.Background(), &framework)}
	request := "POST /memory-stores HTTP/1.1\r\n\r\n{\"llm\":{\"credentials\":{\"apiKey\":\"" + secret + "\"}}}"
	response := "HTTP/1.1 400 Bad Request\r\n\r\n{\"message\":\"invalid credential " + secret + "\"}"
	for _, logger := range []interface {
		Printf(string, ...interface{})
		Println(...interface{})
	}{legacy, modern} {
		logger.Printf("DEBUG: Request %s:\n---[ REQUEST ]---\n%s", "/memory-stores/"+secret, request)
		logger.Printf("DEBUG: Response %s:\n---[ RESPONSE ]---\n%s", "/memory-stores/"+secret, response)
		logger.Println(fmt.Errorf("task %s failed: %s", secret, secret))
	}
	for name, output := range map[string]string{"SDK": sdk.String(), "framework": framework.String()} {
		if strings.Contains(output, secret) {
			t.Fatalf("%s logger leaked credentials", name)
		}
		if !strings.Contains(output, "HTTP POST request") || !strings.Contains(output, "400 Bad Request") {
			t.Fatalf("%s logger lost safe HTTP context: %s", name, output)
		}
	}
}

func TestProviderCredentialsSensitiveInBothSchemas(t *testing.T) {
	sdk := NewSdkProvider("test")()
	modern := NewFrameworkProvider("test")()
	var schema frameworkprovider.SchemaResponse
	modern.Schema(context.Background(), frameworkprovider.SchemaRequest{}, &schema)
	for _, field := range []string{"api_key", "secret_key"} {
		if !sdk.Schema[field].Sensitive || !schema.Schema.Attributes[field].IsSensitive() {
			t.Errorf("credential %s is not sensitive in both mux schemas", field)
		}
	}
}

func TestSDKHTTPRoundTripRedactsPayloads(t *testing.T) {
	const secret = "FAKE_REQUEST_RESPONSE_SECRET_efb2c17d"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, `{"message":"credential rejected %s","apiKey":"%s"}`, secret, secret)
	}))
	defer server.Close()
	var legacyOutput, frameworkOutput bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&legacyOutput)
	defer log.SetOutput(previous)
	loggers := []rediscloudapi.Log{&debugLogger{}, &frameworkDebugLogger{ctx: tflogtest.RootLogger(context.Background(), &frameworkOutput)}}
	for _, logger := range loggers {
		api, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth(secret, secret), rediscloudapi.LogRequests(true), rediscloudapi.Logger(logger))
		if err != nil {
			t.Fatal("could not create mock API client")
		}
		_, err = api.AgentMemory.Create(context.Background(), agentmemoryapi.CreateStore{Name: secret, DatabaseID: 123})
		if err == nil {
			t.Fatal("injected HTTP error was ignored")
		}
		if strings.Contains(client.SafeAPIError(err), secret) || !strings.Contains(client.SafeAPIError(err), "HTTP 400") {
			t.Fatal("error lost safe status or leaked credential")
		}
	}
	for label, output := range map[string]string{"legacy": legacyOutput.String(), "framework": frameworkOutput.String()} {
		if strings.Contains(output, secret) {
			t.Fatalf("%s actual SDK round trip leaked payload", label)
		}
		if !strings.Contains(output, "HTTP POST request") || !strings.Contains(output, "400 Bad Request") {
			t.Fatalf("%s actual SDK round trip lost HTTP context", label)
		}
	}
}
