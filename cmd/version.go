package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示 dilu-ctl 版本",
	Run: func(cmd *cobra.Command, args []string) {
		v := Version
		if v == "" {
			v = "dev"
		}
		fmt.Println(v)
	},
}
