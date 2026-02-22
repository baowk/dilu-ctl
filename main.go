package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	var projectName string
	var useAll bool
	var outputPath string
	var useHTTPS bool
	var gitUsername string
	var help bool

	flag.StringVar(&projectName, "n", "", "项目名称")
	flag.BoolVar(&useAll, "a", false, "是否使用dilu-all仓库")
	flag.StringVar(&outputPath, "o", ".", "项目输出路径")
	flag.BoolVar(&useHTTPS, "https", false, "使用HTTPS协议而非SSH")
	flag.StringVar(&gitUsername, "u", "", "Git用户名（HTTPS模式下可选）")
	flag.BoolVar(&help, "h", false, "显示帮助信息")
	flag.BoolVar(&help, "help", false, "显示帮助信息")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  创建基础项目(仅dilu): ./dilu-ctl -n myproject")
		fmt.Println("  创建完整项目(dilu-all + admin): ./dilu-ctl -n myproject -a")
		fmt.Println("  创建基础项目(HTTPS): ./dilu-ctl -n myproject --https -u username")
		fmt.Println("  创建项目到指定目录: ./dilu-ctl -n myproject -o /path/to/output")
		fmt.Println("  创建完整项目(HTTPS): ./dilu-ctl -n myproject -a -o /path/to/output --https -u username")
	}

	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if projectName == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定项目名称 (-n)")
		flag.Usage()
		os.Exit(1)
	}

	// 解析输出路径
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无效的输出路径 '%s': %v\n", outputPath, err)
		os.Exit(1)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(absOutputPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法创建输出目录 '%s': %v\n", absOutputPath, err)
		os.Exit(1)
	}

	// 检查项目目录是否已存在
	projectPath := filepath.Join(absOutputPath, projectName)
	if _, err := os.Stat(projectPath); err == nil {
		fmt.Fprintf(os.Stderr, "错误: 项目目录 '%s' 已存在\n", projectPath)
		os.Exit(1)
	}

	// 确定要克隆的仓库URL
	mainRepoURL, adminRepoURL := getRepositoryURLs(useAll, useHTTPS, gitUsername)

	fmt.Printf("开始创建项目: %s\n", projectName)
	fmt.Printf("项目路径: %s\n", projectPath)
	fmt.Printf("使用协议: %s\n", getProtocolName(useHTTPS))
	fmt.Printf("主仓库: %s\n", mainRepoURL)
	if useAll {
		fmt.Printf("Admin仓库: %s\n", adminRepoURL)
	}

	// 创建项目目录
	if err := os.Mkdir(projectPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建目录失败: %v\n", err)
		os.Exit(1)
	}

	// 切换到项目目录
	oldDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前目录失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chdir(projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "切换目录失败: %v\n", err)
		os.Exit(1)
	}

	defer func() {
		// 恢复原目录
		os.Chdir(oldDir)
	}()

	// 检查Git是否可用
	if !isGitAvailable() {
		fmt.Fprintln(os.Stderr, "错误: 未找到Git命令，请先安装Git")
		os.Exit(1)
	}

	// 克隆主仓库
	fmt.Println("正在克隆主仓库...")
	if err := cloneRepository(mainRepoURL, ".", useHTTPS); err != nil {
		fmt.Fprintf(os.Stderr, "克隆主仓库失败: %v\n", err)
		os.Exit(1)
	}

	// 如果使用-a参数，同时克隆admin仓库
	adminProjectName := projectName + "-admin"
	adminProjectPath := filepath.Join(absOutputPath, adminProjectName)

	if useAll {
		fmt.Printf("正在克隆admin仓库到: %s\n", adminProjectName)

		// 检查admin目录是否已存在
		if _, err := os.Stat(adminProjectPath); err == nil {
			fmt.Fprintf(os.Stderr, "错误: admin项目目录 '%s' 已存在\n", adminProjectPath)
			os.Exit(1)
		}

		// 创建admin项目目录
		if err := os.Mkdir(adminProjectPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建admin目录失败: %v\n", err)
			os.Exit(1)
		}

		// 克隆admin仓库
		fmt.Printf("使用admin仓库: %s\n", adminRepoURL)

		if err := cloneRepository(adminRepoURL, adminProjectPath, useHTTPS); err != nil {
			fmt.Fprintf(os.Stderr, "克隆admin仓库失败: %v\n", err)
			// 不退出，因为主项目已经成功创建
			fmt.Println("警告: admin仓库克隆失败，但主项目创建成功")
		} else {
			// 在admin目录中重命名包名
			fmt.Println("正在重命名admin项目包名...")
			if err := os.Chdir(adminProjectPath); err != nil {
				fmt.Fprintf(os.Stderr, "切换到admin目录失败: %v\n", err)
			} else {
				if err := renamePackages(adminProjectName); err != nil {
					fmt.Printf("重命名admin包名失败: %v\n", err)
				}
				if err := updateGoMod(adminProjectName); err != nil {
					fmt.Printf("更新admin go.mod失败: %v\n", err)
				}
				// 清理admin的.git目录
				if err := os.RemoveAll(".git"); err != nil {
					fmt.Printf("清理admin .git目录失败: %v\n", err)
				}
				// 恢复到原目录
				os.Chdir(projectPath)
			}
		}
	}

	// 重命名主项目包名
	fmt.Println("正在重命名主项目包名...")
	if err := renamePackages(projectName); err != nil {
		fmt.Fprintf(os.Stderr, "重命名包名失败: %v\n", err)
		os.Exit(1)
	}

	// 更新主项目go.mod
	fmt.Println("正在更新主项目go.mod...")
	if err := updateGoMod(projectName); err != nil {
		fmt.Fprintf(os.Stderr, "更新go.mod失败: %v\n", err)
		os.Exit(1)
	}

	// 更新yaml配置文件中的front-path
	if useAll {
		fmt.Println("正在更新yaml配置文件...")
		if err := updateYamlFrontPath(projectName); err != nil {
			fmt.Printf("警告: 更新yaml配置失败: %v\n", err)
		}
	}

	// 移除主项目.git目录
	fmt.Println("正在清理主项目.git目录...")
	if err := os.RemoveAll(".git"); err != nil {
		fmt.Printf("警告: 清理主项目.git目录失败: %v\n", err)
		// 不退出，这只是清理工作
	}

	fmt.Printf("\n✅ 项目创建成功！\n")
	fmt.Printf("📁 主项目路径: %s\n", projectPath)
	if useAll {
		fmt.Printf("📁 Admin项目路径: %s\n", adminProjectPath)
	}
	fmt.Printf("🚀 请进入相应目录开始开发\n")
	if useHTTPS {
		fmt.Println("💡 提示: 如需避免重复输入密码，可配置Git凭证缓存:")
		fmt.Println("   git config --global credential.helper store")
	}
}

