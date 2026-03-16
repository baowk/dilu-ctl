package cmd

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const gitCommandTimeout = 2 * time.Minute

var projectNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*(/[a-z0-9][a-z0-9_-]*)*$`)

var (
	createProjectName string
	createUseAll      bool
	createOutputPath  string
	createUseHTTPS    bool
	createGitUsername string

	createProjectCmd = &cobra.Command{
		Use:   "create",
		Short: "创建新的 dilu 项目",
		Long:  `从 GitHub 克隆 dilu 或 dilu-all仓库，并创建新的项目`,
		RunE:  runCreateProject,
	}
)

func init() {
	createProjectCmd.Flags().StringVarP(&createProjectName, "name", "n", "", "项目名称（必填）")
	createProjectCmd.Flags().BoolVarP(&createUseAll, "all", "a", false, "是否使用dilu-all仓库（包含完整功能）")
	createProjectCmd.Flags().StringVarP(&createOutputPath, "output", "o", ".", "项目输出路径")
	createProjectCmd.Flags().BoolVar(&createUseHTTPS, "https", false, "使用 HTTPS 协议而非 SSH")
	createProjectCmd.Flags().StringVarP(&createGitUsername, "username", "u", "", "Git 用户名（HTTPS 模式下可选）")

	createProjectCmd.MarkFlagRequired("name")
}

func runCreateProject(cmd *cobra.Command, args []string) error {
	if err := validateProjectName(createProjectName); err != nil {
		return fmt.Errorf("无效的项目名称 '%s': %w", createProjectName, err)
	}

	// 解析输出路径
	absOutputPath, err := filepath.Abs(createOutputPath)
	if err != nil {
		return fmt.Errorf("无效的输出路径 '%s': %w", createOutputPath, err)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(absOutputPath, 0755); err != nil {
		return fmt.Errorf("无法创建输出目录 '%s': %w", absOutputPath, err)
	}

	// 检查项目目录是否已存在
	projectPath := filepath.Join(absOutputPath, createProjectName)
	if _, err := os.Stat(projectPath); err == nil {
		return fmt.Errorf("项目目录 '%s' 已存在", projectPath)
	}

	// 确定要克隆的仓库 URL
	mainRepoURL, adminRepoURL := getRepositoryURLs(createUseAll, createUseHTTPS, createGitUsername)

	fmt.Printf("开始创建项目：%s\n", createProjectName)
	fmt.Printf("项目路径：%s\n", projectPath)
	fmt.Printf("使用协议：%s\n", getProtocolName(createUseHTTPS))
	fmt.Printf("主仓库：%s\n", mainRepoURL)
	if createUseAll {
		fmt.Printf("Admin 仓库：%s\n", adminRepoURL)
	}

	// 创建项目目录
	if err := os.Mkdir(projectPath, 0755); err != nil {
		return fmt.Errorf("创建目录失败：%w", err)
	}

	// 切换到项目目录
	oldDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败：%w", err)
	}

	if err := os.Chdir(projectPath); err != nil {
		return fmt.Errorf("切换目录失败：%w", err)
	}

	defer func() {
		// 恢复原目录
		os.Chdir(oldDir)
	}()

	// 检查 Git 是否可用
	if !isGitAvailable() {
		return fmt.Errorf("未找到 Git 命令，请先安装 Git")
	}

	// 克隆主仓库
	fmt.Println("正在克隆主仓库...")
	if err := cloneRepository(mainRepoURL, ".", createUseHTTPS); err != nil {
		return fmt.Errorf("克隆主仓库失败：%w", err)
	}

	// 如果使用 -a 参数，同时克隆 admin 仓库
	adminProjectName := createProjectName + "-admin"
	adminProjectPath := filepath.Join(absOutputPath, adminProjectName)

	if createUseAll {
		fmt.Printf("正在克隆 admin 仓库到：%s\n", adminProjectName)

		// 检查 admin 目录是否已存在
		if _, err := os.Stat(adminProjectPath); err == nil {
			return fmt.Errorf("admin 项目目录 '%s' 已存在", adminProjectPath)
		}

		// 创建 admin 项目目录
		if err := os.Mkdir(adminProjectPath, 0755); err != nil {
			return fmt.Errorf("创建 admin 目录失败：%w", err)
		}

		// 克隆 admin 仓库
		fmt.Printf("使用 admin 仓库：%s\n", adminRepoURL)

		if err := cloneRepository(adminRepoURL, adminProjectPath, createUseHTTPS); err != nil {
			fmt.Printf("警告：admin 仓库克隆失败，但主项目创建成功\n")
		} else {
			// 在 admin 目录中重命名包名
			fmt.Println("正在重命名 admin 项目包名...")
			if err := os.Chdir(adminProjectPath); err != nil {
				fmt.Fprintf(os.Stderr, "切换到 admin 目录失败：%v\n", err)
			} else {
				if err := renamePackages(adminProjectName); err != nil {
					fmt.Printf("重命名 admin 包名失败：%v\n", err)
				}
				if err := updateGoMod(adminProjectName); err != nil {
					fmt.Printf("更新 admin go.mod 失败：%v\n", err)
				}
				// 清理 admin 的.git 目录
				if err := os.RemoveAll(".git"); err != nil {
					fmt.Printf("清理 admin .git 目录失败：%v\n", err)
				}
				// 恢复到原目录
				os.Chdir(projectPath)
			}
		}
	}

	// 重命名主项目包名
	fmt.Println("正在重命名主项目包名...")
	if err := renamePackages(createProjectName); err != nil {
		return fmt.Errorf("重命名包名失败：%w", err)
	}

	// 更新主项目 go.mod
	fmt.Println("正在更新主项目 go.mod...")
	if err := updateGoMod(createProjectName); err != nil {
		return fmt.Errorf("更新 go.mod 失败：%w", err)
	}

	// 更新 yaml 配置文件中的 front-path
	if createUseAll {
		fmt.Println("正在更新 yaml 配置文件...")
		if err := updateYamlFrontPath(createProjectName); err != nil {
			fmt.Printf("警告：更新 yaml 配置失败：%v\n", err)
		}
	}

	// 移除主项目.git 目录
	fmt.Println("正在清理主项目.git 目录...")
	if err := os.RemoveAll(".git"); err != nil {
		fmt.Printf("警告：清理主项目.git 目录失败：%v\n", err)
	}

	fmt.Printf("\n✅ 项目创建成功！\n")
	fmt.Printf("📁 主项目路径：%s\n", projectPath)
	if createUseAll {
		fmt.Printf("📁 Admin 项目路径：%s\n", adminProjectPath)
	}
	fmt.Printf("🚀 请进入相应目录开始开发\n")
	if createUseHTTPS {
		fmt.Println("💡 提示：如需避免重复输入密码，可配置 Git 凭证缓存:")
		fmt.Println("   git config --global credential.helper store")
	}

	return nil
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
	args := []string{"clone", repoURL}
	if targetPath == "." {
		args = append(args, ".")
	} else {
		args = append(args, targetPath)
	}

	output, err := runGitCommand(args...)
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
	_, err := runGitCommand("--version")
	return err == nil
}

// renamePackages 递归遍历目录，重命名所有包含"dilu"的包导入
func renamePackages(projectName string) error {
	return filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏目录和.git目录
		if info.IsDir() && path != "." && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
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

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return err
	}

	updated := false
	moduleName := strings.ToLower(projectName)
	for _, imp := range file.Imports {
		importPath, unquoteErr := strconv.Unquote(imp.Path.Value)
		if unquoteErr != nil {
			continue
		}
		if strings.HasPrefix(importPath, "dilu/") {
			imp.Path.Value = strconv.Quote(moduleName + "/" + strings.TrimPrefix(importPath, "dilu/"))
			updated = true
		}
	}

	if !updated {
		return nil
	}

	raw, err := formatAST(fset, file)
	if err != nil {
		return err
	}
	formatted, err := format.Source(raw)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, formatted, 0644)
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

func runGitCommand(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	return cmd.CombinedOutput()
}

func validateProjectName(name string) error {
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("不能包含首尾空格")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("不能包含 '..'")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("不能以 '/' 开头或结尾")
	}
	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("仅支持小写字母、数字、下划线、连字符，支持 '/' 作为层级")
	}
	return nil
}

func formatAST(fset *token.FileSet, file any) ([]byte, error) {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
