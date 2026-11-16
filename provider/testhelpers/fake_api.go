package testhelpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// The SDK keeps task status values in its internal package, so we mirror them here
const (
	taskStatusReceived            = "received"
	taskStatusProcessingCompleted = "processing-completed"
)

// FakeAPI is an in-memory stand-in for the Redis Cloud REST API. It is an http.RoundTripper that the SDK client
// talks to in place of the network. It exists so tests can drive the REAL resource implementations — their
// Create/Read/Update/Delete, their plan modifiers and guards against a mocked/faked backend.
//
// It owns two things:
//
//   - transport and routing — matching a request to a handler, encoding what that handler returns as JSON,
//     turning its errors into HTTP statuses;
//   - task management - the one piece of Redis Cloud protocol that belongs to no single resource.
//     Every mutation runs through asynchronous tasks, which the SDK client then polls to wait for their
//     completion (see RegisterTask).
//
// FakeAPI knows nothing about databases or subscriptions. Domain state and per-endpoint behaviour live in a
// per-resource fixture that registers handlers with Handle — see fake_aa_api_test.go in
// provider/activeactive for a working example.
//
// Concurrency: Terraform and the plugin framework call handlers from several goroutines, so fixtureMu
// serialises handler invocation. A fixture's state is therefore touched by one goroutine at a time and
// needs no locking of its own. Test code that needs to read or write fixture state outside of a handler
// should use WithHandlersPaused.
type FakeAPI struct {
	mux *http.ServeMux

	fixtureMu sync.Mutex

	// Tasks can also be registered outside handlers, so ID allocation must be safe for concurrent callers.
	nextTaskID atomic.Uint64
}

var _ http.RoundTripper = (*FakeAPI)(nil)

func NewFakeAPI() *FakeAPI {
	f := &FakeAPI{
		mux: http.NewServeMux(),
	}

	// This is the least specific pattern, so it runs only when nothing else matched. It responds with a 500
	// rather than a 404, because a 404 has meaning to the SDK client. A 404 is how the provider detects a
	// deleted resource, so returning one here would turn a missing handler into confusing provider behaviour
	// and a non-obvious test failure.
	f.Handle("/", func(r *http.Request) (any, error) {
		return nil, fmt.Errorf("no handler registered — add one with Handle(%q, ...)",
			r.Method+" "+r.URL.Path)
	})

	return f
}

// NewAPIClient sets a FakeAPI as the transport for a rediscloudApi.Client, then wraps that in a
// *client.ApiClient, as this is what resources and data sources expect from ProviderData.
// The goal is to run the SDK client's request handling, task polling and model decoding for real,
// but against in-memory responses.
func NewAPIClient(t *testing.T, fake *FakeAPI) *client.ApiClient {
	t.Helper()

	api, err := rediscloudApi.NewClient(
		rediscloudApi.Auth("fake-key", "fake-secret"),
		rediscloudApi.Transporter(fake),
	)
	if err != nil {
		t.Fatalf("building a rediscloud-go-api client over the fake API failed: %s", err)
	}

	return &client.ApiClient{
		Client: api,
		// A fake answers on the first poll, so a waiter's delay before that poll would be dead time.
		// Set a millisecond rather than zero, because zero means "not overridden" and would revert
		// to the default production timings.
		WaitDelayOverride:        time.Millisecond,
		WaitPollIntervalOverride: time.Millisecond,
	}
}

// HandlerFunc handles a single request. Return the value to encode as the JSON response body, reading path
// parameters with req.PathValue("name") and the request body from req.Body. Return an error to fail the
// request as a 500, or a StatusError to respond with a particular status.
type HandlerFunc func(req *http.Request) (any, error)

// StatusError makes a handler respond with a non-200 status. Use the NotFound() below to drive the SDK client's
// 404 handling, which is how the provider detects a deleted resource. For any other status, construct a
// StatusError on your own.
type StatusError struct {
	Code int
	Body any
}

func (e StatusError) Error() string { return fmt.Sprintf("HTTP %d", e.Code) }

// NotFound returns a StatusError that the SDK maps to its own NotFound error types.
func NotFound() error {
	return StatusError{Code: http.StatusNotFound, Body: map[string]any{"description": "not found"}}
}

