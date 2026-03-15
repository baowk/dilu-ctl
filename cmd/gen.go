package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gen"
	"gorm.io/gorm"
)

var (
	genDbName      string
	genTableName   string
	genPackageName string
	genDriver      string
	genDsn         string
	genForce       bool
	genProjectPath string
	genApiPrefix   string
	
	genCmd = &cobra.Command{
		Use:   "gen",
		Short: "根据数据库表生成模块代码（使用 GORM-Gen）",
		Long: `从数据库读取表结构，使用 GORM-Gen 生成 Model 和 Query 层代码。

支持的数据库类型:
  - mysql
  - postgres (PostgreSQL)  
  - sqlite

生成的代码包括:
  - Model 层 (repository/model/) - 使用 GORM-Gen 自动生成
  - Query 层 (repository/query/) - 使用 GORM-Gen 自动生成`,
		RunE: runGenModule,
	}
)

func init() {
	genCmd.Flags().StringVarP(&genDbName, "db", "d", "", "数据库名称（如：sys, notice）")
	genCmd.Flags().StringVarP(&genTableName, "table", "t", "", "表名（必填）")
	genCmd.Flags().StringVarP(&genPackageName, "package", "p", "", "包名（可选，默认与 db 一致）")
	genCmd.Flags().StringVar(&genDriver, "driver", "mysql", "数据库类型：mysql/postgres/sqlite")
	genCmd.Flags().StringVarP(&genDsn, "dns", "", "", "数据库连接字符串（必填）")
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

	// 如果未指定包名，使用数据库名
	if genPackageName == "" {
		genPackageName = genDbName
	}

	// 解析项目路径
	absProjectPath, err := filepath.Abs(genProjectPath)
	if err != nil {
		return fmt.Errorf("无效的项目路径 '%s': %w", genProjectPath, err)
	}

	fmt.Printf("开始使用 GORM-Gen 生成代码...\n")
	fmt.Printf("项目路径：%s\n", absProjectPath)
	fmt.Printf("数据库：%s\n", genDbName)
	fmt.Printf("表名：%s\n", genTableName)
	fmt.Printf("包名：%s\n", genPackageName)
	fmt.Printf("数据库类型：%s\n", genDriver)
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

	// 连接数据库
	db, err := connectDB(genDriver, genDsn)
	if err != nil {
		return fmt.Errorf("连接数据库失败：%w", err)
	}

	// 使用 GORM-Gen 生成 Model 和 Query 层
	if err := generateWithGORMGen(db, genTableName, genPackageName, genForce); err != nil {
		return fmt.Errorf("GORM-Gen 生成失败：%w", err)
	}

	fmt.Printf("\n✅ GORM-Gen 代码生成成功！\n")
	fmt.Printf("📁 生成路径：%s/internal/%s/repository/\n", absProjectPath, genPackageName)
	fmt.Printf("📄 已生成:\n")
	fmt.Printf("   - internal/%s/repository/model/%s.gen.go\n", genPackageName, genTableName)
	fmt.Printf("   - internal/%s/repository/query/%s.gen.go\n", genPackageName, genTableName)

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

	// 配置 GORM-Gen - ModelPkgPath 是相对于当前工作目录的绝对路径或相对路径
	g := gen.NewGenerator(gen.Config{
		OutPath:           queryPath,
		ModelPkgPath:      modelPath,  // 使用完整路径
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