// getRepositoryURLs 根据参数获取仓库URL
func getRepositoryURLs(useAll, useHTTPS bool, username string) (mainURL, adminURL string) {
	if useHTTPS {
		if username != "" {
			if useAll {
				mainURL = fmt.Sprintf("https://github.com/%s/dilu-all.git", username)
			} else {
				mainURL = fmt.Sprintf("https://github.com/%s/dilu.git", username)
			}
			if strings.Contains(username, "/") {
				// 如果username包含组织名，如 "org/username"
				parts := strings.Split(username, "/")
				adminURL = fmt.Sprintf("https://github.com/%s/dilu-admin.git", parts[0])
			} else {
				adminURL = fmt.Sprintf("https://github.com/%s/dilu-admin.git", username)
			}
		} else {
			if useAll {
				mainURL = "https://github.com/baowk/dilu-all.git"
			} else {
				mainURL = "https://github.com/baowk/dilu.git"
			}
			adminURL = "https://github.com/baowk/dilu-admin.git"
		}
	} else {
		if useAll {
			mainURL = "git@github.com:baowk/dilu-all.git"
		} else {
			mainURL = "git@github.com:baowk/dilu.git"
		}
		adminURL = "git@github.com:baowk/dilu-admin.git"
	}
	return mainURL, adminURL
}

// getProtocolName 获取协议名称
func getProtocolName(useHTTPS bool) string {
	if useHTTPS {
		return "HTTPS"
	}
	return "SSH"
}

