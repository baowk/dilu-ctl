package cmd

import (
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gen"
	"gorm.io/gorm"
)

//go:embed templates/*
var templateFS embed.FS

var (
	genTableName    string
	genPackageName  string
	genDriver       string
	genDsn          string
	genForce        bool
	genProjectPath  string
	genApiPrefix    string
	genProjectName  string // 动态项目名称

	genCmd = &cobra.Command{
		Use:   "gen",
		Short: "根据数据库表生成模块代码（GORM-Gen + 模板）",
		Long: `从数据库读取表结构，生成完整的模块代码。

生成内容:
  1. Model/Query 层 - 使用 GORM-Gen 自动生成
  2. Service/DTO 层 - 使用模板生成
  3. API 层 - 使用模板生成
  4. Router 层 - 使用模板生成

支持的数据库类型:
  - mysql
  - postgres (PostgreSQL)  
  - sqlite`,
		RunE: runGenModule,
	}
)

func init() {
	genCmd.Flags().StringVarP(&genTableName, "table", "t", "", "表名（必填）")
	genCmd.Flags().StringVarP(&genPackageName, "package", "p", "", "包名（可选，默认从 DSN 推断）")
	genCmd.Flags().StringVar(&genDriver, "driver", "mysql", "数据库类型：mysql/postgres/sqlite")
	genCmd.Flags().StringVarP(&genDsn, "dns", "d", "", "数据库连接字符串（必填）")
	genCmd.Flags().BoolVarP(&genForce, "force", "f", false, "覆盖已存在的文件")
	genCmd.Flags().StringVarP(&genProjectPath, "project", "P", ".", "项目根目录路径")
	genCmd.Flags().StringVar(&genApiPrefix, "prefix", "/v1", "API 路径前缀")

	genCmd.MarkFlagRequired("table")
	genCmd.MarkFlagRequired("dns")
}

func runGenModule(cmd *cobra.Command, args []string) error {
	if genTableName == "" {
		return fmt.Errorf("必须指定表名 (-table)")
	}

	if genDsn == "" {
		return fmt.Errorf("必须指定数据库连接字符串 (-dns)")
	}

	// 如果未指定包名，使用表名作为默认值（简单处理）
	if genPackageName == "" {
		genPackageName = strings.Split(genTableName, "_")[0]
	}

	// 解析项目路径
	absProjectPath, err := filepath.Abs(genProjectPath)
	if err != nil {
		return fmt.Errorf("无效的项目路径 '%s': %w", genProjectPath, err)
	}

	// 获取项目名称（从 go.mod 中读取）
	genProjectName, err = getProjectName(absProjectPath)
	if err != nil {
		return fmt.Errorf("获取项目名称失败：%w", err)
	}

	fmt.Printf("开始生成代码...\n")
	fmt.Printf("DSN: %s\n", genDsn)
	fmt.Printf("表名：%s\n", genTableName)
	fmt.Printf("包名：%s\n", genPackageName)
	fmt.Printf("数据库类型：%s\n", genDriver)
	fmt.Printf("项目路径：%s\n", absProjectPath)
	fmt.Printf("项目名称：%s\n", genProjectName)
	fmt.Printf("覆盖模式：%v\n", genForce)

	// 切换到项目目录
	oldDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败：%w", err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(absProjectPath); err != nil {
		return fmt.Errorf("切换到项目目录失败：%w", err)
	}

	// 验证是否为 dilu 项目
	if !isValidDiluProject(absProjectPath) {
		return fmt.Errorf("不是有效的 dilu 项目目录")
	}

	// 连接数据库
	db, err := connectDB(genDriver, genDsn)
	if err != nil {
		return fmt.Errorf("连接数据库失败：%w", err)
	}

	// 步骤 1: 使用 GORM-Gen 生成 Model 和 Query 层
	if err := generateWithGORMGen(db, genTableName, genPackageName, genForce); err != nil {
		return fmt.Errorf("GORM-Gen 生成失败：%w", err)
	}

	// 步骤 2: 使用模板生成Service、API、Router 层
	if err := generateWithTemplates(db, genTableName, genPackageName, genApiPrefix, genForce); err != nil {
		return fmt.Errorf("模板生成失败：%w", err)
	}

	fmt.Printf("\n✅ 代码生成成功！\n")
	fmt.Printf("📁 生成路径：%s/internal/%s/\n", absProjectPath, genPackageName)
	fmt.Printf("📄 已生成:\n")
	fmt.Printf("   - repository/model/%s.gen.go (GORM-Gen)\n", genTableName)
	fmt.Printf("   - repository/query/%s.gen.go (GORM-Gen)\n", genTableName)
	fmt.Printf("   - service/dto/%s.go (模板)\n", genTableName)
	fmt.Printf("   - service/%s_service.go (模板)\n", genTableName)
	fmt.Printf("   - apis/%s.go (模板)\n", genTableName)
	fmt.Printf("   - router/%s.go (模板)\n", genTableName)
	fmt.Printf("   - router/router.go (路由配置模板)\n", genTableName)

	return nil
}

