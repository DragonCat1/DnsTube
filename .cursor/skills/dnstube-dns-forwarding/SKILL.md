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

## 配置热更新

- **无周期全量轮询**：引擎不以固定间隔从 DB Reload。
- **触发点**：进程启动；DNS 相关管理 API 写成功后的 `SyncNow`；URL 规则拉取写库后的 `Reload`/`SyncNow`。
- **不 SyncNow**：管理员改名/改密、上游组仅创建或改名（不影响解析 snap）；上游组删除仍 SyncNow。
- **Reload 串行**：`Engine.Reload` 全程 `reloadMu`，避免 API SyncNow 与定时 URL 拉取等并发 Reload 旧结果覆盖新 snap。
- **编译复用**：Reload 按规则内容指纹复用已编译 regexp/domain；指纹不含 `pattern_fetched_at`（URL 拉取正文未变时不重编译）。
- **UDP 绑定失败**：记入实例 `runtime_error`；**无进程内定时重试**，需下次 `SyncNow`（管理端再改 DNS 相关配置）或进程重启后再绑。
- **系统日志**：URL 拉取成功记 `forward rule pattern fetch success`（含 old_count/new_count/delta/content_changed/source）；`syncListeners` 成功记 `dns listeners synced`（含 changed/wanted/running/started/stopped）。

## 调试

- 确认实例未 `paused`，监听地址/端口与防火墙一致。
- 管理 API 变更后引擎 `SyncNow` 会同步 UDP 监听。
- 启动后实例长期 `runtime_error`（端口占用等）：排除占用后通过任意 DNS 相关配置写入触发 SyncNow，或重启进程。
