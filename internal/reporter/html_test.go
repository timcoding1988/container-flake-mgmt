package reporter

import (
	"strings"
	"testing"
	"time"

	"github.com/containers/container-flake-mgmt/internal/analyzer"
)

func TestGenerateHTML(t *testing.T) {
	report := &analyzer.Report{
		GeneratedAt: time.Date(2026, 3, 12, 10, 0, 0, 0, time.UTC),
		Repository:  "containers/podman",
		Branch:      "main",
		WindowDays:  30,
		TotalBuilds: 100,
		TotalTests:  50,
		FlakyCount:  5,
		Tests: []analyzer.TestStats{
			{
				Name:           "flaky test",
				Suite:          "test suite",
				Platform:       "sys fedora rootless",
				Framework:      "ginkgo",
				PassCount:      7,
				FailCount:      3,
				TotalRuns:      10,
				FlakinessPct:   30.0,
				Classification: analyzer.ClassificationHigh,
				DaysSinceFail:  2,
				FirstSeen:      time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
				LastSeen:       time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	html, err := GenerateHTML(report)
	if err != nil {
		t.Fatalf("GenerateHTML: %v", err)
	}

	checks := []string{
		"<!DOCTYPE html>",
		"containers/podman",
		"flaky test",
		"30.0%",
		"High",
		"sys fedora rootless",
		"Days Since Fail",
	}

	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Errorf("HTML should contain %q", check)
		}
	}
}
