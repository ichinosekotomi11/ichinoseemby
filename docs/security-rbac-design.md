# Eden Emby 管理后端安全重构设计

## 用户等级

| 等级 | 定义 | 生命周期 | 默认权限 |
| --- | --- | --- | --- |
| A | 白名单用户 | 永久有效，豁免扣币、到期、删号、禁用检查 | 解锁全部高级功能 |
| B | 已绑定 Emby 用户 | 参与周期扣币；金币不足或到期后降级为 C | 正常使用 Emby、MoviePilot 点播、签到 |
| C | 未绑定或过期用户 | 无 Emby 服务权限，可重新申请 | 签到、充值、申请/重新绑定、查看公告 |

## 数据库表结构

核心 SQL 在 [`migrations/001_security_rbac.sql`](../migrations/001_security_rbac.sql)。

重点变化：

- `users.level` 使用 `A/B/C` 字符串枚举。
- `telegram_id_enc`、`emby_username_enc`、`emby_password_enc` 只保存 AES-256-GCM 加密后的 Base64 文本。
- `password_hash` 只保存 bcrypt 单向哈希，不保存可逆密码。
- `system_configs.value_json` 保存动态功能权限矩阵。
- `sensitive_configs.value_enc` 保存 Emby ApiKey、MoviePilot ApiKey 等第三方凭证。
- `security_events.encrypted_note` 保存加密后的安全事件上下文。

## 加密与自动读写

代码在 [`internal/admin/crypto.go`](../internal/admin/crypto.go)。

启动时读取环境变量：

```bash
export EDEN_DATA_KEY="$(openssl rand -base64 32)"
```

初始化：

```go
cipher, err := admin.LoadFieldCipherFromEnv()
if err != nil {
    panic(err)
}
admin.SetDefaultFieldCipher(cipher)
```

数据库模型使用 `EncryptedString`：

```go
type SQLUser struct {
    TelegramID   admin.EncryptedString `db:"telegram_id_enc"`
    EmbyUsername admin.EncryptedString `db:"emby_username_enc"`
    EmbyPassword admin.EncryptedString `db:"emby_password_enc"`
}
```

`EncryptedString` 实现了 `Scan` 和 `Value`：

- 写库时自动调用 `Value()`，将明文加密为 Base64 密文。
- 读库时自动调用 `Scan()`，将密文解密到 `Plain`。

## JWT 约束

JWT 只包含：

- `sub`：用户 ID。
- `lvl`：用户等级。
- `iat` / `exp`：签发与过期时间。

JWT 不包含 Telegram ID、Emby 用户名、Emby 密码、ApiKey、金币明细等敏感信息。

## 功能权限矩阵

代码在 [`internal/admin/rbac.go`](../internal/admin/rbac.go)。

示例策略：

```json
{
  "moviepilot.request": {
    "enabled": true,
    "allowed_levels": ["A", "B"],
    "coin_cost": 10,
    "level_cost_multiplier": {"A": 0, "B": 1, "C": 2}
  },
  "checkin.daily": {
    "enabled": true,
    "allowed_levels": ["A", "B", "C"],
    "level_reward": {"A": 10, "B": 5, "C": 2}
  }
}
```

中间件用法：

```go
mux.Handle(
    "/api/moviepilot/request",
    admin.AuthorizeFeature(store, "moviepilot.request", moviePilotHandler),
)
```

## B 级金币不足降级流程

代码在 [`internal/admin/lifecycle.go`](../internal/admin/lifecycle.go)。

流程说明：

1. 定时任务扫描 B 级用户，A 级直接跳过。
2. 如果金币足够，扣除周期金币并写入金币流水。
3. 如果金币不足，先调用 Emby API 处理账号：
   - 默认禁用：`POST /Users/{Id}/Policy`，设置 `IsDisabled=true`。
   - 如策略要求彻底删除，可调用 `DELETE /Users/{Id}`。
4. Emby 处理成功后，本地用户等级从 B 改为 C。
5. 写入金币流水和 `security_events`。
6. `security_events.encrypted_note` 使用 AES-256-GCM 加密保存降级原因、Emby 动作、原等级、现等级、金币余额等上下文。

这样做的安全收益：

- Emby 调用失败时不提交本地降级，避免本地显示 C 级但 Emby 仍可用的状态错位。
- 审计事件不暴露 Emby UserID 等敏感上下文。
- A 级白名单不会被定时任务误伤。

## 参考

- Emby 用户管理文档：[Users](https://support.emby.media/support/articles/Users.html)
- Emby 用户策略接口：[POST /Users/{Id}/Policy](https://dev.emby.media/reference/RestAPI/UserService/postUsersByIdPolicy.html)
- MoviePilot API 入口：[Swagger UI](https://api.movie-pilot.org/)
