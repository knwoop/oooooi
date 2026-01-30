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
	// AppleScript buttons are limited to 3, use first 3 meetings
	maxButtons := 3
	if len(meetings) < maxButtons {
		maxButtons = len(meetings)
	}

	// Build button list (AppleScript shows buttons right-to-left, so reverse order)
	var buttons []string
	for i := maxButtons - 1; i >= 0; i-- {
		buttons = append(buttons, fmt.Sprintf("\"%s\"", escapeAppleScript(meetings[i].Title)))
	}

	script := fmt.Sprintf(`
display dialog "Meeting starting!" with title "ooi" buttons {%s} default button 1 with icon caution
set selectedButton to button returned of result
return selectedButton
`, strings.Join(buttons, ", "))

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
