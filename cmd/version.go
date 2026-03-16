package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示 dilu-ctl 版本",
	Run: func(cmd *cobra.Command, args []string) {
		v := resolveVersion()
		fmt.Println(v)
	},
}

func resolveVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	if Version == "" {
		return "dev"
	}
	return Version
}
