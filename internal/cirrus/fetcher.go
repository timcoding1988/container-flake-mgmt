package cirrus

import (
	"context"
	"log"
	"strings"
	"sync"

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

// FetchAll fetches all artifacts concurrently using a worker pool
func (f *Fetcher) FetchAll(ctx context.Context, requests []ArtifactRequest) []FetchResult {
	if len(requests) == 0 {
		return nil
	}

	results := make([]FetchResult, len(requests))
	var mu sync.Mutex
	resultIdx := 0

	// Create work channel
	workCh := make(chan int, len(requests))
	for i := range requests {
		workCh <- i
	}
	close(workCh)

	// Worker pool
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < f.workers; i++ {
		g.Go(func() error {
			for idx := range workCh {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				req := requests[idx]
				artifactName := normalizeArtifactName(req.TaskName)

				data, err := f.client.FetchArtifact(ctx, req.TaskID, artifactName)

				mu.Lock()
				results[resultIdx] = FetchResult{
					Request: req,
					Data:    data,
					Error:   err,
				}
				resultIdx++
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Printf("Fetcher error: %v", err)
	}

	return results[:resultIdx]
}

// normalizeArtifactName converts a task name to the expected artifact filename
// Task: "sys fedora rootless" -> Artifact: "sys_fedora_rootless.log.html"
func normalizeArtifactName(taskName string) string {
	// Replace spaces and special characters with underscores
	name := strings.ReplaceAll(taskName, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name + ".log.html"
}
