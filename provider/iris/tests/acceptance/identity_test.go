package acceptance

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func newIdentityClient(t *testing.T) *client.IrisServiceClient {
	t.Helper()
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("live acceptance requires TF_ACC=1")
	}
	if os.Getenv("REDISCLOUD_IRIS_IDENTITY_URL") == "" {
		t.Skip("requires REDISCLOUD_IRIS_IDENTITY_URL")
	}
	api, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: os.Getenv("REDISCLOUD_IRIS_IDENTITY_URL"), Token: os.Getenv("REDISCLOUD_IRIS_IDENTITY_TOKEN"), APIKey: os.Getenv("REDISCLOUD_ACCESS_KEY"), SecretKey: os.Getenv("REDISCLOUD_SECRET_KEY")})
	if err != nil {
		t.Fatal(err)
	}
	return api
}

func identityAbsent(api *client.IrisServiceClient, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := api.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if client.IrisNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("identity test resource still exists at %s", path)
}

func cleanupIdentity(t *testing.T, api *client.IrisServiceClient, path string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := api.Do(ctx, http.MethodDelete, path, nil, nil, nil); err != nil && !client.IrisNotFound(err) {
		t.Errorf("cleanup Identity test resource: %v", err)
	}
}