func generateWithGORMGen(db *gorm.DB, tableName, packageName string, force bool) error {
	// 创建输出目录
	modelPath := filepath.Join("internal", packageName, "repository", "model")
	queryPath := filepath.Join("internal", packageName, "repository", "query")

	if err := os.MkdirAll(modelPath, 0755); err != nil {
		return fmt.Errorf("创建 model 目录失败：%w", err)
	}
	if err := os.MkdirAll(queryPath, 0755); err != nil {
		return fmt.Errorf("创建 query 目录失败：%w", err)
	}

	// 配置 GORM-Gen
	g := gen.NewGenerator(gen.Config{
		OutPath:           queryPath,
		ModelPkgPath:      modelPath,
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable:     false,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})

	// 设置数据库
	g.UseDB(db)

	// 生成 Model 和 Query
	className := toClassName(tableName)
	g.ApplyBasic(g.GenerateModelAs(tableName, className))

	// 执行生成
	g.Execute()

	return nil
}

func generateWithTemplates(db *gorm.DB, tableName, packageName, apiRoot string, force bool) error {
	// 读取表结构信息
	tableInfo, err := readTableInfo(db, tableName, packageName, apiRoot)
	if err != nil {
		return fmt.Errorf("读取表结构失败：%w", err)
	}

	// 创建目录
	dirs := []string{
		filepath.Join("internal", packageName, "service", "dto"),
		filepath.Join("internal", packageName, "service"),
		filepath.Join("internal", packageName, "apis"),
		filepath.Join("internal", packageName, "router"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败：%w", err)
		}
	}

	// 生成各个层的代码
	generators := []struct {
		name     string
		template string
		output   string
		subDir   string // Template subdirectory (default: "go/service")
	}{
		{"DTO", "dto.go.template", filepath.Join("internal", packageName, "service", "dto", tableName+".go"), "go/service"},
		{"Service", "service.go.template", filepath.Join("internal", packageName, "service", tableName+"_service.go"), "go/service"},
		{"API", "apis.go.template", filepath.Join("internal", packageName, "apis", tableName+".go"), "go/service"},
		{"Router", "router_no_check_role.go.template", filepath.Join("internal", packageName, "router", tableName+".go"), "go/service"},
		{"RouterConfig", "router.template", filepath.Join("internal", packageName, "router", "router.go"), "go/router"},
	}

	for _, gen := range generators {
		if err := generateFile(gen.template, tableInfo, gen.output, force, gen.subDir); err != nil {
			return fmt.Errorf("生成%s失败：%w", gen.name, err)
		}
	}

	return nil
}

type TableInfo struct {
	ProjectName  string // Dynamic project name from go.mod
	PackageName  string
	ModuleName   string // Module name same as package name
	ClassName    string
	TableName    string
	ConfDbName   string
	TBName       string
	TableComment string
	PkGoField    string // Primary key Go field name
	ApiRoot      string // API root path prefix (e.g., /v1)
	Columns      []ColumnInfo
}

type ColumnInfo struct {
	Name          string
	GoField       string
	GoType        string
	Type          string
	Comment       string
	Pk            bool
	NotNull       bool
	IsQuery       bool
	IsEdit        bool // Is editable field
	IsNil         bool // Allow null values
	IsValid       bool // Need validation
	IsZero        bool // Can be zero value
	QueryType     string
	JsonField     string
	ColumnDefault sql.NullString

	// Template aliases for compatibility
	ColumnName    string // Alias for Name
	ColumnComment string // Alias for Comment
}

