package parser

import (
	"testing"
	"time"
)

func TestTestResultString(t *testing.T) {
	tr := TestResult{
		Name:      "Basic ops",
		Suite:     "run basic podman commands",
		Framework: "ginkgo",
		Status:    StatusPassed,
		BuildID:   "123456",
		TaskName:  "sys fedora rootless",
		Platform:  "sys fedora rootless",
		Timestamp: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
	}

	expected := "ginkgo: run basic podman commands/Basic ops [passed]"
	if tr.String() != expected {
		t.Errorf("got %q, want %q", tr.String(), expected)
	}
}

func TestTestResultPlatformKey(t *testing.T) {
	tr := TestResult{
		Name:     "Basic ops",
		Suite:    "run basic podman commands",
		Platform: "sys fedora rootless",
	}

	expected := "[sys fedora rootless] run basic podman commands/Basic ops"
	if tr.PlatformKey() != expected {
		t.Errorf("got %q, want %q", tr.PlatformKey(), expected)
	}
}

func TestTestResultIsFlakeCandidate(t *testing.T) {
	passed := TestResult{Status: StatusPassed}
	failed := TestResult{Status: StatusFailed}
	skipped := TestResult{Status: StatusSkipped}

	if !passed.IsFlakeCandidate() {
		t.Error("passed should be flake candidate")
	}
	if !failed.IsFlakeCandidate() {
		t.Error("failed should be flake candidate")
	}
	if skipped.IsFlakeCandidate() {
		t.Error("skipped should not be flake candidate")
	}
}
