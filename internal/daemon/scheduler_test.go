package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestPIDFileWriteAndRead(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "ooi.pid")
	wantPID := os.Getpid()

	// Write PID
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(wantPID)), 0o644); err != nil {
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
		name       string
		eventCount int
		wantCount  int
	}{
		{
			name:       "cleanup when over 100",
			eventCount: 101,
			wantCount:  0,
		},
		{
			name:       "no cleanup when under 100",
			eventCount: 50,
			wantCount:  50,
		},
		{
			name:       "no cleanup at exactly 100",
			eventCount: 100,
			wantCount:  100,
		},
	}

	baseTime := time.Now()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Scheduler{
				notifiedEvents: make(map[eventKey]bool),
			}

			for i := range tt.eventCount {
				key := eventKey{
					eventID:   strconv.Itoa(i),
					startTime: baseTime.Add(time.Duration(i) * time.Minute),
				}
				s.notifiedEvents[key] = true
			}

			s.cleanupOldEvents()

			if diff := cmp.Diff(tt.wantCount, len(s.notifiedEvents)); diff != "" {
				t.Errorf("notifiedEvents count mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEventKeyWithRescheduledMeeting(t *testing.T) {
	baseTime := time.Now()
	originalTime := baseTime.Add(10 * time.Minute)
	rescheduledTime := baseTime.Add(30 * time.Minute)

	s := &Scheduler{
		notifiedEvents: make(map[eventKey]bool),
	}

	// Original meeting notified
	originalKey := eventKey{
		eventID:   "meeting-123",
		startTime: originalTime,
	}
	s.notifiedEvents[originalKey] = true

	// Rescheduled meeting should have different key
	rescheduledKey := eventKey{
		eventID:   "meeting-123",
		startTime: rescheduledTime,
	}

	// Original should be marked as notified
	if !s.notifiedEvents[originalKey] {
		t.Error("Original meeting should be marked as notified")
	}

	// Rescheduled should NOT be marked as notified
	if s.notifiedEvents[rescheduledKey] {
		t.Error("Rescheduled meeting should NOT be marked as notified")
	}
}