// cloneRepository 克隆仓库
func cloneRepository(repoURL, targetPath string, useHTTPS bool) error {
	var cmd *exec.Cmd

	if targetPath == "." {
		cmd = exec.Command("git", "clone", repoURL, ".")
	} else {
		cmd = exec.Command("git", "clone", repoURL, targetPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// 如果是HTTPS认证失败，给出友好提示
		if useHTTPS && (strings.Contains(string(output), "Authentication failed") ||
			strings.Contains(string(output), "could not read Username")) {
			return fmt.Errorf("HTTPS认证失败，请检查用户名或配置Git凭证:\n%s\n\n"+
				"建议解决方案:\n"+
				"1. 使用 -u 参数指定正确的GitHub用户名\n"+
				"2. 配置Git凭证缓存: git config --global credential.helper store\n"+
				"3. 或改用SSH协议(不使用--https参数)", string(output))
		}
		return fmt.Errorf("%v\n%s", err, string(output))
	}

	return nil
}

// isGitAvailable 检查Git是否可用
func isGitAvailable() bool {
	cmd := exec.Command("git", "--version")
	return cmd.Run() == nil
}

// renamePackages 递归遍历目录，重命名所有包含"dilu"的包导入
func renamePackages(projectName string) error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏目录和.git目录
		if strings.HasPrefix(info.Name(), ".") && info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// 只处理.go文件
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if err := replaceInFile(path, projectName); err != nil {
				return fmt.Errorf("处理文件 %s 失败: %w", path, err)
			}
		}

		return nil
	})
}

// replaceInFile 替换文件中的包名
func replaceInFile(filePath, projectName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	originalContent := string(content)
	newContent := originalContent

	newContent = strings.ReplaceAll(newContent, "\"dilu/", "\""+strings.ToLower(projectName)+"/")

	// 如果内容有变化才写入文件
	if newContent != originalContent {
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	return nil
}

// updateGoMod 更新go.mod文件中的module名称
func updateGoMod(projectName string) error {
	goModPath := "go.mod"
	content, err := os.ReadFile(goModPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果没有go.mod文件，创建一个新的
			fmt.Printf("  创建新的 go.mod 文件...\n")
			content = []byte(fmt.Sprintf("module %s\n\ngo 1.21\n", projectName))
			return os.WriteFile(goModPath, content, 0644)
		}
		return err
	}

	lines := strings.Split(string(content), "\n")
	modified := false

	for i, line := range lines {
		if strings.HasPrefix(line, "module ") {
			lines[i] = fmt.Sprintf("module %s", projectName)
			modified = true
			fmt.Printf("  更新 module 名称为: %s\n", projectName)
			break
		}
	}

	if modified {
		return os.WriteFile(goModPath, []byte(strings.Join(lines, "\n")), 0644)
	}

	return nil
}

// updateYamlFrontPath 更新yaml配置文件中的front-path
func updateYamlFrontPath(projectName string) error {
	// 查找所有.yaml文件
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理.yaml和.yml文件
		if !info.IsDir() && (strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			if err := replaceFrontPathInYaml(path, projectName); err != nil {
				return fmt.Errorf("处理yaml文件 %s 失败: %w", path, err)
			}
		}

		return nil
	})
}

// replaceFrontPathInYaml 替换yaml文件中的front-path
func replaceFrontPathInYaml(filePath, projectName string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	originalContent := string(content)
	newContent := originalContent

	// 查找并替换 front-path: ../dilu-admin/src
	lines := strings.Split(newContent, "\n")
	modified := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		// 匹配 front-path: ../dilu-admin/src 或类似格式
		if strings.Contains(trimmedLine, "front-path:") && strings.Contains(trimmedLine, "../dilu-admin/src") {
			// 保持原有缩进和格式
			indent := line[:len(line)-len(trimmedLine)]
			newPath := fmt.Sprintf("../%s-admin/src", projectName)
			// 替换路径部分
			newLine := strings.Replace(trimmedLine, "../dilu-admin/src", newPath, 1)
			lines[i] = indent + newLine
			modified = true
			fmt.Printf("  更新 %s 中的 front-path: %s\n", filePath, newPath)
		}
	}

	if modified {
		newContent = strings.Join(lines, "\n")
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	return nil
}

// capitalizeFirst 将字符串首字母大写
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
