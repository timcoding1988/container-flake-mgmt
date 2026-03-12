package parser

import (
	"os"
	"testing"
)

func TestParseBATS(t *testing.T) {
	html, err := os.ReadFile("../../testdata/sample_bats.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	results, err := ParseHTML(html, "bats")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	statusCount := map[Status]int{}
	for _, r := range results {
		statusCount[r.Status]++
	}

	if statusCount[StatusPassed] != 1 {
		t.Errorf("expected 1 passed, got %d", statusCount[StatusPassed])
	}
	if statusCount[StatusFailed] != 1 {
		t.Errorf("expected 1 failed, got %d", statusCount[StatusFailed])
	}
	if statusCount[StatusSkipped] != 1 {
		t.Errorf("expected 1 skipped, got %d", statusCount[StatusSkipped])
	}
}

func TestParseGinkgo(t *testing.T) {
	html, err := os.ReadFile("../../testdata/sample_ginkgo.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	results, err := ParseHTML(html, "ginkgo")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Should find: Basic ops (pass), machine stop/start (fail), setup failed (fail),
	// teardown failed (fail), advanced networking (skip) = 5 results
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}

	// Check that BeforeEach/AfterEach failures are captured
	phases := map[string]int{}
	for _, r := range results {
		phases[r.Phase]++
	}

	if phases["BeforeEach"] != 1 {
		t.Errorf("expected 1 BeforeEach, got %d", phases["BeforeEach"])
	}
	if phases["AfterEach"] != 1 {
		t.Errorf("expected 1 AfterEach, got %d", phases["AfterEach"])
	}
	if phases["It"] != 3 {
		t.Errorf("expected 3 It, got %d", phases["It"])
	}
}

func TestParseEmptyHTML(t *testing.T) {
	results, err := ParseHTML([]byte("<html><body></body></html>"), "bats")
	if err != nil {
		t.Fatalf("parse empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
