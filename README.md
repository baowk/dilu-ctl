# dilu-ctl 🚀

Dilu 项目快速创建和代码生成工具

---

## 使用场景

### 👤 人类开发者

使用 `create` + `gen` 完整工作流，从数据库表一键生成全部代码：

```
1. dilu-ctl create  →  创建项目骨架
2. 建表
3. dilu-ctl gen     →  生成 Model / Query / Service / API / Router 全套代码
4. 注册路由，启动
```

### 🤖 AI 协作开发（Claude Code）

只用 `create` 创建项目骨架，后续代码由 AI 按 `CLAUDE.md` 规范直接编写：

```
1. dilu-ctl create  →  创建项目骨架
2. 建表 SQL 告知 AI
3. AI 编写 model → service → api → router
4. 注册路由，启动
```

> AI 不需要运行 `gen`，直接写标准 GORM 代码，无工具依赖。

---

## 📦 安装

```bash
go install github.com/baowk/dilu-ctl@latest
```

---

## 🎯 命令说明

### create — 创建项目

```bash
# 基础项目
dilu-ctl create -n myproject

# 完整项目（含 admin 前端）
dilu-ctl create -n myproject -a

# HTTPS 协议
dilu-ctl create -n myproject --https -u username
```

| 参数 | 说明 |
|------|------|
| `-n, --name` | 项目名称（必填） |
| `-a, --all` | 使用 dilu-all 仓库（含前端） |
| `-o, --output` | 输出路径（默认 `.`） |
| `--https` | 使用 HTTPS 协议 |
| `-u, --username` | Git 用户名 |

---

### gen — 生成模块代码（人类使用）

从数据库表结构生成完整模块代码，进入项目根目录后执行。

**生成内容：**
- Model / Query 层（GORM-Gen 类型安全）
- Service / DTO 层（模板）
- API 层（模板）
- Router 层（模板）

**支持数据库：** MySQL · PostgreSQL · SQLite

```bash
# MySQL
dilu-ctl gen -d 'root:123456@tcp(localhost:3306)/sys' -t sys_user -p sys

# PostgreSQL
dilu-ctl gen -d 'postgres://user:pass@localhost:5432/app' -t users -p app

# SQLite
dilu-ctl gen -d 'sqlite:./data/app.db' -t configs -p data

# 覆盖已有文件
dilu-ctl gen -d '...' -t sys_user -p sys -f
```

| 参数 | 说明 |
|------|------|
| `-t, --table` | 表名（必填） |
| `-d, --dns` | 数据库连接字符串（必填） |
| `-p, --package` | 包名（可选，默认从表名推断） |
| `--driver` | 数据库类型（可选，自动推断） |
| `-f, --force` | 覆盖已存在的文件 |
| `-P, --project` | 项目根目录（默认 `.`） |
| `--prefix` | API 路径前缀（默认 `/v1`） |

---

### version — 查看版本

```bash
dilu-ctl version
```

---

## 🛠️ 技术栈

- Go 1.21+
- Cobra CLI Framework
- GORM v1.25+ / GORM-Gen v0.3.20
