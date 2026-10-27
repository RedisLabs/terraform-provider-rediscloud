package unit_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/config"
	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/agentmemory"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

type deletionFixture struct {
	api                            *testhelpers.FakeAPI
	remainingReads                 int
	creates, deletes, reads        int
	tasks                          int
	deleteHTTP, taskHTTP, readHTTP int
	polling, gone                  bool
	deadline                       time.Time
	changedDeadline                bool
	responseID                     string
	serviceStatus                  string
	taskPending                    bool
	cancelOnRead                   context.CancelFunc
}

func newDeletionFixture(remainingReads int) *deletionFixture {
	f := &deletionFixture{api: testhelpers.NewFakeAPI(), remainingReads: remainingReads, responseID: "store-1", serviceStatus: api.StatusReady}
	f.api.Handle("POST /memory-stores", func(*http.Request) (any, error) {
		f.creates++
		return map[string]any{"taskId": "create-task"}, nil
	})
	f.api.Handle("GET /tasks/create-task", func(*http.Request) (any, error) {
		return map[string]any{"status": "processing-completed", "response": map[string]any{"resource": map[string]any{"storeId": "store-1"}}}, nil
	})
	f.api.Handle("DELETE /memory-stores/store-1", func(req *http.Request) (any, error) {
		f.deletes++
		f.deadline, _ = req.Context().Deadline()
		if f.deleteHTTP != 0 {
			return nil, deletionHTTPError(f.deleteHTTP)
		}
		f.polling = true
		return map[string]any{"taskId": "delete-task"}, nil
	})
	f.api.Handle("GET /tasks/delete-task", func(req *http.Request) (any, error) {
		f.tasks++
		f.checkDeadline(req)
		if f.taskHTTP != 0 {
			return nil, deletionHTTPError(f.taskHTTP)
		}
		if f.taskPending {
			return map[string]any{"status": "processing-in-progress"}, nil
		}
		return map[string]any{"status": "processing-completed"}, nil
	})
	f.api.Handle("GET /memory-stores/store-1", func(req *http.Request) (any, error) {
		if f.gone {
			return nil, testhelpers.NotFound()
		}
		if f.polling {
			f.reads++
			f.checkDeadline(req)
			if f.cancelOnRead != nil {
				f.cancelOnRead()
				return nil, req.Context().Err()
			}
			if f.readHTTP != 0 {
				f.polling = false
				return nil, deletionHTTPError(f.readHTTP)
			}
			if f.remainingReads == 0 {
				f.gone = true
				return nil, testhelpers.NotFound()
			}
			if f.remainingReads > 0 {
				f.remainingReads--
			}
		}
		return &api.Store{
			StoreID: redis.String(f.responseID), Name: redis.String("store"), DatabaseID: redis.String("123"),
			Status: redis.String(f.serviceStatus), ShortMemory: &api.ShortMemoryConfig{TTLSeconds: redis.Int(86400)},
			LongTermMemory:     &api.LongTermMemoryStatus{TTLSeconds: redis.Int(31536000)},
			ExtractionStrategy: redis.String("INSTRUCT"), ExtractionCadence: &api.ExtractionCadenceConfig{ActiveIntervalSeconds: redis.Int(300)},
		}, nil
	})
	return f
}

func (f *deletionFixture) checkDeadline(req *http.Request) {
	deadline, _ := req.Context().Deadline()
	if f.deadline.IsZero() || !deadline.Equal(f.deadline) {
		f.changedDeadline = true
	}
}

func deletionHTTPError(status int) error {
	return testhelpers.StatusError{Code: status, Body: map[string]any{"description": "fixture deletion failure"}}
}

