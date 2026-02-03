package menubar

import (
	"context"
	"fmt"
	"time"

	"fyne.io/systray"
	"github.com/knwoop/ooi/internal/calendar"
	"github.com/knwoop/ooi/internal/notifier"
)

type EventProvider interface {
	GetOngoingEvent() *calendar.Event
	GetNextEvent() *calendar.Event
	Sync()
}

func Run(ctx context.Context, provider EventProvider) {
	systray.Run(func() { onReady(ctx, provider) }, onExit)
}

func onReady(ctx context.Context, provider EventProvider) {
	systray.SetTitle("📅 No meetings")
	systray.SetTooltip("ooi - Meeting Reminder")

	mMeetingInfo := systray.AddMenuItem("No meetings", "Current meeting info")
	mMeetingInfo.Disable()

	systray.AddSeparator()

	mOpenMeet := systray.AddMenuItem("Open Meet", "Open Google Meet link")
	mOpenMeet.Disable()

	systray.AddSeparator()

	mSync := systray.AddMenuItem("Sync", "Sync calendar")
	mQuit := systray.AddMenuItem("Quit", "Quit ooi")

	var currentMeetLink string

	// Start update ticker
	ticker := time.NewTicker(1 * time.Second)

	// Update display function
	update := func() {
		ongoing := provider.GetOngoingEvent()
		next := provider.GetNextEvent()
		updateDisplay(ongoing, next, mMeetingInfo, mOpenMeet, &currentMeetLink)
	}

	go func() {
		defer ticker.Stop()

		// Initial update after short delay for scheduler to fetch
		time.Sleep(500 * time.Millisecond)
		update()

		for {
			select {
			case <-ctx.Done():
				systray.Quit()
				return
			case <-ticker.C:
				update()
			case <-mSync.ClickedCh:
				provider.Sync()
			case <-mOpenMeet.ClickedCh:
				if currentMeetLink != "" {
					notifier.OpenMeetLink(currentMeetLink)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Cleanup if needed
}

func updateDisplay(ongoing, next *calendar.Event, mInfo, mOpenMeet *systray.MenuItem, meetLink *string) {
	if ongoing != nil {
		remaining := time.Until(ongoing.EndTime)
		mins := int(remaining.Minutes())
		if mins < 0 {
			mins = 0
		}
		title := truncateTitle(ongoing.Title, 20)
		systray.SetTitle(fmt.Sprintf("🟢 %dm %s", mins, title))
		mInfo.SetTitle(fmt.Sprintf("Ongoing: %s (%dm remaining)", ongoing.Title, mins))
		mInfo.Enable()
		*meetLink = ongoing.MeetLink
		mOpenMeet.Enable()
		return
	}

	if next != nil {
		until := time.Until(next.StartTime)
		mins := int(until.Minutes())
		if mins < 0 {
			mins = 0
		}
		title := truncateTitle(next.Title, 20)
		systray.SetTitle(fmt.Sprintf("⏳ %dm %s", mins, title))
		mInfo.SetTitle(fmt.Sprintf("Next: %s (in %dm)", next.Title, mins))
		mInfo.Enable()
		*meetLink = next.MeetLink
		mOpenMeet.Enable()
		return
	}

	systray.SetTitle("📅 No meetings")
	mInfo.SetTitle("No meetings")
	mInfo.Disable()
	*meetLink = ""
	mOpenMeet.Disable()
}

func truncateTitle(title string, maxLen int) string {
	runes := []rune(title)
	if len(runes) <= maxLen {
		return title
	}
	return string(runes[:maxLen-1]) + "…"
}
