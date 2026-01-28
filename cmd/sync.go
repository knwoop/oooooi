package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/knwoop/ooi/internal/daemon"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Trigger immediate calendar sync",
	Long:  "Send signal to daemon to fetch events from Google Calendar immediately.",
	Run: func(cmd *cobra.Command, args []string) {
		pid, err := daemon.ReadPID()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Daemon not running or PID file not found: %v\n", err)
			os.Exit(1)
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to find process: %v\n", err)
			os.Exit(1)
		}

		if err := process.Signal(syscall.SIGUSR1); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send signal: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Sync signal sent to daemon.")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
