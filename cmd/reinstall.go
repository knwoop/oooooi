package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/knwoop/ooi/internal/launchd"
	"github.com/spf13/cobra"
)

var reinstallCmd = &cobra.Command{
	Use:   "reinstall",
	Short: "Rebuild and restart the launchd service",
	Long:  "Rebuild the binary and restart the launchd service to apply changes.",
	Run: func(cmd *cobra.Command, args []string) {
		if !launchd.IsInstalled() {
			fmt.Println("Service is not installed. Run 'ooi install' first.")
			return
		}

		// Regenerate plist
		fmt.Println("Updating plist...")
		if err := launchd.WritePlist(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update plist: %v\n", err)
			os.Exit(1)
		}

		// Restart service
		fmt.Println("Restarting service...")
		uid := os.Getuid()
		kickstartCmd := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("gui/%d/%s", uid, launchd.Label))
		if err := kickstartCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restart service: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Done! Service restarted.")
	},
}

func init() {
	rootCmd.AddCommand(reinstallCmd)
}
