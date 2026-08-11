package activeactive_test

// This file holds a stateful in-memory Redis Cloud API for Active-Active databases, layered on
// testhelpers.FakeAPI. The FakeAPI handles transport, routing and the asynchronous task protocol.
// Everything here is domain scoped — the states that subscriptions and databases have,
// and what each endpoint does to that state.
//
// Test code that touches fixture state outside a handler should go through FakeAPI.WithHandlersPaused.
// Refer to setRunningVersion and getRequestedUpgrades for an example.
//
// A fixture holds one subscription and one database, because a Terraform test drives one resource of a
// given type. Ids come from a counter rather than being hardcoded, so that they are obviously synthetic in
// failure output and two fixtures in the same run never collide.

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-version"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// defaultRedisVersion is the version the API picks when a create request omits redisVersion.
const defaultRedisVersion = "8.0"

type databaseState struct {
	id   int
	name string
	// globalPassword is whatever the last create or update request supplied, or a generated value when the
	// create request omitted one. Read handlers have to echo it back, because the provider planned the
	// password it sent, and returning anything else fails the apply with "inconsistent values for sensitive
	// attribute".
	globalPassword string
	// actualRedisVersion is the version the database is actually running. It is the value the API reports back and
	// the one redis_version_actual reflects. Use setRunningVersion to change it during a test, which
	// simulates a background auto minor upgrade moving the version out from under Terraform.
	actualRedisVersion string
}

type aaAPI struct {
	fake *testhelpers.FakeAPI

	nextID int
	subID  int
	// database is nil before create and after delete. It has to become nil on delete, because
	// deleteDatabase waits for the API to answer 404, and a fixture that kept answering 200 would spin until
	// that wait timed out.
	database *databaseState

	// upgradeRequests records every target passed to the version-upgrade endpoint, in order, whether or not
	// the fixture honoured it. Assert on it through getRequestedUpgrades when a test needs to prove that the
	// provider asked for something it should not have, because a rejected request and a request that was
	// never sent leave identical state behind.
	upgradeRequests []string
}

// newAAAPI returns a fixture with its subscription allocated and its handlers registered, ready to be
// wired into a rediscloud-go-api client with testhelpers.NewAPIClient.
func newAAAPI() *aaAPI {
	a := &aaAPI{
		fake: testhelpers.NewFakeAPI(),
		// Start high enough, so that an id is never mistaken for an array index, or for one of the
		// small numbers that appear in the test configs.
		nextID: 1000,
	}

	a.subID = a.getNextID()
	a.registerHandlers()

	return a
}

func (a *aaAPI) getNextID() int {
	a.nextID++
	return a.nextID
}

// setRunningVersion simulates the running version changing out from under Terraform, the same way
// a background auto minor upgrade would.
func (a *aaAPI) setRunningVersion(redisVersion string) {
	a.fake.WithHandlersPaused(func() { a.database.actualRedisVersion = redisVersion })
}

// getRequestedUpgrades returns the targets sent to the version-upgrade endpoint so far, in order.
func (a *aaAPI) getRequestedUpgrades() (targets []string) {
	a.fake.WithHandlersPaused(func() { targets = append([]string(nil), a.upgradeRequests...) })
	return targets
}

// getRequestedDatabase returns the fixture's database when a request addresses it and NotFound otherwise.
func (a *aaAPI) getRequestedDatabase(r *http.Request) (*databaseState, error) {
	// Check the id rather than just returning the only database there is, so that a provider bug that
	// addressed the wrong resource shows up as a failure instead of passing unnoticed
	if a.database == nil || pathInt(r, "dbID") != a.database.id {
		return nil, testhelpers.NotFound()
	}
	return a.database, nil
}

