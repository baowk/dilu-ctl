# Dilu 脚手架工具

这是一个用于快速创建 Dilu 项目的脚手架工具。

## 功能特性

- ✅ 通过命令行快速创建 Dilu 项目
- ✅ 支持选择不同的模板仓库
- ✅ 智能包名替换（仅替换本地包，保留外部依赖）
- ✅ 自动生成正确的 go.mod 文件
- ✅ 自动清理 Git 历史记录
- ✅ 支持自定义项目输出路径
- ✅ 智能的错误处理和提示信息

## 安装

### 方法一：直接编译
```bash
go build -o dilu-ctl
```

### 方法二：安装到 GOPATH
```bash
go install
```

## 使用方法

### 查看帮助
```bash
./dilu-ctl -h
```

### 基本用法

```bash
./dilu-ctl -n 项目名称
```

### 参数说明

| 参数 | 说明 | 必填 | 默认值 |
|------|------|------|--------|
| `-n` | 指定项目名称 | 是 | 无 |
| `-o` | 指定项目输出路径 | 否 | 当前目录(.) |
| `-a` | 使用 dilu-all 仓库 | 否 | false |
| `-h/-help` | 显示帮助信息 | 否 | false |

### 使用示例

1. **创建基础项目到当前目录**：
```bash
./dilu-ctl -n myproject
```

2. **创建项目到指定目录**：
```bash
./dilu-ctl -n myproject -o /path/to/output
```

3. **创建完整项目到指定目录**：
```bash
./dilu-ctl -n myproject -a -o /path/to/output
```

4. **查看帮助信息**：
```bash
./dilu-ctl -h
```

## 工作流程

1. 📂 根据项目名称和输出路径创建新目录
2. 🔧 根据 `-a` 参数确定要克隆的 Git 仓库
3. 📥 克隆代码到项目目录
4. 🔍 递归遍历所有 `.go` 文件
5. 🔄 智能替换本地包导入（保持外部依赖不变）
6. 🔄 替换代码中的类型引用 `Dilu` → `ProjectName`
7. 📝 更新 `go.mod` 文件中的 module 名称
8. 🗑️ 清理 `.git` 目录（移除原始仓库历史）
9. ✅ 完成项目初始化

## 包替换规则

工具会智能识别并只替换本地包导入，保留外部依赖包：

### ✅ 会被替换的本地包
```go
import "dilu/common/codes"        // → "myproject/common/codes"
import "dilu/core/config"         // → "myproject/core/config"
core "dilu/core"                  // → core "myproject/core"
```

### ❌ 不会被替换的外部依赖
```go
import "github.com/baowk/dilu-core/config"  // 保持不变
import "github.com/gin-gonic/gin"           // 保持不变
```

### 类型引用替换
```go
DiluApp := NewDiluApplication()   // → MyprojectApp := NewMyprojectApplication()
```

## 注意事项

⚠️ **重要提醒**：
- 需要确保系统已安装 Git
- 需要有访问 GitHub 仓库的权限（SSH密钥配置）
- 项目目录不能已存在
- 建议项目名称使用小写字母和数字
- 输出路径会自动创建（如果不存在）

## 仓库地址

- **基础模板**：`git@github.com:baowk/dilu.git`
- **完整模板**：`git@github.com:baowk/dilu-all.git`

## 故障排除

### Git 相关问题
```bash
# 检查Git是否安装
git --version

# 测试SSH连接
ssh -T git@github.com
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

## 开发者信息

如有问题或建议，请提交 Issue 或 Pull Request。