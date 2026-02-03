package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/knwoop/ooi/internal/calendar"
	"github.com/knwoop/ooi/internal/daemon"
	"github.com/knwoop/ooi/internal/menubar"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "ooi",
	Short:   "Meeting reminder CLI tool",
	Long:    "ooi is a macOS CLI tool that automatically opens Google Meet 1 minute before meetings.",
	Version: getVersion(),
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigCh
			log.Println("Received shutdown signal")
			cancel()
		}()

		log.Println("Starting ooi daemon...")

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

		scheduler := daemon.NewScheduler(client)

		// Run scheduler in background
		go func() {
			if err := scheduler.Run(ctx); err != nil {
				if err != context.Canceled {
					log.Printf("Scheduler error: %v", err)
				}
			}
		}()

		// Run systray on main thread (required by systray library)
		menubar.Run(ctx, scheduler)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	if revision != "" {
		v := revision[:7]
		if modified == "true" {
			v += "-dirty"
		}
		return v
	}

	return "dev"
}
