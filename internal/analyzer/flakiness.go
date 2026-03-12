package analyzer

import (
	"sort"
	"time"

	"github.com/containers/container-flake-mgmt/internal/parser"
)

// Classification represents the flakiness severity
type Classification string

const (
	ClassificationStable Classification = "stable"
	ClassificationLow    Classification = "low"
	ClassificationMedium Classification = "medium"
	ClassificationHigh   Classification = "high"
	ClassificationBroken Classification = "broken"
)

// TestStats holds aggregated statistics for a single test on a single platform
type TestStats struct {
	Name           string         `json:"name"`
	Suite          string         `json:"suite"`
	Platform       string         `json:"platform"`
	Framework      string         `json:"framework"`
	PassCount      int            `json:"pass_count"`
	FailCount      int            `json:"fail_count"`
	TotalRuns      int            `json:"total_runs"`
	FlakinessPct   float64        `json:"flakiness_pct"`
	Classification Classification `json:"classification"`
	FirstSeen      time.Time      `json:"first_seen"`
	LastSeen       time.Time      `json:"last_seen"`
	LastFailure    *time.Time     `json:"last_failure,omitempty"`
	DaysSinceFail  int            `json:"days_since_fail,omitempty"`
	FailingBuilds  []string       `json:"failing_builds,omitempty"`
}

// Classify returns the flakiness classification based on pass/fail ratio
func (ts *TestStats) Classify() Classification {
	if ts.PassCount == 0 && ts.FailCount > 0 {
		return ClassificationBroken
	}
	if ts.FailCount == 0 {
		return ClassificationStable
	}

	pct := ts.FlakinessPct
	switch {
	case pct >= 30:
		return ClassificationHigh
	case pct > 10:
		return ClassificationMedium
	case pct > 0:
		return ClassificationLow
	default:
		return ClassificationStable
	}
}

// IsFlaky returns true if the test has both passes and failures
func (ts *TestStats) IsFlaky() bool {
	return ts.PassCount > 0 && ts.FailCount > 0
}

// Report contains the complete flakiness analysis
type Report struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Repository  string      `json:"repository"`
	Branch      string      `json:"branch"`
	WindowDays  int         `json:"window_days"`
	TotalBuilds int         `json:"total_builds"`
	TotalTests  int         `json:"total_tests"`
	FlakyCount  int         `json:"flaky_count"`
	Tests       []TestStats `json:"tests"`
}

// Analyze processes test results and generates a flakiness report
// Groups tests by platform to avoid misclassifying platform-specific failures as flakiness
func Analyze(results []parser.TestResult) *Report {
	// Group results by platform-qualified key
	testMap := make(map[string]*TestStats)

	for _, r := range results {
		if !r.IsFlakeCandidate() {
			continue
		}

		// Use platform-aware key to separate results by environment
		key := r.PlatformKey()
		stats, exists := testMap[key]
		if !exists {
			stats = &TestStats{
				Name:      r.Name,
				Suite:     r.Suite,
				Platform:  r.Platform,
				Framework: r.Framework,
				FirstSeen: r.Timestamp,
				LastSeen:  r.Timestamp,
			}
			testMap[key] = stats
		}

		// Update counts
		switch r.Status {
		case parser.StatusPassed:
			stats.PassCount++
		case parser.StatusFailed:
			stats.FailCount++
			if stats.LastFailure == nil || r.Timestamp.After(*stats.LastFailure) {
				t := r.Timestamp
				stats.LastFailure = &t
			}
			if r.BuildID != "" {
				stats.FailingBuilds = append(stats.FailingBuilds, r.BuildID)
			}
		}

		// Update timestamps
		if r.Timestamp.Before(stats.FirstSeen) {
			stats.FirstSeen = r.Timestamp
		}
		if r.Timestamp.After(stats.LastSeen) {
			stats.LastSeen = r.Timestamp
		}
	}

	// Build report
	report := &Report{
		GeneratedAt: time.Now(),
		Tests:       make([]TestStats, 0, len(testMap)),
	}

	for _, stats := range testMap {
		stats.TotalRuns = stats.PassCount + stats.FailCount
		if stats.TotalRuns > 0 {
			stats.FlakinessPct = float64(stats.FailCount) / float64(stats.TotalRuns) * 100
		}
		stats.Classification = stats.Classify()

		// Calculate days since last failure
		if stats.LastFailure != nil {
			stats.DaysSinceFail = int(time.Since(*stats.LastFailure).Hours() / 24)
		}

		report.Tests = append(report.Tests, *stats)
		report.TotalTests++
		if stats.IsFlaky() {
			report.FlakyCount++
		}
	}

	// Sort by flakiness (highest first), then by name
	sort.Slice(report.Tests, func(i, j int) bool {
		if report.Tests[i].FlakinessPct != report.Tests[j].FlakinessPct {
			return report.Tests[i].FlakinessPct > report.Tests[j].FlakinessPct
		}
		return report.Tests[i].Name < report.Tests[j].Name
	})

	return report
}
