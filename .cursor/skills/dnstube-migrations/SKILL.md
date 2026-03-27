---
name: dnstube-migrations
description: Adds or changes PostgreSQL schema and store code for DnsTube. Use when editing migrations, sql, or repository types.
---

# DnsTube 数据库迁移

## 步骤

1. 在 `internal/store/migrations/` 新增 `NNNN_description.up.sql` / `.down.sql`（与 golang-migrate 命名一致）。
2. 更新 `internal/store/models.go` 的 struct 与 `json` tag。
3. 在对应 `internal/store/*.go` 中增加或调整查询方法。
4. 应用启动时会自动执行迁移；本地可用 `DATABASE_URL` 启动一次进程验证。

## 注意

- 与 `forward_rules`、`query_logs` 等大表变更时考虑索引与后续分区。
