package testhelpers

// MockAPI is an in-memory stand-in for the Redis Cloud REST API, injected via the SDK's
// rediscloudApi.Transporter option. It exists so tests can drive the REAL resource implementations —
// their Create/Read/Update/Delete, their plan modifiers, their guards — rather than a hand-written
// test resource that mirrors them and can silently drift.
//
// The mock sits at the http.RoundTripper boundary because that is the only seam the SDK offers:
// rediscloudApi.Client exposes its services as concrete *databases.API, *subscriptions.API and so on,
// so there is nothing to substitute at the Go level. Nothing listens on a socket — routing runs
// in-process through an http.ServeMux, whose Go 1.22 patterns give method matching, {named} path
// wildcards and specificity-based precedence for free.
//
// Unmatched requests fail loudly with the method and path, so a test that reaches an unstubbed
// endpoint tells you exactly which handler to add instead of failing obscurely.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/google/uuid"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// MockAPIClient builds a real rediscloud SDK client whose transport is the given MockAPI, wrapped in
// the *client.ApiClient that resources expect from ProviderData. The SDK's request handling, task
// polling and model decoding all run for real, against in-memory responses.
func MockAPIClient(mock *MockAPI) (*client.ApiClient, error) {
	api, err := rediscloudApi.NewClient(
		rediscloudApi.Auth("mock-key", "mock-secret"),
		rediscloudApi.Transporter(mock),
	)
	if err != nil {
		return nil, err
	}
	return &client.ApiClient{
		Client: api,
		// The fixture answers on the first poll, so every waiter's pre-poll delay is dead time and the
		// interval is never reached.
		WaitDelayOverride:        time.Millisecond,
		WaitPollIntervalOverride: time.Millisecond,
	}, nil
}

// Handler returns the value to encode as the JSON response body. Read path wildcards with
// req.PathValue("name"), and the request body from req.Body. Returning an error fails the round trip;
// returning a StatusError responds with that HTTP status instead.
type Handler func(req *http.Request) (any, error)

// StatusError makes a handler respond with a non-200 status. Return NotFound() to drive the SDK's
// 404 handling — which is how the provider detects a deleted resource.
type StatusError struct {
	Code int
	Body any
}

func (e StatusError) Error() string { return fmt.Sprintf("mock api: HTTP %d", e.Code) }

// NotFound returns a StatusError the SDK maps to its NotFound error types.
func NotFound() error {
	return StatusError{Code: http.StatusNotFound, Body: map[string]any{"description": "not found"}}
}

// MockAPI serves requests registered by ServeMux pattern.
//
// Handlers run on goroutines owned by Terraform and the plugin framework — the gRPC server's per-RPC
// goroutine, and retry.StateChangeConf's separate Refresh goroutine — so nothing about their ordering is
// the fixture's to control. Terraform's default parallelism is 10, and a config with two resources would
// genuinely overlap.
//
// Two mutexes, never held together:
//
//   - handlerMu serialises handler invocation, so a fixture's own state needs no locking of its own.
//   - mu guards this type's fields below.
//
// The one ordering that occurs is handlerMu then mu, because NewTask and the task-poll handler run under
// handlerMu and take mu inside. RoundTrip releases mu before dispatching, so it never holds both — do not
// change that to hold mu across dispatch, which would invert the order and deadlock.
type MockAPI struct {
	mux *http.ServeMux

	// handlerMu serialises handler invocation. Held only around the handler call.
	handlerMu sync.Mutex

	// mu guards requests and taskResource.
	mu sync.Mutex
	// requests records every request the provider made, in order, as "METHOD /path".
	requests []string
	// taskResource maps a task id to the resource id that task resolves to.
	taskResource map[string]int
}

