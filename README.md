# dilu-ctl 🚀

Dilu 项目快速创建和代码生成工具

## ✨ 功能特性

1. **快速创建项目** - 从 GitHub 克隆 dilu 或 dilu-all 仓库
2. **代码生成** - 根据数据库表结构生成模块代码（使用 GORM-Gen）

## 📦 安装

### 方式一：go install（推荐）

```bash
go install github.com/baowk/dilu-ctl@latest
```

安装后可执行文件位于 `$GOPATH/bin/dilu-ctl`

### 方式二：源码编译

```bash
cd dilu-ctl
go build -o dilu-ctl
```

## 🎯 命令说明

### 1. 创建项目 (create)

```bash
# 创建基础项目
dilu-ctl create -n myproject

# 创建完整项目（包含 admin 前端）
dilu-ctl create -n myproject -a

# 使用 HTTPS 协议
dilu-ctl create -n myproject --https -u username
```

**参数：**
- `-n, --name` - 项目名称（必填）
- `-a, --all` - 使用dili-all 仓库
- `-o, --output` - 输出路径
- `--https` - 使用 HTTPS 协议
- `-u, --username` - Git 用户名

### 2. 生成代码 (gen)

```bash
# MySQL
dilu-ctl gen \
  -db sys \
  -table sys_user \
  -dns 'root:123456@tcp(localhost:3306)/sys'

# PostgreSQL
dilu-ctl gen \
  -db app \
  -table users \
  --driver=postgres \
  --dns='postgres://user:pass@localhost:5432/app'

# SQLite
dilu-ctl gen \
  -db data \
  -table configs \
  --driver=sqlite \
  --dns='./data/app.db'
```

**参数：**
- `-t, --table` - 表名（必填）
- `--dns` - 数据库连接字符串（必填）
- `-d, --db` - 数据库名称
- `-p, --package` - 包名（默认与 db 一致）
- `--driver` - 数据库类型：mysql/postgres/sqlite（默认 mysql）
- `-f, --force` - 覆盖已存在的文件
- `-P, --project` - 项目根目录路径（默认 .）
- `--prefix` - API 路径前缀（默认 /v1）

## 🔧 使用流程

### 步骤 1: 创建项目
```bash
dilu-ctl create -n myproject -a
cd myproject
```

### 步骤 2: 准备数据库
```sql
CREATE DATABASE notice;
USE notice;

CREATE TABLE message (
  id INT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(100),
  content TEXT,
  created_at DATETIME
);
```

### 步骤 3: 生成代码
```bash
dilu-ctl gen \
  -db notice \
  -table message \
  -dns 'root:123456@tcp(localhost:3306)/notice'
```

生成的文件：
- `internal/notice/repository/model/message.gen.go` - Model 层
- `internal/notice/repository/query/message.gen.go` - Query 层

## 📋 注意事项

1. gen 命令必须在 dilu 项目根目录下执行
2. 建议为表和字段添加注释
3. 使用 `-f` 参数覆盖已生成的文件
4. 支持 MySQL、PostgreSQL、SQLite 三种数据库

## 🛠️ 技术栈

- Go 1.21+
- Cobra CLI Framework
- GORM v1.25+
- GORM-Gen v0.3.20