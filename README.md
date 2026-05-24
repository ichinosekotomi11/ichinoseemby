# Eden Emby Admin

一个面向 Emby 网页 Docker 管理场景的安全后端骨架，重点覆盖：

- A/B/C 用户等级体系
- 定时开号、开放注册与金币管理
- MoviePilot、抽奖、注册、签到等功能分级权限开关
- AES-256-GCM 敏感数据加密
- bcrypt 本地密码哈希
- JWT 登录态
- B 级用户金币不足时的 Emby 禁用/删除与加密审计记录

## 快速开始

```bash
export EDEN_ADMIN_TOKEN="change-me"
export EDEN_DATA_KEY="$(openssl rand -base64 32)"
export EMBY_BASE_URL="https://emby.example.com"
export EMBY_API_KEY="your-emby-api-key"

go run ./cmd/eden-admin
```

默认监听 `:8080`，默认数据文件为 `data/eden-admin.json`。

## 核心文件

- `migrations/001_security_rbac.sql`：数据库表结构。
- `docs/security-rbac-design.md`：安全与权限设计说明。
- `internal/admin/crypto.go`：AES-256-GCM、bcrypt、数据库字段自动加解密。
- `internal/admin/rbac.go`：功能分级权限中间件。
- `internal/admin/lifecycle.go`：B 级金币不足降级为 C 级及 Emby 安全处置流程。
- `internal/admin/emby.go`：Emby 管理 API 客户端。

## 敏感数据规范

以下字段必须加密存储：

- Telegram ID
- Emby 用户名
- Emby 密码
- MoviePilot API Key
- Emby API Key
- 其他第三方凭证

本地登录密码只保存 bcrypt 哈希，JWT 载荷不包含任何敏感明文。
# ichinoseemby