func (a *aaAPI) registerHandlers() {
	// This endpoint serves subscription status, which WaitForSubscriptionToBeActive polls.
	a.fake.Handle("GET /subscriptions/{subID}", func(r *http.Request) (any, error) {
		if pathInt(r, "subID") != a.subID {
			return nil, testhelpers.NotFound()
		}
		return map[string]any{"id": a.subID, "status": "active"}, nil
	})

	// createDatabase lists regions in order to build localThroughputMeasurement.
	a.fake.Handle("GET /subscriptions/{subID}/regions", func(r *http.Request) (any, error) {
		return map[string]any{
			"subscriptionId": pathInt(r, "subID"),
			"regions": []map[string]any{
				{"region": "us-east-1", "networking": []map[string]any{}},
			},
		}, nil
	})

	a.fake.Handle("POST /subscriptions/{subID}/databases", func(r *http.Request) (any, error) {
		var body struct {
			Name         string  `json:"name"`
			RedisVersion string  `json:"redisVersion"`
			Password     *string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("decoding create request: %w", err)
		}

		// This mirrors the real create API. When redisVersion is omitted, the database is created at a
		// server-chosen default rather than failing.
		running := body.RedisVersion
		if running == "" {
			running = defaultRedisVersion
		}

		id := a.getNextID()

		// A supplied password is honoured verbatim, including an empty one, because createDatabase sends ""
		// to mean passwordless. Only an absent field means "generate me one", which mirrors the real API.
		// The generated value itself is the fixture's own business. It is per-database and non-empty, since
		// an empty password would read back as passwordless, and no test asserts what it's actual value is.
		password := fmt.Sprintf("generated-password-%d", id)
		if body.Password != nil {
			password = *body.Password
		}

		a.database = &databaseState{
			id:                 id,
			name:               body.Name,
			globalPassword:     password,
			actualRedisVersion: running,
		}

		return a.fake.RegisterTask("databaseCreateRequest", id), nil
	})

	// Active-Active updates go to the per-database regions endpoint, not the database itself.
	a.fake.Handle("PUT /subscriptions/{subID}/databases/{dbID}/regions", func(r *http.Request) (any, error) {
		var body struct {
			Name           *string `json:"name"`
			GlobalPassword *string `json:"globalPassword"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)

		db, err := a.getRequestedDatabase(r)
		if err != nil {
			return nil, err
		}
		// Apply what the request asked for. Otherwise tests will fail with "Provider produced inconsistent
		// result after apply" when TF refreshes the state after the apply.
		if body.Name != nil {
			db.name = *body.Name
		}
		if body.GlobalPassword != nil {
			db.globalPassword = *body.GlobalPassword
		}
		return a.fake.RegisterTask("databaseUpdateRequest", db.id), nil
	})

	// The version-upgrade endpoint records every request, then honours it only when the target really is an
	// upgrade, because no API moves a live database to a lower version in place.
	a.fake.Handle("POST /subscriptions/{subID}/databases/{dbID}/upgrade", func(r *http.Request) (any, error) {
		var body struct {
			TargetRedisVersion string `json:"targetRedisVersion"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("decoding upgrade request: %w", err)
		}

		db, err := a.getRequestedDatabase(r)
		if err != nil {
			return nil, err
		}

		a.upgradeRequests = append(a.upgradeRequests, body.TargetRedisVersion)

		if isUpgrade(body.TargetRedisVersion, db.actualRedisVersion) {
			db.actualRedisVersion = body.TargetRedisVersion
		}

		return a.fake.RegisterTask("databaseUpgradeRequest", db.id), nil
	})

	a.fake.Handle("PUT /subscriptions/{subID}/databases/{dbID}/tags", func(_ *http.Request) (any, error) {
		return map[string]any{"tags": []map[string]any{}}, nil
	})

	a.fake.Handle("GET /subscriptions/{subID}/databases/{dbID}/tags", func(_ *http.Request) (any, error) {
		return map[string]any{"tags": []map[string]any{}}, nil
	})

	a.fake.Handle("DELETE /subscriptions/{subID}/databases/{dbID}", func(r *http.Request) (any, error) {
		db, err := a.getRequestedDatabase(r)
		if err != nil {
			return nil, err
		}
		a.database = nil

		return a.fake.RegisterTask("databaseDeleteRequest", db.id), nil
	})

	// GetActiveActive reads this endpoint and WaitForDatabaseToBeActive polls it. Keep every field that the
	// real API always returns present here, because readDatabase leaves an Optional+Computed attribute
	// unknown when the response omits it, and the framework rejects an unknown value after apply.
	a.fake.Handle("GET /subscriptions/{subID}/databases/{dbID}", func(r *http.Request) (any, error) {
		db, err := a.getRequestedDatabase(r)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"databaseId":                          db.id,
			"name":                                db.name,
			"status":                              "active",
			"redisVersion":                        db.actualRedisVersion,
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
					"redisVersionCompliance": db.actualRedisVersion,
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
}

// pathInt reads a {named} path parameter as an int
func pathInt(r *http.Request, name string) int {
	var v int
	_, _ = fmt.Sscanf(r.PathValue(name), "%d", &v)
	return v
}

// isUpgrade reports whether target is genuinely a higher version than current.
func isUpgrade(target, current string) bool {
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
