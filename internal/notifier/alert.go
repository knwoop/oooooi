package notifier

import (
	"fmt"
	"os/exec"
	"strings"
)

type AlertResult int

const (
	AlertResultJoin AlertResult = iota
	AlertResultLater
	AlertResultError
)

func ShowMeetingAlert(title string) (AlertResult, error) {
	script := fmt.Sprintf(`
display dialog "Meeting starting!\n%s" with title "ooi" buttons {"Join"} default button "Join" with icon caution
`, escapeAppleScript(title))

	cmd := exec.Command("osascript", "-e", script)
	_, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// User closed dialog without clicking button
				return AlertResultLater, nil
			}
		}
		return AlertResultError, fmt.Errorf("failed to show alert: %w", err)
	}

	return AlertResultJoin, nil
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
