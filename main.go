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
	var help bool

	flag.StringVar(&projectName, "n", "", "项目名称")
	flag.BoolVar(&useAll, "a", false, "是否使用dilu-all仓库")
	flag.StringVar(&outputPath, "o", ".", "项目输出路径")
	flag.BoolVar(&help, "h", false, "显示帮助信息")
	flag.BoolVar(&help, "help", false, "显示帮助信息")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  创建基础项目到当前目录: ./dilu-ctl -n myproject")
		fmt.Println("  创建项目到指定目录: ./dilu-ctl -n myproject -o /path/to/output")
		fmt.Println("  创建完整项目: ./dilu-ctl -n myproject -a -o /path/to/output")
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

	// 确定要克隆的仓库
	var repoURL string
	if useAll {
		repoURL = "git@github.com:baowk/dilu-all.git"
	} else {
		repoURL = "git@github.com:baowk/dilu.git"
	}

	fmt.Printf("开始创建项目: %s\n", projectName)
	fmt.Printf("项目路径: %s\n", projectPath)
	fmt.Printf("使用仓库: %s\n", repoURL)

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

	// 克隆仓库
	fmt.Println("正在克隆仓库...")
	cmd := exec.Command("git", "clone", repoURL, ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "克隆失败: %v\n%s", err, string(output))
		os.Exit(1)
	}

	// 重命名包名
	fmt.Println("正在重命名包名...")
	if err := renamePackages(projectName); err != nil {
		fmt.Fprintf(os.Stderr, "重命名包名失败: %v\n", err)
		os.Exit(1)
	}

	// 更新go.mod
	fmt.Println("正在更新go.mod...")
	if err := updateGoMod(projectName); err != nil {
		fmt.Fprintf(os.Stderr, "更新go.mod失败: %v\n", err)
		os.Exit(1)
	}

	// 移除.git目录
	fmt.Println("正在清理.git目录...")
	if err := os.RemoveAll(".git"); err != nil {
		fmt.Printf("警告: 清理.git目录失败: %v\n", err)
		// 不退出，这只是清理工作
	}

	fmt.Printf("\n✅ 项目 %s 创建成功！\n", projectName)
	fmt.Printf("📁 项目路径: %s\n", projectPath)
	fmt.Printf("🚀 请进入目录 cd %s 并开始开发\n", projectPath)
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

		// 处理.go文件
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if err := replaceInFile(path, projectName); err != nil {
				return fmt.Errorf("处理文件 %s 失败: %w", path, err)
			}
		} else if !info.IsDir() && strings.HasSuffix(info.Name(), ".template") { // 处理.template文件
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

	// 只替换本地包导入，不替换外部依赖
	// 匹配 "dilu/xxx" 格式的导入，但不匹配 "github.com/xxx/dilu-xxx" 格式
	// lines := strings.Split(newContent, "\n")
	// modified := false

	// lowerProjectName := strings.ToLower(projectName)

	// for _, line := range lines {
	// 	// 检查是否为导入语句
	// 	//trimmedLine := strings.TrimSpace(line)

	// 	strings.Replace(line, "\"dilu/", "\""+lowerProjectName+"/", -1)
	// 	modified = true

	// 	// // 处理单行导入: import "dilu/xxx"
	// 	// if strings.HasPrefix(trimmedLine, `import "`) && strings.Contains(trimmedLine, `"dilu/`) && !strings.Contains(trimmedLine, "github.com/") {
	// 	// 	// 保持原有缩进
	// 	// 	indent := line[:len(line)-len(trimmedLine)]
	// 	// 	oldImport := strings.TrimPrefix(trimmedLine, `import "`)
	// 	// 	oldImport = strings.TrimSuffix(oldImport, `"`)
	// 	// 	newImport := strings.ReplaceAll(oldImport, "dilu/", lowerProjectName+"/")
	// 	// 	lines[i] = indent + fmt.Sprintf(`import "%s"`, newImport)
	// 	// 	modified = true
	// 	// 	continue
	// 	// }

	// 	// // 处理多行导入块中的单行: "dilu/xxx"
	// 	// if strings.HasPrefix(trimmedLine, `"`) && strings.Contains(trimmedLine, `"dilu/`) && !strings.Contains(trimmedLine, "github.com/") {
	// 	// 	// 保持原有缩进
	// 	// 	indent := line[:len(line)-len(trimmedLine)]
	// 	// 	oldImport := strings.Trim(trimmedLine, `"`)
	// 	// 	newImport := strings.ReplaceAll(oldImport, "dilu/", lowerProjectName+"/")
	// 	// 	lines[i] = indent + fmt.Sprintf(`"%s"`, newImport)
	// 	// 	modified = true
	// 	// 	continue
	// 	// }

	// 	// // 处理带别名的导入: alias "dilu/xxx"
	// 	// if strings.Contains(trimmedLine, `"dilu/`) && !strings.Contains(trimmedLine, "github.com/") {
	// 	// 	// 保持原有缩进
	// 	// 	indent := line[:len(line)-len(trimmedLine)]
	// 	// 	parts := strings.SplitN(trimmedLine, `"`, 3)
	// 	// 	if len(parts) >= 3 {
	// 	// 		aliasPart := parts[0]
	// 	// 		importPath := parts[1]
	// 	// 		if strings.Contains(importPath, "dilu/") {
	// 	// 			newImportPath := strings.ReplaceAll(importPath, "dilu/", lowerProjectName+"/")
	// 	// 			lines[i] = indent + fmt.Sprintf(`%s"%s"`, aliasPart, newImportPath)
	// 	// 			modified = true
	// 	// 		}
	// 	// 	}
	// 	// }
	// }

	// if modified {
	// 	newContent = strings.Join(lines, "\n")
	// }

	// 替换代码中的类型引用 Dilu -> ProjectName
	newContent = strings.ReplaceAll(newContent, "Dilu", capitalizeFirst(projectName))
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

// capitalizeFirst 将字符串首字母大写
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
