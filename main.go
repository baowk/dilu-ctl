// Package main 实现了一个用于快速创建和管理 Dilu 项目的命令行工具。
//
// dilu-ctl - CLI tool for creating dilu projects and generating module code
//
// 主要功能:
// 1. 创建新的 dilu 项目 (基础版或完整版)
// 2. 为已有项目生成模块代码 (-gen 命令)
//
// 使用示例:
//   # 创建新项目
//   ./dilu-ctl create -n myproject
//   ./dilu-ctl create -n myproject -a  # 包含 admin 前端
//
//   # 生成模块代码
//   ./dilu-ctl gen -d sys -t sys_user -dns 'user:pass@tcp(localhost:3306)/sys'
package main

import (
	"github.com/baowk/dilu-ctl/cmd"
)

func main() {
	cmd.Execute()
}
