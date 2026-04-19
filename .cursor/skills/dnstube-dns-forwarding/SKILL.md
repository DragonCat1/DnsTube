---
name: dnstube-dns-forwarding
description: Implements or debugs DNS forwarding, caching, and forward rules in DnsTube. Use when changing internal/dns or forward semantics.
---

# DnsTube DNS 转发

## 语义（产品已确认）

- **并行**：多上游竞速，**首个有效响应即返回**（其余交换取消），与 AGENTS「最快有效解析」一致。
- **顺序**：按 `sort_order` 尝试，**首个有效响应即停止**。
- **静态记录**优先于转发；无上游配置时返回 SERVFAIL。

## 上游协议

- 支持 **UDP / DoT / DoH** 三种传输；`store.UpstreamServer.Protocol` 字段决定走哪条分支。
- 地址只接受 **IP 字面量**（IPv4/IPv6），避免引入 bootstrap DNS；DoT/DoH 可填可选 `tls_server_name` 做 SNI；DoH 端点路径默认 `/dns-query`。
- 进程级 `*ExchangeClient` 同时持有 `*mdns.Client` 与 `*http.Client`（HTTP/2 + keep-alive），由 `Engine` 注入，避免每次查询重建 TLS 与 HTTP 连接。
- 失败信息会带协议化前缀（`udp://`、`tls://`、`https://...`），便于在 `query_logs.error_message` 中定位是哪一台/哪种协议的上游异常。

## 代码位置

- 核心：`internal/dns/handler.go`、`forward.go`、`engine.go`、`cache.go`。
- 数据：`internal/store` 与迁移中的 `dns_records`、`forward_rules`、`upstream_*`（`upstream_servers` 含 `protocol/path/tls_server_name` 列）。

## 调试

- 确认实例未 `paused`，监听地址/端口与防火墙一致。
- 管理 API 变更后引擎 `SyncNow` 会同步 UDP 监听。
