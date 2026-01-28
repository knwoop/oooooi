package calendar

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// Response status priority (lower = higher priority)
var responseStatusPriority = map[string]int{
	"accepted":    0,
	"tentative":   1,
	"needsAction": 2,
	"declined":    3,
}

type Event struct {
	ID             string
	Title          string
	StartTime      time.Time
	MeetLink       string
	ResponseStatus string // accepted, tentative, needsAction, declined
}

type Client struct {
	service *calendar.Service
}

func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "ooi"), nil
}

func NewClient(ctx context.Context, token *oauth2.Token) (*Client, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	credentialsPath := filepath.Join(configDir, "credentials.json")
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials.json: %w", err)
	}

	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	client := config.Client(ctx, token)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return &Client{service: service}, nil
}

func (c *Client) GetUpcomingEvents(ctx context.Context, duration time.Duration) ([]Event, error) {
	now := time.Now()
	end := now.Add(duration)

	events, err := c.service.Events.List("primary").
		TimeMin(now.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}

	var result []Event
	for _, item := range events.Items {
		if item.HangoutLink == "" {
			continue
		}

		// Skip declined events
		responseStatus := getResponseStatus(item)
		if responseStatus == "declined" {
			continue
		}

		startTime, err := parseEventTime(item.Start)
		if err != nil {
			continue
		}

		result = append(result, Event{
			ID:             item.Id,
			Title:          item.Summary,
			StartTime:      startTime,
			MeetLink:       item.HangoutLink,
			ResponseStatus: responseStatus,
		})
	}

	// Sort by start time, then by response status priority
	sort.Slice(result, func(i, j int) bool {
		if result[i].StartTime.Equal(result[j].StartTime) {
			return responseStatusPriority[result[i].ResponseStatus] < responseStatusPriority[result[j].ResponseStatus]
		}
		return result[i].StartTime.Before(result[j].StartTime)
	})

	return result, nil
}

func getResponseStatus(event *calendar.Event) string {
	// Check if self is the organizer
	if event.Organizer != nil && event.Organizer.Self {
		return "accepted"
	}

	// Find self in attendees
	for _, attendee := range event.Attendees {
		if attendee.Self {
			return attendee.ResponseStatus
		}
	}

	// Default to needsAction if not found
	return "needsAction"
}

func (c *Client) GetNextMeetEvent(ctx context.Context) (*Event, error) {
	events, err := c.GetUpcomingEvents(ctx, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, nil
	}

	return &events[0], nil
}

func parseEventTime(eventTime *calendar.EventDateTime) (time.Time, error) {
	if eventTime.DateTime != "" {
		return time.Parse(time.RFC3339, eventTime.DateTime)
	}
	if eventTime.Date != "" {
		return time.Parse("2006-01-02", eventTime.Date)
	}
	return time.Time{}, fmt.Errorf("no valid time found")
}

func GetOAuthConfig() (*oauth2.Config, error) {
	configDir, err := ConfigDir()
	if err != nil {
		return nil, err
	}

	credentialsPath := filepath.Join(configDir, "credentials.json")
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials.json: %w", err)
	}

	return google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
}
