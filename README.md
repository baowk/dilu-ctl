# Dilu 脚手架工具

[![GitHub](https://img.shields.io/github/license/baowk/dilu-ctl)](https://github.com/baowk/dilu-ctl)
[![Go Report Card](https://goreportcard.com/badge/github.com/baowk/dilu-ctl)](https://goreportcard.com/report/github.com/baowk/dilu-ctl)

这是一个用于快速创建 Dilu 项目的脚手架工具。

GitHub 仓库地址: https://github.com/baowk/dilu-ctl

## 功能特性

- ✅ 通过命令行快速创建 Dilu 项目
- ✅ 支持选择不同的模板仓库
- ✅ 智能包名替换（仅替换本地包，保留外部依赖）
- ✅ 自动生成正确的 go.mod 文件
- ✅ 自动清理 Git 历史记录
- ✅ 支持自定义项目输出路径
- ✅ **支持SSH和HTTPS两种Git协议**
- ✅ **使用 `-a` 参数时自动创建配套的 admin 项目**
- ✅ **自动更新 yaml 配置文件中的 front-path 路径**
- ✅ 智能的错误处理和提示信息

## 安装方式

### 方式一：使用 Go Install（推荐）
```bash
go install github.com/baowk/dilu-ctl@latest
```

安装完成后，可直接使用：
```bash
dilu-ctl -h
```

### 方式二：从源码编译
```bash
# 克隆仓库
git clone https://github.com/baowk/dilu-ctl.git
cd dilu-ctl

# 编译
go build -o dilu-ctl

# 或者安装到 GOPATH
go install
```

### 方式三：直接下载二进制文件
从 [Releases](https://github.com/baowk/dilu-ctl/releases) 页面下载对应平台的二进制文件。

## 使用方法

### 查看帮助
```bash
dilu-ctl -h
```

### 基本用法

```bash
# 使用SSH协议（默认）
dilu-ctl -n 项目名称

# 使用HTTPS协议
dilu-ctl -n 项目名称 --https -u username
```

### 使用示例

1. **创建基础项目（仅克隆dilu）**：
```bash
dilu-ctl -n myproject
```

2. **创建完整项目（克隆dilu-all + admin）**：
```bash
dilu-ctl -n myproject -a
```

3. **创建基础项目（HTTPS协议）**：
```bash
dilu-ctl -n myproject --https -u your-github-username
```

4. **创建完整项目（HTTPS协议）**：
```bash
dilu-ctl -n myproject -a --https -u your-github-username
```

### 参数说明

| 参数 | 说明 | 必填 | 默认值 |
|------|------|------|--------|
| `-n` | 指定项目名称 | 是 | 无 |
| `-a` | 克隆dilu-all和dilu-admin项目 | 否 | false |
| `-o` | 指定项目输出路径 | 否 | 当前目录(.) |
| `--https` | 使用HTTPS协议而非SSH | 否 | false |
| `-u` | Git用户名（HTTPS模式下可选） | 否 | 无 |
| `-h/-help` | 显示帮助信息 | 否 | false |

### 项目克隆行为

**不使用 `-a` 参数（默认）**：
- 只克隆主项目：`dilu.git`
- 适用于简单的后端项目

**使用 `-a` 参数**：
- 克隆主项目：`dilu-all.git` 
- 克隆Admin项目：`dilu-admin.git`（命名为 `项目名-admin`）
- 自动更新配置文件中的路径引用
- 适用于完整的前后端项目

```
/path/to/output/
├── myproject/          # 主项目 (dilu-all)
│   ├── go.mod
│   ├── main.go
│   ├── config.yaml     # front-path 已自动更新
│   └── ...其他文件
└── myproject-admin/    # Admin项目 (dilu-admin)
    ├── go.mod
    ├── main.go
    └── ...其他文件
```

## 工作流程

1. 📂 根据项目名称和输出路径创建新目录
2. 🔧 根据 `-a` 参数确定要克隆的 Git 仓库
3. 📥 克隆主项目代码到项目目录
4. 🔄 如果使用 `-a` 参数，同时克隆 admin 项目
5. 🔍 递归遍历所有 `.go` 文件进行包名替换
6. 🔄 替换代码中的类型引用 `Dilu` → `ProjectName`
7. 📝 更新 `go.mod` 文件中的 module 名称
8. ⚙️ **扫描并更新所有 yaml 配置文件中的 front-path**
9. 🗑️ 清理 `.git` 目录（移除原始仓库历史）
10. ✅ 完成项目初始化

## Git协议选择指南

### SSH协议（推荐）
**优点：**
- 安全性高
- 无需每次输入密码
- 适合频繁操作

**前提条件：**
- 需要配置SSH密钥
- 需要将公钥添加到GitHub账户

**使用方式：**
```bash
dilu-ctl -n myproject  # 默认使用SSH
```

### HTTPS协议
**优点：**
- 配置简单
- 无需SSH密钥
- 适合临时使用

**缺点：**
- 需要每次输入用户名密码（除非配置凭证缓存）
- 安全性相对较低

**使用方式：**
```bash
# 基本使用
dilu-ctl -n myproject --https

# 指定用户名
dilu-ctl -n myproject --https -u your-github-username

# 配置凭证缓存避免重复输入密码
git config --global credential.helper store
```

## 注意事项

⚠️ **重要提醒**：
- 需要确保系统已安装 Git
- 项目目录不能已存在
- 建议项目名称使用小写字母和数字
- 输出路径会自动创建（如果不存在）
- 使用 `-a` 参数时会同时创建两个项目目录
- YAML 配置更新功能仅在使用 `-a` 参数时生效

⚠️ **SSH协议注意事项**：
- 需要预先配置好SSH密钥
- 确保公钥已添加到GitHub账户

⚠️ **HTTPS协议注意事项**：
- 可能需要多次输入用户名密码
- 建议配置凭证缓存：`git config --global credential.helper store`
- 使用 `-u` 参数可以指定GitHub用户名

## 仓库地址

根据协议不同，使用的仓库地址也会相应变化：

### SSH协议（默认）
- **基础模板**：`git@github.com:baowk/dilu.git`
- **完整模板**：`git@github.com:baowk/dilu-all.git`
- **Admin模板**：`git@github.com:baowk/dilu-admin.git`

### HTTPS协议
- **基础模板**：`https://github.com/baowk/dilu.git`
- **完整模板**：`https://github.com/baowk/dilu-all.git`
- **Admin模板**：`https://github.com/baowk/dilu-admin.git`

## 故障排除

### Git 相关问题
```bash
# 检查Git是否安装
git --version

# 测试SSH连接（SSH协议）
ssh -T git@github.com

# 配置HTTPS凭证缓存（HTTPS协议）
git config --global credential.helper store
```

### 权限问题
```bash
# 确保对目标目录有写权限
ls -la /path/to/target/directory
```

### 网络问题
如果克隆失败，可以尝试：
1. 检查网络连接
2. 验证GitHub SSH密钥配置
3. 使用HTTPS方式克隆（需要修改源码）

## 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

## License

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 开发者信息

如有问题或建议，请提交 Issue 或 Pull Request。