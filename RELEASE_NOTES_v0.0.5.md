# dilu-ctl v0.0.5 Release Notes

发布日期：2026-03-16  
版本标签：`v0.0.5`

## 本次重点

1. 版本显示优化  
   - `dilu-ctl version` 优先读取 Go build info  
   - 通过 `go install ...@v0.0.5` 安装时可直接显示版本号  
   - 仍兼容 `-ldflags` 注入版本号

## 构建示例
```bash
go install github.com/baowk/dilu-ctl@v0.0.5

# 或显式注入
go build -ldflags "-X 'github.com/baowk/dilu-ctl/cmd.Version=v0.0.5'" -o dilu-ctl
```