func NewMockAPI() *MockAPI {
	m := &MockAPI{mux: http.NewServeMux(), taskResource: map[string]int{}}

	// Every Redis Cloud mutation is asynchronous: it returns a task id and the SDK polls this endpoint
	// until the task completes, then reads the resource id out of the response. Registered here rather
	// than per-fixture because it is protocol, not resource behaviour — see NewTask.
	m.Handle("GET /tasks/{taskID}", func(r *http.Request) (any, error) {
		taskID := r.PathValue("taskID")

		// sm-cloud-api types task ids as UUID throughout (RedisRepository.saveTask,
		// getLatestTaskStatus) and rejects ids that will not parse, so a fixture that accepted other
		// shapes could let a bug through.
		if _, err := uuid.Parse(taskID); err != nil {
			return nil, fmt.Errorf("task id %q is not a UUID: the API types these as UUIDs, so whatever "+
				"NewTask returned must be round-tripped unchanged: %w", taskID, err)
		}

		m.mu.Lock()
		resourceID, ok := m.taskResource[taskID]
		m.mu.Unlock()

		if !ok {
			return nil, fmt.Errorf("unknown task %q — tasks must be created with MockAPI.NewTask", taskID)
		}

		return map[string]any{
			"taskId":   taskID,
			"status":   "processing-completed",
			"response": map[string]any{"resourceId": resourceID},
		}, nil
	})

	// Least specific pattern, so it only runs when nothing else matched. 500 rather than 404: a 404 is
	// meaningful to the SDK (deleted resource), and silently returning one would turn a missing handler
	// into confusing provider behaviour rather than an obvious test failure.
	m.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, fmt.Sprintf(
			"mock api: no handler for %s %s — register one with Handle(%q, ...)",
			r.Method, r.URL.Path, r.Method+" "+r.URL.Path,
		), http.StatusInternalServerError)
	})

	return m
}

// Handle registers a handler for a ServeMux pattern, e.g.
// "GET /subscriptions/{subID}/databases/{dbID}". Method matching, {named} wildcards and precedence
// (most specific pattern wins, regardless of registration order) come from ServeMux; conflicting
// patterns panic at registration.
func (m *MockAPI) Handle(pattern string, handler Handler) *MockAPI {
	m.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		m.handlerMu.Lock()
		payload, err := handler(r)
		m.handlerMu.Unlock()

		if err != nil {
			var statusErr StatusError
			if !errors.As(err, &statusErr) {
				http.Error(w, fmt.Sprintf("mock api: handler for %s: %v", pattern, err), http.StatusInternalServerError)
				return
			}
			status, payload = statusErr.Code, statusErr.Body
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, fmt.Sprintf("mock api: encoding response for %s: %v", pattern, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(encoded)
	})
	return m
}

// NewTask registers an asynchronous task resolving to resourceID and returns the body a mutation
// endpoint should respond with. The SDK then polls GET /tasks/{taskID}, which NewMockAPI already serves,
// so a fixture only has to return this from its create/update/delete handlers.
//
// The id is a UUID, as the real API's is. commandType is echoed back purely for realism; nothing depends on it.
//
// Tasks complete immediately: the poll endpoint returns processing-completed on the first call. Neither
// processing-error nor the API's brief 404-before-the-task-is-known window (internal/service.go:145) is
// modelled, so no test here exercises task-failure handling or that retry path.
func (m *MockAPI) NewTask(commandType string, resourceID int) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()
	m.taskResource[id] = resourceID

	return map[string]any{
		"taskId":      id,
		"commandType": commandType,
		"status":      "received",
		"description": "Task request received and is being queued for processing.",
	}
}

// Locked runs fn with handler invocation suspended. Use it for fixture state that a test mutates or
// reads from the test goroutine rather than from inside a handler.
func (m *MockAPI) Locked(fn func()) {
	m.handlerMu.Lock()
	defer m.handlerMu.Unlock()

	fn()
}

// Requests returns every request made so far, in order, as "METHOD /path".
func (m *MockAPI) Requests() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]string(nil), m.requests...)
}

// Calls counts requests whose "METHOD /path" equals the given string. Use it to assert that the
// provider did — or did not — call an endpoint.
func (m *MockAPI) Calls(methodAndPath string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, r := range m.requests {
		if r == methodAndPath {
			count++
		}
	}
	return count
}

func (m *MockAPI) RoundTrip(req *http.Request) (*http.Response, error) {
	// Strip the SDK base URL's /v1 prefix so patterns don't have to repeat it. Clone rather than mutate:
	// the request belongs to the caller.
	routed := req.Clone(req.Context())
	routed.URL.Path = strings.TrimPrefix(req.URL.Path, "/v1")

	m.mu.Lock()
	m.requests = append(m.requests, routed.Method+" "+routed.URL.Path)
	m.mu.Unlock()

	rec := httptest.NewRecorder()
	m.mux.ServeHTTP(rec, routed)

	resp := rec.Result()
	resp.Request = req
	return resp, nil
}