func readTableInfo(db *gorm.DB, tableName, packageName, apiRoot string) (*TableInfo, error) {
	columns, err := getColumns(db, tableName)
	if err != nil {
		return nil, err
	}

	className := toClassName(tableName)

	// Find primary key Go field name
	pkGoField := "ID" // default to ID
	for _, col := range columns {
		if col.Pk {
			pkGoField = col.GoField
			break
		}
	}

	return &TableInfo{
		ProjectName:  genProjectName, // Use global project name from go.mod
		PackageName:  packageName,
		ModuleName:   packageName, // Module name same as package name
		ClassName:    className,
		TableName:    tableName,
		ConfDbName:   packageName,
		TBName:       tableName,
		TableComment: "", // TODO: 读取表注释
		PkGoField:    pkGoField,
		ApiRoot:      apiRoot,
		Columns:      columns,
	}, nil
}

func getColumns(db *gorm.DB, tableName string) ([]ColumnInfo, error) {
	var columns []ColumnInfo

	// 使用 GORM 的 Migrate 来获取表结构
	migrator := db.Migrator()
	if !migrator.HasTable(tableName) {
		return nil, fmt.Errorf("表 %s 不存在", tableName)
	}

	colTypes, err := migrator.ColumnTypes(tableName)
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败：%w", err)
	}

	for _, colType := range colTypes {
		name := colType.Name()
		dbType := colType.DatabaseTypeName()

		// 获取默认值（可能为 NULL）
		var defaultVal sql.NullString
		if dv, ok := colType.DefaultValue(); ok {
			defaultVal = sql.NullString{String: dv, Valid: true}
		}

		// Go 字段名
		goField := toGoFieldName(name)

		// JSON 字段名（驼峰命名，特殊缩写转小写）
		jsonField := toCamelCase(name)

		// Go 类型映射
		goType := mapDBTypeToGoType(dbType)

		// 判断是否为主键（简化处理，假设 id 列是主键）
		pk := name == "id"

		// 判断是否为 NOT NULL
		notNull := false
		if nullable, ok := colType.Nullable(); ok {
			notNull = !nullable
		}

		columns = append(columns, ColumnInfo{
			Name:          name,
			GoField:       goField,
			GoType:        goType,
			Type:          dbType,
			Comment:       "", // TODO: 读取注释
			Pk:            pk,
			NotNull:       notNull,
			IsQuery:       true,     // 默认都是查询字段
			IsEdit:        true,     // 默认都可编辑
			IsNil:         !notNull, // 可为空
			IsValid:       false,    // 暂时不需要验证
			IsZero:        true,     // 可以为零值
			QueryType:     "=",      // 默认精确匹配
			JsonField:     jsonField,
			ColumnDefault: defaultVal,
			// Template aliases
			ColumnName:    name,
			ColumnComment: "",
		})
	}

	return columns, nil
}

func mapDBTypeToGoType(dbType string) string {
	switch strings.ToUpper(dbType) {
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT":
		return "int"
	case "VARCHAR", "CHAR", "TEXT", "NVARCHAR", "NVARCHAR2":
		return "string"
	case "DATETIME", "TIMESTAMP", "DATE":
		return "time.Time"
	case "DECIMAL", "NUMERIC", "MONEY":
		return "float64"
	case "BOOLEAN", "BIT":
		return "bool"
	default:
		return "string"
	}
}

