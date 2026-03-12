package cirrus

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientFetchBuilds(t *testing.T) {
	// Use a recent timestamp (current time in milliseconds)
	recentTimestamp := time.Now().UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := fmt.Sprintf(`{
			"data": {
				"repository": {
					"builds": {
						"edges": [
							{
								"node": {
									"id": "123",
									"branch": "main",
									"changeIdInRepo": "abc123",
									"changeTimestamp": %d,
									"status": "COMPLETED"
								}
							}
						],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": null
						}
					}
				}
			}
		}`, recentTimestamp)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	builds, err := client.FetchBuilds(context.Background(), "containers", "podman", "main", 30*24*time.Hour)
	if err != nil {
		t.Fatalf("FetchBuilds failed: %v", err)
	}

	if len(builds) != 1 {
		t.Fatalf("expected 1 build, got %d", len(builds))
	}

	if builds[0].ID != "123" {
		t.Errorf("expected build ID 123, got %s", builds[0].ID)
	}
}

func TestClientFetchTasks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"build": {
					"tasks": [
						{
							"id": "456",
							"name": "sys fedora rootless",
							"status": "COMPLETED",
							"durationInSeconds": 300
						},
						{
							"id": "457",
							"name": "Build x86_64",
							"status": "COMPLETED",
							"durationInSeconds": 600
						}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	tasks, err := client.FetchTasks(context.Background(), "123")
	if err != nil {
		t.Fatalf("FetchTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Filter for test tasks
	testTasks := make([]Task, 0)
	for _, task := range tasks {
		if task.IsTestTask() {
			testTasks = append(testTasks, task)
		}
	}

	if len(testTasks) != 1 {
		t.Errorf("expected 1 test task, got %d", len(testTasks))
	}
}

func TestClientRateLimitBackoff(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	client.maxRetries = 3
	client.baseDelay = 10 * time.Millisecond
	client.artifactBaseURL = server.URL // Override for testing

	_, err := client.FetchArtifact(context.Background(), "123", "test.html")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
