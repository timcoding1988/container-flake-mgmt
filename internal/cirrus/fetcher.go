package cirrus

import (
	"context"
	"strings"

	"golang.org/x/sync/errgroup"
)

// FetchResult contains the result of fetching an artifact
type FetchResult struct {
	Request ArtifactRequest
	Data    []byte
	Error   error
}

// Fetcher handles concurrent artifact fetching
type Fetcher struct {
	client  *Client
	workers int
}

// NewFetcher creates a new concurrent fetcher
func NewFetcher(client *Client, workers int) *Fetcher {
	if workers <= 0 {
		workers = 10
	}
	return &Fetcher{
		client:  client,
		workers: workers,
	}
}

// FetchAll fetches all artifacts concurrently using a worker pool.
// Results are returned in the same order as requests (by original index).
// Errors from individual fetches are captured in FetchResult.Error rather than
// failing the entire operation, allowing partial success.
func (f *Fetcher) FetchAll(ctx context.Context, requests []ArtifactRequest) []FetchResult {
	if len(requests) == 0 {
		return nil
	}

	// Pre-allocate results array - each worker writes to its own index (no mutex needed)
	results := make([]FetchResult, len(requests))

	// Create work channel - send indices, not the full request
	workCh := make(chan int, len(requests))
	for i := range requests {
		workCh <- i
	}
	close(workCh)

	// Worker pool using errgroup for coordination
	g, gctx := errgroup.WithContext(ctx)
	for i := 0; i < f.workers; i++ {
		g.Go(func() error {
			for idx := range workCh {
				// Check for cancellation before starting work
				select {
				case <-gctx.Done():
					return gctx.Err()
				default:
				}

				req := requests[idx]
				artifactName := normalizeArtifactName(req.TaskName)

				data, err := f.client.FetchArtifact(gctx, req.TaskID, artifactName)

				// Store result at original index - no mutex needed since each index is unique
				results[idx] = FetchResult{
					Request: req,
					Data:    data,
					Error:   err,
				}
			}
			return nil
		})
	}

	// Wait for all workers to complete
	// Note: Context cancellation errors are expected and don't indicate a problem
	// with the fetching logic - individual fetch errors are in FetchResult.Error
	_ = g.Wait()

	return results
}

// normalizeArtifactName converts a task name to the expected artifact filename
// Task: "sys fedora rootless" -> Artifact: "sys_fedora_rootless.log.html"
func normalizeArtifactName(taskName string) string {
	// Replace spaces and special characters with underscores
	name := strings.ReplaceAll(taskName, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name + ".log.html"
}
