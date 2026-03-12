package cirrus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestFetcherConcurrency(t *testing.T) {
	var concurrent int32
	var maxConcurrent int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&concurrent, 1)
		defer atomic.AddInt32(&concurrent, -1)

		// Track max concurrent requests
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if cur <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, cur) {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
		w.Write([]byte("<html><body>test</body></html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	client.artifactBaseURL = server.URL // Use test server for artifact fetches
	fetcher := NewFetcher(client, 5) // 5 workers

	requests := make([]ArtifactRequest, 20)
	for i := range requests {
		requests[i] = ArtifactRequest{
			TaskID:   "task-" + string(rune('a'+i)),
			TaskName: "test task",
		}
	}

	results := fetcher.FetchAll(context.Background(), requests)

	if len(results) != 20 {
		t.Errorf("expected 20 results, got %d", len(results))
	}

	// Verify concurrency was limited
	if maxConcurrent > 5 {
		t.Errorf("max concurrent requests %d exceeded limit of 5", maxConcurrent)
	}
}
