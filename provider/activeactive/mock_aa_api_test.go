package activeactive_test

// A stateful in-memory Redis Cloud API for Active-Active databases, wired to the real resource via
// testhelpers.MockAPI. Tests drive activeactive.NewActiveActiveDatabaseResource against it, so
// production Create/Read/Update/Delete, plan modifiers and guards all execute for real.
//
// A fixture holds ONE subscription and ONE database, because a Terraform test drives one resource of a
// given type. Ids come from a counter rather than being hardcoded, so they are obviously synthetic in
// failure output and two fixtures in the same run never collide.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-version"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const (
	// defaultRedisVersion is the version the API picks when a create request omits redisVersion.
	defaultRedisVersion = "8.0"
)

type mockDatabase struct {
	id   int
	name string
	// globalPassword is whatever the last create/update request supplied, or a generated value when the
	// create request omitted one. Echoed back on read: returning some other value would contradict the
	// plan, since the provider planned the password it sent — "inconsistent values for sensitive
	// attribute".
	globalPassword string
	// redisVersion is the version the database is actually RUNNING — the value the API reports back,
	// which redis_version_actual reflects. Tests mutate it directly to simulate a background auto minor
	// upgrade moving it out from under Terraform.
	redisVersion string
}

// mockAAAPI holds the fixture's state.
//
// No mutex of its own: testhelpers.MockAPI serialises handler invocation, so handlers may touch this
// freely. The two methods a test calls from its own goroutine go through MockAPI.Locked.
type mockAAAPI struct {
	mock *testhelpers.MockAPI

	nextID int
	subID  int
	// database is nil before create and after delete. Nil is what lets deleteDatabase's wait terminate:
	// it polls until the API 404s, so a fixture that kept answering 200 would spin until the wait's
	// timeout.
	database *mockDatabase

	// upgradeRequests records every target passed to the version-upgrade endpoint, in order, whether or
	// not the API honoured it. A rejected downgrade is indistinguishable from a request never sent if
	// you only look at resulting state, so this is what lets a test prove the provider asked for
	// something it should not have.
	upgradeRequests []string
}

func newMockAAAPI() *mockAAAPI {
	// Start high enough that ids are never mistaken for array indices or for the small numbers that
	// appear in fixtures.
	return &mockAAAPI{mock: testhelpers.NewMockAPI(), nextID: 1000}
}

func (a *mockAAAPI) allocID() int {
	a.nextID++
	return a.nextID
}

// addSubscription registers the fixture's subscription and returns its id, for use in a config.
func (a *mockAAAPI) addSubscription() int {
	a.mock.Locked(func() { a.subID = a.allocID() })
	return a.subID
}

// setRunningVersion simulates the running version changing out from under Terraform, e.g. a background
// auto minor upgrade.
func (a *mockAAAPI) setRunningVersion(redisVersion string) {
	a.mock.Locked(func() { a.database.redisVersion = redisVersion })
}

func (a *mockAAAPI) requestedUpgrades() (targets []string) {
	a.mock.Locked(func() { targets = a.upgradeRequests })
	return targets
}

// requestedDatabase returns the fixture's database if the request addresses it, or NotFound. The id check
// is what would catch the provider addressing the wrong resource — a parseResourceId bug, say — rather
// than silently operating on the only database there is.
func (a *mockAAAPI) requestedDatabase(r *http.Request) (*mockDatabase, error) {
	if a.database == nil || pathInt(r, "dbID") != a.database.id {
		return nil, testhelpers.NotFound()
	}
	return a.database, nil
}

