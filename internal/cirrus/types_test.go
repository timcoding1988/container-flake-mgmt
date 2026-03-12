package cirrus

import (
	"testing"
	"time"
)

func TestBuildIsRecent(t *testing.T) {
	now := time.Now()

	recent := Build{
		ChangeTimestamp: now.Add(-24 * time.Hour).UnixMilli(),
	}
	old := Build{
		ChangeTimestamp: now.Add(-60 * 24 * time.Hour).UnixMilli(),
	}

	if !recent.IsRecent(30 * 24 * time.Hour) {
		t.Error("1-day-old build should be recent within 30 days")
	}
	if old.IsRecent(30 * 24 * time.Hour) {
		t.Error("60-day-old build should not be recent within 30 days")
	}
}

func TestTaskIsTestTask(t *testing.T) {
	tests := []struct {
		name     string
		taskName string
		want     bool
	}{
		{"sys test", "sys fedora rootless", true},
		{"int test", "int podman fedora amd64", true},
		{"bindings", "bindings_test", true},
		{"build task", "Build x86_64", false},
		{"validate", "Validate", false},
	}

	for _, tt := range tests {
		task := Task{Name: tt.taskName}
		got := task.IsTestTask()
		if got != tt.want {
			t.Errorf("Task(%q).IsTestTask() = %v, want %v", tt.taskName, got, tt.want)
		}
	}
}
