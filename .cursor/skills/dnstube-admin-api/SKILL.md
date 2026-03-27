---
name: dnstube-admin-api
description: Work on DnsTube admin REST API, JWT, or JSON DTOs. Use when changing internal/api or auth behavior.
---

# DnsTube 管理 API

- 入口：`internal/api/server.go` 注册路由；各 `*_handlers.go` 实现逻辑。
- 响应：`success`、`code`（`0` 成功，`APICodeNum` 映射非 `0`）、`message`；列表另含 `page`、`total`，`data` 必为数组。错误码字符串见 `errcode.go`，文案用 `msgForCode`，勿在 handler 硬编码。详见 `.cursor/rules/dnstube-api.mdc`。
- 写库后配置落地：`sync` → `Engine.SyncNow`；绑定失败返回 `DNS_UDP_BIND_FAILED`。
- 鉴权：`internal/api/middleware.go` 与 `internal/auth/jwt.go`。
- 变更路由时同步更新 `web/src/api/index.ts`、`client.ts`、相关页面。
