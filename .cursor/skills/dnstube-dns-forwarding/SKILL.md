---
name: dnstube-dns-forwarding
description: Implements or debugs DNS forwarding, caching, and forward rules in DnsTube. Use when changing internal/dns or forward semantics.
---

# DnsTube DNS 转发

## 语义（产品已确认）

- **并行**：多上游竞速，**首个有效响应即返回**（其余交换取消），与 AGENTS「最快有效解析」一致。
- **顺序**：按 `sort_order` 尝试，**首个有效响应即停止**。
- **静态记录**优先于转发；无上游配置时返回 SERVFAIL。

## 代码位置

- 核心：`internal/dns/handler.go`、`forward.go`、`engine.go`、`cache.go`。
- 数据：`internal/store` 与迁移中的 `dns_records`、`forward_rules`、`upstream_*`。

## 调试

- 确认实例未 `paused`，监听地址/端口与防火墙一致。
- 管理 API 变更后引擎 `SyncNow` 会同步 UDP 监听。
