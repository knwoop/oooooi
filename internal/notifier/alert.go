package notifier

import (
	"fmt"
	"os/exec"
	"strings"
)

type AlertResult struct {
	Joined bool
	Index  int // Index of selected meeting (-1 if cancelled)
}

// Meeting represents a meeting for the alert dialog
type Meeting struct {
	Title    string
	MeetLink string
}

func ShowMeetingAlert(meetings []Meeting) (AlertResult, error) {
	if len(meetings) == 0 {
		return AlertResult{Joined: false, Index: -1}, nil
	}

	if len(meetings) == 1 {
		return showSingleMeetingAlert(meetings[0].Title)
	}

	return showMultipleMeetingsAlert(meetings)
}

func showSingleMeetingAlert(title string) (AlertResult, error) {
	script := fmt.Sprintf(`
display dialog "Meeting starting!\n%s" with title "ooi" buttons {"Join"} default button "Join" with icon caution
`, escapeAppleScript(title))

	cmd := exec.Command("osascript", "-e", script)
	_, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return AlertResult{Joined: false, Index: -1}, nil
			}
		}
		return AlertResult{Joined: false, Index: -1}, fmt.Errorf("failed to show alert: %w", err)
	}

	return AlertResult{Joined: true, Index: 0}, nil
}

func showMultipleMeetingsAlert(meetings []Meeting) (AlertResult, error) {
	// Build list of meeting titles
	var titles []string
	for _, m := range meetings {
		titles = append(titles, escapeAppleScript(m.Title))
	}

	script := fmt.Sprintf(`
set meetingList to {"%s"}
set selectedMeeting to choose from list meetingList with title "ooi" with prompt "Multiple meetings starting! Select one to join:" default items {item 1 of meetingList}
if selectedMeeting is false then
	return "CANCELLED"
else
	return item 1 of selectedMeeting
end if
`, strings.Join(titles, "\", \""))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return AlertResult{Joined: false, Index: -1}, nil
			}
		}
		return AlertResult{Joined: false, Index: -1}, fmt.Errorf("failed to show alert: %w", err)
	}

	selected := strings.TrimSpace(string(output))
	if selected == "CANCELLED" {
		return AlertResult{Joined: false, Index: -1}, nil
	}

	// Find the index of the selected meeting
	for i, m := range meetings {
		if m.Title == selected {
			return AlertResult{Joined: true, Index: i}, nil
		}
	}

	return AlertResult{Joined: true, Index: 0}, nil
}

func OpenMeetLink(url string) error {
	cmd := exec.Command("open", url)
	return cmd.Run()
}

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
