package testhelpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestFakeAPIConcurrentTasks(t *testing.T) {
	fake := NewFakeAPI()

	for resourceID := range 32 {
		t.Run(fmt.Sprintf("resource-%d", resourceID), func(t *testing.T) {
			t.Parallel()

			taskID := registerDatabaseCreationTask(t, fake, resourceID)
			assertTaskResolvesToResource(t, fake, taskID, resourceID)
		})
	}
}

func registerDatabaseCreationTask(t *testing.T, fake *FakeAPI, resourceID int) string {
	t.Helper()

	task := fake.RegisterTask("databaseCreateRequest", resourceID)
	taskID, ok := task["taskId"].(string)
	if !ok {
		t.Fatalf("task registration returned a non-string task ID: %v", task["taskId"])
	}
	return taskID
}

func assertTaskResolvesToResource(t *testing.T, fake *FakeAPI, taskID string, resourceID int) {
	t.Helper()

	actualTaskID, actualResourceID := readTaskResult(t, fake, taskID)
	if actualTaskID != taskID || actualResourceID != resourceID {
		t.Fatalf("task lookup returned task %q for resource %d; want task %q for resource %d",
			actualTaskID, actualResourceID, taskID, resourceID)
	}
}

func readTaskResult(t *testing.T, fake *FakeAPI, taskID string) (string, int) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		fmt.Sprintf("https://fake.example/v1/tasks/%s", taskID), nil)
	if err != nil {
		t.Fatal(err)
	}
	api := &http.Client{Transport: fake}
	resp, err := api.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("task lookup returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		TaskID   string `json:"taskId"`
		Response struct {
			ResourceID int `json:"resourceId"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result.TaskID, result.Response.ResourceID
}
