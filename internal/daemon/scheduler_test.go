package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPIDFileWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "ooi.pid")
	wantPID := os.Getpid()

	// Write PID
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(wantPID)), 0644); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Read PID
	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	gotPID, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatalf("Failed to parse PID: %v", err)
	}

	if diff := cmp.Diff(wantPID, gotPID); diff != "" {
		t.Errorf("PID mismatch (-want +got):\n%s", diff)
	}
}

func TestSchedulerNotifiedEventsCleanup(t *testing.T) {
	tests := []struct {
		name          string
		eventCount    int
		wantCount     int
		shouldCleanup bool
	}{
		{
			name:          "cleanup when over 100",
			eventCount:    101,
			wantCount:     0,
			shouldCleanup: true,
		},
		{
			name:          "no cleanup when under 100",
			eventCount:    50,
			wantCount:     50,
			shouldCleanup: false,
		},
		{
			name:          "no cleanup at exactly 100",
			eventCount:    100,
			wantCount:     100,
			shouldCleanup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Scheduler{
				notifiedEvents: make(map[string]bool),
			}

			for i := range tt.eventCount {
				s.notifiedEvents[strconv.Itoa(i)] = true
			}

			s.cleanupOldEvents()

			if diff := cmp.Diff(tt.wantCount, len(s.notifiedEvents)); diff != "" {
				t.Errorf("notifiedEvents count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
