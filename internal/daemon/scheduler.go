package daemon

import (
	"context"
	"fmt"
	"log"
	"sync"
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

type Scheduler struct {
	client         *calendar.Client
	cachedEvents   []calendar.Event
	cacheMu        sync.RWMutex
	notifiedEvents map[string]bool
}

func NewScheduler(client *calendar.Client) *Scheduler {
	return &Scheduler{
		client:         client,
		notifiedEvents: make(map[string]bool),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	log.Printf("Scheduler started (fetch: %v, alert check: %v)", fetchInterval, alertInterval)

	// Initial fetch
	s.fetchEvents(ctx)

	fetchTicker := time.NewTicker(fetchInterval)
	alertTicker := time.NewTicker(alertInterval)
	defer fetchTicker.Stop()
	defer alertTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler stopped")
			return ctx.Err()
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

	for _, event := range events {
		if s.notifiedEvents[event.ID] {
			continue
		}

		timeUntil := event.StartTime.Sub(now)

		if timeUntil <= notifyBefore && timeUntil > -notifyBefore {
			s.notify(event)
			s.notifiedEvents[event.ID] = true
		}
	}

	s.cleanupOldEvents()
}

func (s *Scheduler) notify(event calendar.Event) {
	log.Printf("Notifying: %s (starts at %s)", event.Title, event.StartTime.Format("15:04"))

	result, err := notifier.ShowMeetingAlert(event.Title)
	if err != nil {
		log.Printf("Failed to show alert: %v", err)
		return
	}

	if result == notifier.AlertResultJoin {
		log.Printf("Opening Meet: %s", event.MeetLink)
		if err := notifier.OpenMeetLink(event.MeetLink); err != nil {
			log.Printf("Failed to open Meet link: %v", err)
		}
	} else {
		log.Printf("User chose 'Later' for: %s", event.Title)
	}
}

func (s *Scheduler) cleanupOldEvents() {
	if len(s.notifiedEvents) > 100 {
		s.notifiedEvents = make(map[string]bool)
	}
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
