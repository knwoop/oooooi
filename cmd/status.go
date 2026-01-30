package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/knwoop/ooi/internal/calendar"
	"github.com/spf13/cobra"
)

const (
	statusLookBack  = 2 * time.Hour
	statusLookAhead = 3 * time.Hour
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show upcoming meetings",
	Long:  "Display upcoming meetings with Google Meet links.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		token, err := calendar.LoadToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Not authenticated. Run 'ooi auth' first.\n")
			os.Exit(1)
		}

		client, err := calendar.NewClient(ctx, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create calendar client: %v\n", err)
			os.Exit(1)
		}

		events, err := client.GetEventsInRange(ctx, statusLookBack, statusLookAhead)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch events: %v\n", err)
			os.Exit(1)
		}

		if len(events) == 0 {
			fmt.Println("No upcoming meetings with Google Meet.")
			return
		}

		now := time.Now()

		// Find ongoing and next meetings
		var ongoingEvent *calendar.Event
		var nextEvent *calendar.Event

		for i := range events {
			if events[i].StartTime.Before(now) || events[i].StartTime.Equal(now) {
				// Meeting has started (ongoing)
				if ongoingEvent == nil {
					ongoingEvent = &events[i]
				}
			} else {
				// Meeting hasn't started yet (upcoming)
				if nextEvent == nil {
					nextEvent = &events[i]
				}
			}
			// Stop once we have both
			if ongoingEvent != nil && nextEvent != nil {
				break
			}
		}

		if ongoingEvent == nil && nextEvent == nil {
			fmt.Println("No upcoming meetings with Google Meet.")
			return
		}

		if ongoingEvent != nil {
			printMeeting("Ongoing meeting", ongoingEvent, "[ongoing]")
		}

		if nextEvent != nil {
			if ongoingEvent != nil {
				fmt.Println()
			}
			printMeeting("Next meeting", nextEvent, formatEventStatus(nextEvent.StartTime, now))
		}
	},
}

func printMeeting(label string, event *calendar.Event, timeStatus string) {
	const tmpl = `%s:
  Title:  %s
  Time:   %s %s
  Status: %s
  Meet:   %s`
	fmt.Printf(tmpl+"\n", label, event.Title, event.StartTime.Format("15:04"), timeStatus, event.ResponseStatus, event.MeetLink)
}

func formatEventStatus(startTime, now time.Time) string {
	until := startTime.Sub(now)

	if until < 0 {
		return "[ongoing]"
	}

	h := int(until.Hours())
	m := int(until.Minutes()) % 60

	if h > 0 {
		return fmt.Sprintf("[in %dh %dm]", h, m)
	}
	return fmt.Sprintf("[in %dm]", m)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
