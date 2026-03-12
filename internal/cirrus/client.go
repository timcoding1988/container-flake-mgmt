package cirrus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const (
	DefaultAPIURL     = "https://api.cirrus-ci.com/graphql"
	// Artifact URL format: html_artifacts is the artifact name in podman's .cirrus.yml
	ArtifactURLFormat = "https://api.cirrus-ci.com/v1/artifact/task/%s/html_artifacts/%s"
)

// Client is a GraphQL client for the Cirrus CI API
type Client struct {
	apiURL          string
	artifactBaseURL string
	httpClient      *http.Client
	token           string
	maxRetries      int
	baseDelay       time.Duration
}

// NewClient creates a new Cirrus CI client
func NewClient(apiURL, token string) *Client {
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	return &Client{
		apiURL:          apiURL,
		artifactBaseURL: "", // empty means use default ArtifactURLFormat
		httpClient:      &http.Client{Timeout: 30 * time.Second},
		token:           token,
		maxRetries:      5,
		baseDelay:       1 * time.Second,
	}
}

// graphQLRequest executes a GraphQL query
func (c *Client) graphQLRequest(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Truncate error response to prevent log spam
		errMsg := string(respBody)
		if len(errMsg) > 500 {
			errMsg = errMsg[:500] + "..."
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, errMsg)
	}

	var result struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %s", result.Errors[0].Message)
	}

	return result.Data, nil
}

// FetchBuilds fetches builds for a repository within the given time window
func (c *Client) FetchBuilds(ctx context.Context, owner, repo, branch string, window time.Duration) ([]Build, error) {
	query := `
		query($owner: String!, $repo: String!, $branch: String!, $cursor: String) {
			repository(owner: $owner, name: $repo) {
				builds(branch: $branch, first: 100, after: $cursor) {
					edges {
						node {
							id
							branch
							changeIdInRepo
							changeTimestamp
							status
						}
					}
					pageInfo {
						hasNextPage
						endCursor
					}
				}
			}
		}
	`

	var allBuilds []Build
	var cursor *string

	for {
		variables := map[string]interface{}{
			"owner":  owner,
			"repo":   repo,
			"branch": branch,
			"cursor": cursor,
		}

		data, err := c.graphQLRequest(ctx, query, variables)
		if err != nil {
			return nil, err
		}

		var response struct {
			Repository struct {
				Builds struct {
					Edges []struct {
						Node Build `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool    `json:"hasNextPage"`
						EndCursor   *string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"builds"`
			} `json:"repository"`
		}

		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("unmarshal builds: %w", err)
		}

		for _, edge := range response.Repository.Builds.Edges {
			build := edge.Node
			if !build.IsRecent(window) {
				// Builds are ordered by time, so we can stop here
				return allBuilds, nil
			}
			allBuilds = append(allBuilds, build)
		}

		if !response.Repository.Builds.PageInfo.HasNextPage {
			break
		}
		cursor = response.Repository.Builds.PageInfo.EndCursor
	}

	return allBuilds, nil
}

// FetchTasks fetches all tasks for a build
func (c *Client) FetchTasks(ctx context.Context, buildID string) ([]Task, error) {
	query := `
		query($buildID: ID!) {
			build(id: $buildID) {
				tasks {
					id
					name
					status
					durationInSeconds
				}
			}
		}
	`

	variables := map[string]interface{}{
		"buildID": buildID,
	}

	data, err := c.graphQLRequest(ctx, query, variables)
	if err != nil {
		return nil, err
	}

	var response struct {
		Build struct {
			Tasks []Task `json:"tasks"`
		} `json:"build"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}

	return response.Build.Tasks, nil
}

// FetchArtifact downloads an artifact file from a task with retry and backoff
func (c *Client) FetchArtifact(ctx context.Context, taskID, filename string) ([]byte, error) {
	var url string
	if c.artifactBaseURL != "" {
		// For testing: use custom base URL
		url = c.artifactBaseURL
	} else {
		// Production: use standard Cirrus CI artifact URL format
		url = fmt.Sprintf(ArtifactURLFormat, taskID, filename)
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s, 8s, 16s
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * c.baseDelay
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("fetch artifact: %w", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Check read error first
		if readErr != nil {
			lastErr = fmt.Errorf("read body: %w", readErr)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("artifact not found: %s", filename)
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("artifact fetch error: status %d", resp.StatusCode)
			continue
		}

		return body, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// GetArtifactURL returns the URL for downloading an artifact
func GetArtifactURL(taskID, filename string) string {
	return fmt.Sprintf(ArtifactURLFormat, taskID, filename)
}
