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
display dialog "MTG開始！\n%s" with title "oooooi" buttons {"あとで", "参加"} default button "参加" with icon caution
`, escapeAppleScript(title))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return AlertResultLater, nil
			}
		}
		return AlertResultError, fmt.Errorf("failed to show alert: %w", err)
	}

	result := strings.TrimSpace(string(output))
	if strings.Contains(result, "参加") {
		return AlertResultJoin, nil
	}
	return AlertResultLater, nil
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
