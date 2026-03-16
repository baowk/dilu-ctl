# dilu-ctl v0.0.4 Release Notes

发布日期：2026-03-16  
版本标签：`v0.0.4`

## 本次重点

1. 版本命令  
   - 新增 `dilu-ctl version`  
   - 支持 `-ldflags` 注入版本号

2. 数据库适配增强  
   - PostgreSQL 支持 `schema.table`  
   - DSN 自动识别增强（支持 `host=... dbname=...`）

3. 生成链路更稳定  
   - Service/DTO/API 字段与类型完全来自 GORM-Gen 模型  
   - 生成路径统一为 `internal/modules/<module>`

## 构建示例
```bash
go build -ldflags "-X 'github.com/baowk/dilu-ctl/cmd.Version=v0.0.4'" -o dilu-ctl
```