// Handle registers a handler for a ServeMux pattern, such as
// "GET /subscriptions/{subID}/databases/{dbID}". Write patterns without the /v1 prefix, as SDK Client
// requests are matched to handler patterns without the prefix. See RoundTrip() for more info.
// Method matching, {named} path parameters and precedence all come from ServeMux, so the
// most specific pattern wins whatever the registration order, and registering the same pattern twice
// panics.
func (f *FakeAPI) Handle(pattern string, handler HandlerFunc) *FakeAPI {
	f.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		request := r.Method + " " + r.URL.Path

		status := http.StatusOK
		payload, err := f.invoke(handler, r)
		if err != nil {
			var statusErr StatusError
			if !errors.As(err, &statusErr) {
				http.Error(w, fmt.Sprintf("fake api: %s: %v", request, err), http.StatusInternalServerError)
				return
			}
			status, payload = statusErr.Code, statusErr.Body
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("fake api: encoding the response to %s: %v", request, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(encoded)
	})
	return f
}

func (f *FakeAPI) invoke(handler HandlerFunc, r *http.Request) (payload any, err error) {
	f.fixtureMu.Lock()
	defer f.fixtureMu.Unlock()

	// convert a panic into an error
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("handler panicked: %v\n\n%s", p, debug.Stack())
		}
	}()

	return handler(r)
}

// RegisterTask registers an asynchronous task that maps to a resourceID and returns the response body that
// should be returned from a fixture's create, update and delete handlers.
// RegisterTask installs the task's GET /tasks/{taskID} handler so the SDK's WaitForResourceId can retrieve
// the result without further fixture setup.
//
// Every task completes immediately and succeeds, so no test can exercise a failed task or the SDK Client's
// retry mechanisms.
func (f *FakeAPI) RegisterTask(commandType string, resourceID int) map[string]any {
	id := fmt.Sprintf("task-%d", f.nextTaskID.Add(1))
	getTask := func(*http.Request) (any, error) {
		return map[string]any{
			"taskId":   id,
			"status":   taskStatusProcessingCompleted,
			"response": map[string]any{"resourceId": resourceID},
		}, nil
	}
	f.Handle("GET /tasks/"+id, getTask)

	return map[string]any{
		"taskId":      id,
		"commandType": commandType,
		"status":      taskStatusReceived,
		"description": "Task request received and is being queued for processing.",
	}
}

// RegisterResourceTask registers an asynchronous task that returns a resource
// payload through the SDK's WaitForResource. RegisterResourceTask installs the
// task's GET /tasks/{taskID} handler, which immediately reports successful completion.
func (f *FakeAPI) RegisterResourceTask(commandType string, resource any) map[string]any {
	id := fmt.Sprintf("task-%d", f.nextTaskID.Add(1))
	getTask := func(*http.Request) (any, error) {
		return map[string]any{
			"taskId":   id,
			"status":   taskStatusProcessingCompleted,
			"response": map[string]any{"resource": resource},
		}, nil
	}
	f.Handle("GET /tasks/"+id, getTask)

	return map[string]any{
		"taskId":      id,
		"commandType": commandType,
		"status":      taskStatusReceived,
		"description": "Task request received and is being queued for processing.",
	}
}

// WithHandlersPaused runs fn() with handler invocation suspended. Use it whenever test code needs to read or write
// fixture state outside a handler, so that data access doesn't race between test code and handlers.
func (f *FakeAPI) WithHandlersPaused(fn func()) {
	f.fixtureMu.Lock()
	defer f.fixtureMu.Unlock()

	fn()
}

// RoundTrip makes FakeAPI an http.RoundTripper. It routes the request to a registered handler in process
// and returns that handler's response, so nothing reaches the network.
func (f *FakeAPI) RoundTrip(req *http.Request) (*http.Response, error) {
	const apiPathPrefix = "/v1"

	// An http.RoundTripper is required to close the request body, including when it returns an error.
	if req.Body != nil {
		defer func() {
			_ = req.Body.Close()
		}()
	}

	// RoundTrip serves requests made by the SDK client, that is why all requests come with a /v1 prefix.
	// Fail loudly here if the SDK client's base URL ever stops carrying this prefix. Otherwise no
	// handler would match and the test will fail with a confusing "no handler" instead.
	if !strings.HasPrefix(req.URL.Path, apiPathPrefix) {
		return nil, fmt.Errorf("fake api: %s does not start with %s, but patterns are registered without it",
			req.URL.Path, apiPathPrefix)
	}

	// Clone the request rather than mutating it, because the request belongs to the caller.
	routed := req.Clone(req.Context())
	routed.URL.Path = strings.TrimPrefix(req.URL.Path, apiPathPrefix)

	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, routed)

	resp := rec.Result()
	resp.Request = req
	return resp, nil
}
