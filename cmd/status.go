package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/knwoop/ooi/internal/calendar"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show next upcoming meeting",
	Long:  "Display the next scheduled meeting with Google Meet link.",
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

		event, err := client.GetNextMeetEvent(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch events: %v\n", err)
			os.Exit(1)
		}

		if event == nil {
			fmt.Println("No upcoming meetings with Google Meet.")
			return
		}

		until := time.Until(event.StartTime)
		fmt.Printf("Next meeting:\n")
		fmt.Printf("  Title:  %s\n", event.Title)
		fmt.Printf("  Start:  %s\n", event.StartTime.Format("2006-01-02 15:04"))
		fmt.Printf("  In:     %s\n", formatDuration(until))
		fmt.Printf("  Status: %s\n", event.ResponseStatus)
		fmt.Printf("  Meet:   %s\n", event.MeetLink)
	},
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