func TestAgentMemoryDeleteConfirmsDisappearance(t *testing.T) {
	for _, remaining := range []int{0, 2} {
		t.Run(fmt.Sprintf("visible-reads-%d", remaining), func(t *testing.T) {
			f := newDeletionFixture(remaining)
			if remaining > 0 {
				f.serviceStatus = api.StatusUnavailable
			}
			r, _, state := configuredAgentMemoryResource(t, testhelpers.NewAPIClient(t, f.api), "500ms")
			resp := resource.DeleteResponse{State: state}
			r.Delete(t.Context(), resource.DeleteRequest{State: state}, &resp)
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			f.api.WithHandlersPaused(func() {
				assert.True(t, f.gone)
				assert.Equal(t, remaining+1, f.reads)
				assert.Equal(t, 1, f.deletes)
				assert.Equal(t, 1, f.tasks)
				assert.False(t, f.changedDeadline)
			})
		})
	}
}

func TestAgentMemoryDeleteFailureKeepsState(t *testing.T) {
	cases := []struct {
		name                           string
		deleteHTTP, taskHTTP, readHTTP int
		remaining                      int
		responseID                     string
		cancel                         bool
		pending                        bool
	}{
		{name: "initial-not-found", deleteHTTP: http.StatusNotFound},
		{name: "initial-forbidden", deleteHTTP: http.StatusForbidden},
		{name: "initial-server-error", deleteHTTP: http.StatusInternalServerError},
		{name: "task-error", taskHTTP: http.StatusInternalServerError},
		{name: "task-timeout", pending: true},
		{name: "poll-error", readHTTP: http.StatusInternalServerError},
		{name: "poll-forbidden", readHTTP: http.StatusForbidden},
		{name: "timeout", remaining: -1},
		{name: "wrong-store", remaining: 1, responseID: "unrelated-store"},
		{name: "cancelled", remaining: 1, cancel: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDeletionFixture(tc.remaining)
			f.deleteHTTP, f.taskHTTP, f.readHTTP = tc.deleteHTTP, tc.taskHTTP, tc.readHTTP
			f.taskPending = tc.pending
			if tc.responseID != "" {
				f.responseID = tc.responseID
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.cancel {
				f.cancelOnRead = cancel
			}
			r, _, state := configuredAgentMemoryResource(t, testhelpers.NewAPIClient(t, f.api), "100ms")
			resp := resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, &resp)
			require.True(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			assert.True(t, resp.State.Raw.Equal(state.Raw), "failed deletion must retain the tracked state")
			f.api.WithHandlersPaused(func() {
				assert.Equal(t, 1, f.deletes)
				assert.False(t, f.gone)
				assert.False(t, f.changedDeadline)
				if tc.deleteHTTP != 0 {
					assert.Zero(t, f.tasks)
					assert.Zero(t, f.reads)
				} else if tc.taskHTTP != 0 || tc.pending {
					assert.Zero(t, f.reads)
				} else {
					assert.Positive(t, f.reads)
				}
			})
		})
	}
}

func TestAgentMemoryDeleteTerraformRetry(t *testing.T) {
	f := newDeletionFixture(2)
	testingresource.UnitTest(t, testingresource.TestCase{
		ProtoV5ProviderFactories: testhelpers.FrameworkProviderFactories(testhelpers.NewAPIClient(t, f.api), agentmemory.NewAgentMemoryResource),
		Steps: []testingresource.TestStep{
			{ConfigFile: config.StaticFile("testdata/delete_confirmation.tf")},
			{
				ConfigFile:  config.StaticFile("testdata/delete_confirmation.tf"),
				Destroy:     true,
				PreConfig:   func() { f.api.WithHandlersPaused(func() { f.readHTTP = http.StatusInternalServerError }) },
				ExpectError: regexp.MustCompile("Agent Memory service failed to delete"),
			},
			{
				ConfigFile: config.StaticFile("testdata/delete_confirmation.tf"),
				Destroy:    true,
				PreConfig:  func() { f.api.WithHandlersPaused(func() { f.readHTTP = 0 }) },
			},
		},
	})
	f.api.WithHandlersPaused(func() {
		assert.True(t, f.gone)
		assert.Equal(t, 1, f.creates)
		assert.Equal(t, 2, f.deletes)
		assert.Equal(t, 4, f.reads)
		assert.False(t, f.changedDeadline)
	})
}
