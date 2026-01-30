package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(getVersion())
	},
}

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	// Check for version from go install
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	// Check for vcs revision
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

func init() {
	rootCmd.AddCommand(versionCmd)
}