// mock builds the MockAPI serving this fixture's state.
func (a *mockAAAPI) buildMock() *testhelpers.MockAPI {
	mock := a.mock

	// Subscription status, polled by WaitForSubscriptionToBeActive.
	mock.Handle("GET /subscriptions/{subID}", func(r *http.Request) (any, error) {

		if pathInt(r, "subID") != a.subID {
			return nil, testhelpers.NotFound()
		}
		return map[string]any{"id": a.subID, "status": "active"}, nil
	})

	// Regions, listed during createDatabase to build localThroughputMeasurement.
	mock.Handle("GET /subscriptions/{subID}/regions", func(r *http.Request) (any, error) {
		return map[string]any{
			"subscriptionId": pathInt(r, "subID"),
			"regions": []map[string]any{
				{"region": "us-east-1", "networking": []map[string]any{}},
			},
		}, nil
	})

	mock.Handle("POST /subscriptions/{subID}/databases", func(r *http.Request) (any, error) {
		var body struct {
			Name         string  `json:"name"`
			RedisVersion string  `json:"redisVersion"`
			Password     *string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("decoding create request: %w", err)
		}

		// Mirror the real create API: when redisVersion is omitted the database is created at a
		// server-chosen default rather than failing.
		running := body.RedisVersion
		if running == "" {
			running = defaultRedisVersion
		}

		id := a.allocID()

		// A supplied password is honoured verbatim, including an empty one — createDatabase sends "" to
		// mean passwordless. Only an absent field means "generate me one", mirroring the real API's "if
		// left empty, the password will be generated automatically". The generated value is the fixture's
		// own business: it is per-database, non-empty (an empty password reads back as passwordless), and
		// no test asserts it.
		password := fmt.Sprintf("generated-password-%d", id)
		if body.Password != nil {
			password = *body.Password
		}

		a.database = &mockDatabase{
			id:             id,
			name:           body.Name,
			globalPassword: password,
			redisVersion:   running,
		}

		return mock.NewTask("databaseCreateRequest", id), nil
	})

	// Active-Active updates go to the per-database regions endpoint, not the database itself.
	mock.Handle("PUT /subscriptions/{subID}/databases/{dbID}/regions", func(r *http.Request) (any, error) {
		var body struct {
			Name           *string `json:"name"`
			GlobalPassword *string `json:"globalPassword"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		db, err := a.requestedDatabase(r)
		if err != nil {
			return nil, err
		}
		// Apply what the request actually asked for. Ignoring it would make the subsequent read report
		// the old name, contradicting the plan — "Provider produced inconsistent result after apply".
		if body.Name != nil {
			db.name = *body.Name
		}
		if body.GlobalPassword != nil {
			db.globalPassword = *body.GlobalPassword
		}
		return mock.NewTask("databaseUpdateRequest", db.id), nil
	})

	// The version-upgrade endpoint. Records every request, then honours it only when it really is an
	// upgrade: no API moves a live database to a lower version in place.
	mock.Handle("POST /subscriptions/{subID}/databases/{dbID}/upgrade", func(r *http.Request) (any, error) {
		var body struct {
			TargetRedisVersion string `json:"targetRedisVersion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("decoding upgrade request: %w", err)
		}

		db, err := a.requestedDatabase(r)
		if err != nil {
			return nil, err
		}

		a.upgradeRequests = append(a.upgradeRequests, body.TargetRedisVersion)

		if higherVersion(body.TargetRedisVersion, db.redisVersion) {
			db.redisVersion = body.TargetRedisVersion
		}

		return mock.NewTask("databaseUpgradeRequest", db.id), nil
	})

	mock.Handle("PUT /subscriptions/{subID}/databases/{dbID}/tags", func(_ *http.Request) (any, error) {
		return map[string]any{"tags": []map[string]any{}}, nil
	})

	mock.Handle("GET /subscriptions/{subID}/databases/{dbID}/tags", func(_ *http.Request) (any, error) {
		return map[string]any{"tags": []map[string]any{}}, nil
	})

	mock.Handle("DELETE /subscriptions/{subID}/databases/{dbID}", func(r *http.Request) (any, error) {

		db, err := a.requestedDatabase(r)
		if err != nil {
			return nil, err
		}
		a.database = nil

		return mock.NewTask("databaseDeleteRequest", db.id), nil
	})

	// The database, read by GetActiveActive and polled by WaitForDatabaseToBeActive. Every field the
	// real API always returns has to be present: readDatabase leaves Optional+Computed attributes
	// unknown if the response omits them, and the framework rejects unknowns after apply.
	mock.Handle("GET /subscriptions/{subID}/databases/{dbID}", func(r *http.Request) (any, error) {

		db, err := a.requestedDatabase(r)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"databaseId":                          db.id,
			"name":                                db.name,
			"status":                              "active",
			"redisVersion":                        db.redisVersion,
			"memoryLimitInGb":                     1,
			"datasetSizeInGb":                     1,
			"globalPassword":                      db.globalPassword,
			"dataEvictionPolicy":                  "volatile-lru",
			"supportOSSClusterApi":                false,
			"useExternalEndpointForOSSClusterApi": false,
			"crdbDatabases": []map[string]any{
				{
					"provider":               "AWS",
					"region":                 "us-east-1",
					"redisVersionCompliance": db.redisVersion,
					"publicEndpoint":         "pub.example.com:12000",
					"privateEndpoint":        "priv.example.com:12000",
					"dataPersistence":        "none",
					"security": map[string]any{
						"enableDefaultUser": true,
						"sourceIps":         []string{"0.0.0.0/0"},
					},
					"alerts": []map[string]any{},
				},
			},
		}, nil
	})

	return mock
}

// pathInt reads a {named} path wildcard as an int. Path values are always strings, so numeric ids need
// converting before they go anywhere near a JSON response.
func pathInt(r *http.Request, name string) int {
	var v int
	_, _ = fmt.Sscanf(r.PathValue(name), "%d", &v)
	return v
}

// higherVersion reports whether target is a genuinely higher version than current.
func higherVersion(target, current string) bool {
	t, err := version.NewVersion(target)
	if err != nil {
		return false
	}
	c, err := version.NewVersion(current)
	if err != nil {
		return false
	}
	return t.GreaterThan(c)
}
