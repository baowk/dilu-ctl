# dilu-ctl 🚀

Dilu 项目快速创建和代码生成工具

## 📚 功能特性

### 1. `create` - 创建新项目
从模板生成完整的项目骨架，支持交互式配置。

### 2. `gen` - 生成模块代码
**核心功能**：结合 GORM-Gen + 模板系统，生成完整的 CRUD 代码。

**生成内容：**
1. **Model/Query 层** - 使用 GORM-Gen 自动生成（类型安全）✅
2. **Service/DTO 层** - 使用模板生成（业务逻辑）✅
3. **API 层** - 使用模板生成（HTTP 接口）✅
4. **Router 层** - 使用模板生成（路由配置）✅

**技术栈：**
- **GORM-Gen**：强大的 ORM 代码生成器，自动生成 Model 和 Query 对象
- **Go Template**：灵活的模板引擎，生成 Service、API、Router 层代码

**支持的数据库：**
- ✅ MySQL
- ✅ PostgreSQL  
- ✅ SQLite

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
# MySQL 示例（driver 可省略，自动推断）
dilu-ctl gen \
  --dns='root:123456@tcp(localhost:3306)/sys' \
  -t sys_user \
  -p sys

# PostgreSQL 示例
dilu-ctl gen \
  --dns='postgres://user:pass@localhost:5432/app' \
  -t users \
  -p app

# SQLite 示例
dilu-ctl gen \
  --dns='sqlite:./data/app.db' \
  -t configs \
  -p data

# 使用短标志（推荐）
dilu-ctl gen \
  -d 'root:123456@tcp(localhost:3306)/sys' \
  -t sys_user \
  -p sys \
  -f
```

**参数：**
- `-t, --table` - 表名（必填）
- `-d, --dns` - 数据库连接字符串（必填）
- `-p, --package` - 包名（可选，默认从表名推断）
- `--driver` - 数据库类型：mysql/postgres/sqlite（可选，自动推断）
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
  -t message \
  -p notice \
  -d 'root:123456@tcp(localhost:3306)/notice'
```

生成的文件：
- `internal/notice/repository/model/message.gen.go` - Model 层
- `internal/notice/repository/query/message.gen.go` - Query 层

## 📋 注意事项

1. 项目目录必须包含 `go.mod`（用于获取 module 名称）
2. 建议为表和字段添加注释
3. 使用 `-f` 参数覆盖已生成的文件
4. 支持 MySQL、PostgreSQL、SQLite 三种数据库
5. Service/DTO/API 字段与类型以 GORM-Gen 的 Model 为准

## 🛠️ 技术栈

- Go 1.21+
- Cobra CLI Framework
- GORM v1.25+
- GORM-Gen v0.3.20
