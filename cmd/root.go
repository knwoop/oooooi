package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/knwoop/ooi/internal/daemon"
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
		if err := daemon.Start(ctx); err != nil {
			if err != context.Canceled {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
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
