package analyzer

import (
	"testing"
	"time"

	"github.com/containers/container-flake-mgmt/internal/parser"
)

func TestAnalyzeWithPlatformGrouping(t *testing.T) {
	now := time.Now()

	results := []parser.TestResult{
		// Test passes on Linux but fails on Mac - NOT flaky, platform-specific
		{Name: "test1", Suite: "suite", Platform: "sys fedora", Status: parser.StatusPassed, Timestamp: now},
		{Name: "test1", Suite: "suite", Platform: "sys fedora", Status: parser.StatusPassed, Timestamp: now},
		{Name: "test1", Suite: "suite", Platform: "sys darwin", Status: parser.StatusFailed, Timestamp: now},
		{Name: "test1", Suite: "suite", Platform: "sys darwin", Status: parser.StatusFailed, Timestamp: now},

		// Actually flaky test - same platform, mixed results
		{Name: "flaky", Suite: "suite", Platform: "sys fedora", Status: parser.StatusPassed, Timestamp: now},
		{Name: "flaky", Suite: "suite", Platform: "sys fedora", Status: parser.StatusFailed, Timestamp: now},
	}

	report := Analyze(results)

	// Should have 3 test entries: test1@fedora, test1@darwin, flaky@fedora
	if len(report.Tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(report.Tests))
	}

	// Only "flaky" should be marked as flaky
	if report.FlakyCount != 1 {
		t.Errorf("expected 1 flaky test, got %d", report.FlakyCount)
	}

	// Find the flaky test and verify
	for _, ts := range report.Tests {
		if ts.Name == "flaky" {
			if !ts.IsFlaky() {
				t.Error("'flaky' test should be marked as flaky")
			}
		}
		if ts.Name == "test1" {
			if ts.IsFlaky() {
				t.Errorf("test1 on %s should NOT be flaky (platform-specific)", ts.Platform)
			}
		}
	}
}

func TestClassification(t *testing.T) {
	tests := []struct {
		passes int
		fails  int
		want   Classification
	}{
		{10, 0, ClassificationStable},
		{99, 1, ClassificationLow},
		{90, 10, ClassificationLow},
		{80, 20, ClassificationMedium},
		{70, 30, ClassificationHigh},
		{50, 50, ClassificationHigh},
		{0, 10, ClassificationBroken},
	}

	for _, tt := range tests {
		stats := TestStats{PassCount: tt.passes, FailCount: tt.fails}
		stats.FlakinessPct = float64(tt.fails) / float64(tt.passes+tt.fails) * 100
		got := stats.Classify()
		if got != tt.want {
			t.Errorf("Classify(%d pass, %d fail) = %s, want %s",
				tt.passes, tt.fails, got, tt.want)
		}
	}
}
