package utils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteCreationPlanDatabase(t *testing.T) {
	tests := []struct {
		name         string
		states       []string
		taskErrors   []int
		deleteStatus int
		getStatus    int
		wantDeletes  int
		wantError    string
	}{
		{
			name:        "retries an asynchronous backend failure",
			states:      []string{"active", "active", "missing"},
			taskErrors:  []int{http.StatusInternalServerError, 0},
			wantDeletes: 2,
		},
		{
			name:        "does not retry when the failed task removed the database",
			states:      []string{"active", "missing"},
			taskErrors:  []int{http.StatusInternalServerError},
			wantDeletes: 1,
		},
		{
			name:        "waits for deletion already in progress after a failed task",
			states:      []string{"active", "delete-pending", "deleting", "missing"},
			taskErrors:  []int{http.StatusServiceUnavailable},
			wantDeletes: 1,
		},
		{
			name:        "does not resubmit an accepted deletion while reads are stale",
			states:      []string{"active", "active", "delete-pending", "missing"},
			taskErrors:  []int{0},
			wantDeletes: 1,
		},
		{
			name:        "already absent",
			states:      []string{"missing"},
			wantDeletes: 0,
		},
		{
			name:        "bounds repeated backend failures",
			states:      []string{"active"},
			taskErrors:  []int{http.StatusInternalServerError, http.StatusInternalServerError, http.StatusInternalServerError},
			wantDeletes: 3,
			wantError:   "database deletion failed after 3 attempts",
		},
		{
			name:        "does not retry authentication errors",
			states:      []string{"active"},
			taskErrors:  []int{http.StatusUnauthorized},
			wantDeletes: 1,
			wantError:   "401",
		},
		{
			name:        "does not retry permission errors",
			states:      []string{"active"},
			taskErrors:  []int{http.StatusForbidden},
			wantDeletes: 1,
			wantError:   "403",
		},
		{
			name:         "does not retry HTTP authentication errors",
			states:       []string{"active"},
			deleteStatus: http.StatusUnauthorized,
			wantDeletes:  1,
			wantError:    "401",
		},
		{
			name:        "does not treat a reconciliation error as absence",
			states:      []string{"active"},
			getStatus:   http.StatusServiceUnavailable,
			wantDeletes: 0,
			wantError:   "failed to reconcile database deletion",
		},
		{
			name:        "does not delete a database in an unexpected state",
			states:      []string{"error"},
			wantDeletes: 0,
			wantError:   `unexpected status "error"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gets, deletes atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/subscriptions/12/databases/34":
					if test.getStatus != 0 {
						w.WriteHeader(test.getStatus)
						_, _ = w.Write([]byte(`{}`))
						return
					}
					state := test.states[min(int(gets.Add(1))-1, len(test.states)-1)]
					if state == "missing" {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{}`))
						return
					}
					_, _ = fmt.Fprintf(w, `{"databaseId":34,"status":%q}`, state)
				case r.Method == http.MethodDelete && r.URL.Path == "/subscriptions/12/databases/34":
					attempt := deletes.Add(1)
					if test.deleteStatus != 0 {
						w.WriteHeader(test.deleteStatus)
						_, _ = w.Write([]byte(`{}`))
						return
					}
					_, _ = fmt.Fprintf(w, `{"taskId":"delete-%d"}`, attempt)
				case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/delete-"):
					attempt, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/tasks/delete-"))
					if err != nil || attempt < 1 || attempt > len(test.taskErrors) {
						t.Errorf("unexpected task request %s", r.URL.Path)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					status := test.taskErrors[attempt-1]
					if status == 0 {
						_, _ = fmt.Fprintf(w, `{"taskId":"delete-%d","status":"processing-completed","response":{}}`, attempt)
						return
					}
					_, _ = fmt.Fprintf(w, `{"taskId":"delete-%d","status":"processing-error","response":{"error":{"status":"%d %s","type":"GENERAL_ERROR","description":"cleanup failed"}}}`, attempt, status, http.StatusText(status))
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
				}
			}))
			t.Cleanup(server.Close)
			api, err := rediscloudApi.NewClient(rediscloudApi.BaseURL(server.URL), rediscloudApi.Auth("test", "test"))
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			t.Cleanup(cancel)

			err = deleteCreationPlanDatabase(ctx, 12, 34, api.Database, time.Millisecond)
			if test.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantError)
			}
			assert.Equal(t, test.wantDeletes, int(deletes.Load()))
		})
	}
}

func TestDeleteCreationPlanDatabaseCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	api := &cancellingCleanupAPI{cancel: cancel}
	started := time.Now()
	err := deleteCreationPlanDatabase(ctx, 12, 34, api, time.Hour)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, api.deletes)
	assert.Less(t, time.Since(started), time.Second)
}

func TestDeleteCreationPlanDatabaseWaitsForAbsenceWithinDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	api := &cancellingCleanupAPI{}
	err := deleteCreationPlanDatabase(ctx, 12, 34, api, time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 1, api.deletes)
}

type cancellingCleanupAPI struct {
	cancel  context.CancelFunc
	deletes int
}

func (a *cancellingCleanupAPI) Get(context.Context, int, int) (*databases.Database, error) {
	return &databases.Database{ID: redis.Int(34), Status: redis.String(databases.StatusActive)}, nil
}

func (a *cancellingCleanupAPI) Delete(context.Context, int, int) error {
	a.deletes++
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}
