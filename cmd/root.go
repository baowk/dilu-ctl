package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dilu-ctl",
	Short: "Dilu 项目快速创建和代码生成工具",
	Long: `dilu-ctl 是一个用于快速创建和管理 Dilu 项目的命令行工具。

主要功能:
  1. 创建新的 dilu 项目 (基础版或完整版)
  2. 为已有项目生成模块代码 (基于数据库表结构)`,
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// 添加全局标志
	rootCmd.AddCommand(createProjectCmd)
	rootCmd.AddCommand(genCmd)
}