func toGoFieldName(name string) string {
	// 常见缩写映射，保持全大写
	abbreviations := map[string]bool{
		"id": true, "url": true, "api": true, "sql": true,
		"orm": true, "db": true, "io": true, "http": true,
		"https": true, "ftp": true, "ssh": true, "xml": true,
		"json": true, "yaml": true, "csv": true, "html": true,
		"css": true, "js": true, "ts": true, "go": true,
	}
	
	parts := strings.Split(name, "_")
	fieldName := ""
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		// 如果整个部分都是大写字母（如 URL、API），保持全大写
		if isAllUpper(part) {
			fieldName += part
		} else if abbreviations[strings.ToLower(part)] {
			// 常见缩写转为全大写
			fieldName += strings.ToUpper(part)
		} else if len(part) == 1 {
			// 单个字母转为大写
			fieldName += strings.ToUpper(part)
		} else {
			// 首字母大写，其余小写
			fieldName += strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return fieldName
}

// isAllUpper 检查字符串是否全部由大写字母组成
func isAllUpper(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// toCamelCase 将 snake_case 转为 camelCase，特殊缩写词转小写
func toCamelCase(name string) string {
	// 特殊缩写词映射（全大写 -> 全小写）
	specialAbbreviations := map[string]string{
		"id": "id", "url": "url", "api": "api", "sql": "sql",
		"orm": "orm", "db": "db", "io": "io", "http": "http",
		"https": "https", "ftp": "ftp", "ssh": "ssh", "xml": "xml",
		"json": "json", "yaml": "yaml", "csv": "csv", "html": "html",
		"css": "css", "js": "js", "ts": "ts", "go": "go",
		"dns": "dns", "ip": "ip", "tcp": "tcp", "udp": "udp",
	}
	
	parts := strings.Split(name, "_")
	result := ""
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		
		lowerPart := strings.ToLower(part)
		// 如果是特殊缩写词，使用全小写
		if special, ok := specialAbbreviations[lowerPart]; ok {
			if i == 0 {
				result += special
			} else {
				result += strings.ToUpper(string(special[0])) + special[1:]
			}
		} else if len(part) == 1 {
			// 单个字母
			if i == 0 {
				result += strings.ToLower(part)
			} else {
				result += strings.ToUpper(part)
			}
		} else {
			// 普通单词：首字母小写（第一个单词）或大写（后续单词）
			if i == 0 {
				result += strings.ToLower(part)
			} else {
				result += strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
			}
		}
	}
	return result
}

func generateFile(templateName string, data *TableInfo, outputPath string, force bool, subDir ...string) error {
	// 检查文件是否存在
	if _, err := os.Stat(outputPath); err == nil && !force {
		fmt.Printf("⚠️  跳过 %s (已存在，使用 -f 覆盖)\n", outputPath)
		return nil
	}

	// 确定模板子目录（默认为 go/service）
	templateSubDir := "go/service"
	if len(subDir) > 0 && subDir[0] != "" {
		templateSubDir = subDir[0]
	}

	// 读取模板
	tmplContent, err := templateFS.ReadFile("templates/" + templateSubDir + "/" + templateName)
	if err != nil {
		return fmt.Errorf("读取模板失败：%w", err)
	}

	// 解析模板
	tmpl, err := template.New(templateName).Funcs(template.FuncMap{
		"hasValue": func(v interface{}) bool {
			return v != nil
		},
	}).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("解析模板失败：%w", err)
	}

	// 渲染模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("渲染模板失败：%w", err)
	}

	// 格式化代码
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// 格式化失败，使用原始内容
		formatted = buf.Bytes()
	}

	// 写入文件
	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("写入文件失败：%w", err)
	}

	fmt.Printf("  → 生成 %s\n", outputPath)
	return nil
}

func connectDB(driver, dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	switch driver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	case "postgres", "postgresql", "pg":
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	case "sqlite", "sqlite3":
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	default:
		return nil, fmt.Errorf("不支持的数据库类型：%s，支持 mysql/postgres/sqlite", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("数据库连接失败：%w", err)
	}

	return db, nil
}

func isValidDiluProject(projectPath string) bool {
	requiredDirs := []string{"internal", "cmd", "common"}
	for _, dir := range requiredDirs {
		path := filepath.Join(projectPath, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("⚠️  警告：缺少 %s 目录，可能不是有效的 dilu 项目\n", dir)
			return false
		}
	}
	return true
}

// getProjectName 从 go.mod 文件中提取项目名称
func getProjectName(projectPath string) (string, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("无法读取 go.mod 文件：%w", err)
	}
	
	content := string(data)
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			// 移除可能的版本号后缀
			if idx := strings.Index(moduleName, "/v"); idx > 0 {
				moduleName = moduleName[:idx]
			}
			return moduleName, nil
		}
	}
	
	return "", fmt.Errorf("未在 go.mod 中找到 module 声明")
}

func toClassName(tableName string) string {
	parts := strings.Split(tableName, "_")
	className := ""
	for _, part := range parts {
		if len(part) > 0 {
			className += strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return className
}
