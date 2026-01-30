package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/knwoop/ooi/internal/calendar"
	"github.com/knwoop/ooi/internal/notifier"
)

const (
	fetchInterval   = 3 * time.Minute
	alertInterval   = 1 * time.Second
	notifyBefore    = 1 * time.Minute
	lookAheadWindow = 5 * time.Minute
)

type eventKey struct {
	eventID   string
	startTime time.Time
}

type Scheduler struct {
	client         *calendar.Client
	cachedEvents   []calendar.Event
	cacheMu        sync.RWMutex
	notifiedEvents map[eventKey]bool
}

func NewScheduler(client *calendar.Client) *Scheduler {
	return &Scheduler{
		client:         client,
		notifiedEvents: make(map[eventKey]bool),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	log.Printf("Scheduler started (fetch: %v, alert check: %v)", fetchInterval, alertInterval)

	// Write PID file
	if err := writePIDFile(); err != nil {
		log.Printf("Warning: failed to write PID file: %v", err)
	}
	defer removePIDFile()

	// Initial fetch
	s.fetchEvents(ctx)

	// Listen for SIGUSR1 to trigger immediate fetch
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1)

	fetchTicker := time.NewTicker(fetchInterval)
	alertTicker := time.NewTicker(alertInterval)
	defer fetchTicker.Stop()
	defer alertTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return ctx.Err()
		case <-sigCh:
			log.Println("Received SIGUSR1, syncing...")
			s.fetchEvents(ctx)
		case <-fetchTicker.C:
			s.fetchEvents(ctx)
		case <-alertTicker.C:
			s.checkAlerts()
		}
	}
}

func (s *Scheduler) fetchEvents(ctx context.Context) {
	events, err := s.client.GetUpcomingEvents(ctx, lookAheadWindow)
	if err != nil {
		log.Printf("Failed to fetch events: %v", err)
		return
	}

	s.cacheMu.Lock()
	s.cachedEvents = events
	s.cacheMu.Unlock()

	log.Printf("Fetched %d events", len(events))
}

func (s *Scheduler) checkAlerts() {
	s.cacheMu.RLock()
	events := s.cachedEvents
	s.cacheMu.RUnlock()

	now := time.Now()

	// Collect all events that need notification
	var eventsToNotify []calendar.Event
	for _, event := range events {
		key := eventKey{
			eventID:   event.ID,
			startTime: event.StartTime,
		}

		if s.notifiedEvents[key] {
			continue
		}

		timeUntil := event.StartTime.Sub(now)

		if timeUntil <= notifyBefore && timeUntil > -notifyBefore {
			eventsToNotify = append(eventsToNotify, event)
		}
	}

	if len(eventsToNotify) > 0 {
		s.notifyMultiple(eventsToNotify)
		// Mark all as notified
		for _, event := range eventsToNotify {
			key := eventKey{
				eventID:   event.ID,
				startTime: event.StartTime,
			}
			s.notifiedEvents[key] = true
		}
	}

	s.cleanupOldEvents()
}

func (s *Scheduler) notifyMultiple(events []calendar.Event) {
	for _, event := range events {
		log.Printf("Notifying: %s (starts at %s)", event.Title, event.StartTime.Format("15:04"))
	}

	// Convert to notifier.Meeting slice
	meetings := make([]notifier.Meeting, len(events))
	for i, event := range events {
		meetings[i] = notifier.Meeting{
			Title:    event.Title,
			MeetLink: event.MeetLink,
		}
	}

	result, err := notifier.ShowMeetingAlert(meetings)
	if err != nil {
		log.Printf("Failed to show alert: %v", err)
		return
	}

	if result.Joined && result.Index >= 0 && result.Index < len(events) {
		selectedEvent := events[result.Index]
		log.Printf("Opening Meet: %s", selectedEvent.MeetLink)
		if err := notifier.OpenMeetLink(selectedEvent.MeetLink); err != nil {
			log.Printf("Failed to open Meet link: %v", err)
		}
	} else {
		log.Printf("User cancelled or closed the dialog")
	}
}

func (s *Scheduler) cleanupOldEvents() {
	if len(s.notifiedEvents) > 100 {
		s.notifiedEvents = make(map[eventKey]bool)
	}
}

func PIDFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ooi", "ooi.pid"), nil
}

func writePIDFile() error {
	path, err := PIDFilePath()
	if err != nil {
		return err
	}
	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

func removePIDFile() {
	path, err := PIDFilePath()
	if err != nil {
		return
	}
	os.Remove(path)
}

func ReadPID() (int, error) {
	path, err := PIDFilePath()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

func Start(ctx context.Context) error {
	token, err := calendar.LoadToken()
	if err != nil {
		return fmt.Errorf("not authenticated, run 'ooi auth' first: %w", err)
	}

	client, err := calendar.NewClient(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to create calendar client: %w", err)
	}

	scheduler := NewScheduler(client)
	return scheduler.Run(ctx)
}
