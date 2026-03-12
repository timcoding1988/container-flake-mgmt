package parser

import (
	"fmt"
	"time"
)

// Status represents the outcome of a test
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// TestResult represents a single test execution result
type TestResult struct {
	// Test identification
	Name      string `json:"name"`
	Suite     string `json:"suite"`
	Framework string `json:"framework"` // "bats", "ginkgo"
	Phase     string `json:"phase"`     // "It", "BeforeEach", "AfterEach", etc.

	// Result
	Status   Status  `json:"status"`
	Duration float64 `json:"duration_seconds,omitempty"`

	// Build context
	BuildID   string    `json:"build_id"`
	TaskID    string    `json:"task_id"`
	TaskName  string    `json:"task_name"`
	Platform  string    `json:"platform"` // Normalized task name for grouping
	CommitSHA string    `json:"commit_sha"`
	Timestamp time.Time `json:"timestamp"`
}

// String returns a human-readable representation
func (tr TestResult) String() string {
	return fmt.Sprintf("%s: %s/%s [%s]", tr.Framework, tr.Suite, tr.Name, tr.Status)
}

// IsFlakeCandidate returns true if this result should be included in flakiness analysis
// Skipped tests are not candidates since they don't indicate pass/fail behavior
func (tr TestResult) IsFlakeCandidate() bool {
	return tr.Status == StatusPassed || tr.Status == StatusFailed
}

// FullName returns the fully qualified test name (suite + name)
func (tr TestResult) FullName() string {
	if tr.Suite == "" {
		return tr.Name
	}
	return tr.Suite + "/" + tr.Name
}

// PlatformKey returns a unique key that includes platform context
// This prevents misclassifying platform-specific failures as flakiness
func (tr TestResult) PlatformKey() string {
	return fmt.Sprintf("[%s] %s", tr.Platform, tr.FullName())
}
