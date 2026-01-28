package calendar

import (
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestResponseStatusPriority(t *testing.T) {
	want := map[string]int{
		"accepted":    0,
		"tentative":   1,
		"needsAction": 2,
		"declined":    3,
	}

	if diff := cmp.Diff(want, responseStatusPriority); diff != "" {
		t.Errorf("responseStatusPriority mismatch (-want +got):\n%s", diff)
	}
}

func TestEventSortByResponseStatus(t *testing.T) {
	now := time.Now()
	sameTime := now.Add(10 * time.Minute)

	events := []Event{
		{ID: "1", Title: "NeedsAction", StartTime: sameTime, ResponseStatus: "needsAction"},
		{ID: "2", Title: "Accepted", StartTime: sameTime, ResponseStatus: "accepted"},
		{ID: "3", Title: "Tentative", StartTime: sameTime, ResponseStatus: "tentative"},
	}

	// Sort using same logic as GetUpcomingEvents
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartTime.Equal(events[j].StartTime) {
			return responseStatusPriority[events[i].ResponseStatus] < responseStatusPriority[events[j].ResponseStatus]
		}
		return events[i].StartTime.Before(events[j].StartTime)
	})

	wantOrder := []string{"accepted", "tentative", "needsAction"}
	gotOrder := make([]string, len(events))
	for i, e := range events {
		gotOrder[i] = e.ResponseStatus
	}

	if diff := cmp.Diff(wantOrder, gotOrder); diff != "" {
		t.Errorf("event order mismatch (-want +got):\n%s", diff)
	}
}

func TestEventSortByStartTimeThenStatus(t *testing.T) {
	now := time.Now()
	time1 := now.Add(10 * time.Minute)
	time2 := now.Add(20 * time.Minute)

	events := []Event{
		{ID: "1", Title: "Later-Accepted", StartTime: time2, ResponseStatus: "accepted"},
		{ID: "2", Title: "Earlier-Tentative", StartTime: time1, ResponseStatus: "tentative"},
		{ID: "3", Title: "Earlier-Accepted", StartTime: time1, ResponseStatus: "accepted"},
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].StartTime.Equal(events[j].StartTime) {
			return responseStatusPriority[events[i].ResponseStatus] < responseStatusPriority[events[j].ResponseStatus]
		}
		return events[i].StartTime.Before(events[j].StartTime)
	})

	wantTitles := []string{"Earlier-Accepted", "Earlier-Tentative", "Later-Accepted"}
	gotTitles := make([]string, len(events))
	for i, e := range events {
		gotTitles[i] = e.Title
	}

	if diff := cmp.Diff(wantTitles, gotTitles); diff != "" {
		t.Errorf("event order mismatch (-want +got):\n%s", diff)
	}
}
