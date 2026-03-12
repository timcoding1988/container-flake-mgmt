package cirrus

import (
	"strings"
	"time"
)

// Build represents a Cirrus CI build
type Build struct {
	ID              string `json:"id"`
	Branch          string `json:"branch"`
	ChangeIDInRepo  string `json:"changeIdInRepo"` // commit SHA
	ChangeTimestamp int64  `json:"changeTimestamp"` // milliseconds since epoch
	Status          string `json:"status"`
}

// IsRecent returns true if the build is within the given duration
func (b Build) IsRecent(window time.Duration) bool {
	buildTime := time.UnixMilli(b.ChangeTimestamp)
	return time.Since(buildTime) <= window
}

// Timestamp returns the build time as time.Time
func (b Build) Timestamp() time.Time {
	return time.UnixMilli(b.ChangeTimestamp)
}

// Task represents a Cirrus CI task within a build
type Task struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	DurationInSeconds int    `json:"durationInSeconds"`
}

// IsTestTask returns true if this task runs tests (vs build/validate tasks)
// Podman CI task naming: "sys ...", "int ...", "bindings_test", etc.
func (t Task) IsTestTask() bool {
	name := strings.ToLower(t.Name)
	return strings.HasPrefix(name, "sys ") ||
		strings.HasPrefix(name, "int ") ||
		strings.Contains(name, "_test") ||
		strings.Contains(name, "test_")
}

// TaskStatus constants for Cirrus CI task statuses
const (
	TaskStatusCompleted = "COMPLETED"
	TaskStatusFailed    = "FAILED"
	TaskStatusAborted   = "ABORTED"
)

// IsFinished returns true if the task has completed (success or failure)
func (t Task) IsFinished() bool {
	return t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed
}

// ArtifactRequest represents a request to fetch an artifact
type ArtifactRequest struct {
	TaskID   string
	TaskName string
	BuildID  string
	Commit   string
	Time     time.Time
}
