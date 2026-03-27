# DnsTube — Agent / 贡献者指南

## 架构

- **单进程**：`cmd/dnstube` 同时启动 **UDP DNS**（每实例一监听）与 **HTTP 管理 API**。
- **分层**：`internal/dns`（解析、缓存、转发）→ 不直接写 SQL；`internal/store`（PostgreSQL）；`internal/api`（REST、DTO）；`internal/auth`（JWT、密码）。
- **配置**：运行时以 **PostgreSQL** 为准；无配置文件热重载。

## 目录索引

| 需求                 | 目录                                            |
| -------------------- | ----------------------------------------------- |
| 转发、缓存、规则匹配 | `internal/dns/`                                 |
| 管理 REST、路由      | `internal/api/`                                 |
| 表与 repository      | `internal/store/`、`internal/store/migrations/` |
| 入口与配置           | `cmd/dnstube/`、`internal/config/`              |
| 管理后台 UI          | `web/`                                          |

## Web 管理端（Vue）

- **日期时间展示**：所有面向用户的日期+时间统一为 **`YYYY-MM-DD HH:mm:ss`**（24 小时制，浏览器本地时区）。使用 `web/src/datetime.ts` 的 `formatDateTime` / `DATE_TIME_FORMAT`，勿用 `toLocaleString()` 等与运行环境强绑定的格式。Cursor 规则见 `.cursor/rules/dnstube-web.mdc`。

## 已决行为（摘要）

- **并行转发**：多上游竞速，取最快有效解析；超时/防投毒按通用实践。
- **顺序转发**：组内顺序尝试，**首个解析结果即停**。
- **静态记录优先**于转发。
- **首启管理员**：无账号时创建用户 `admin` / 密码 `admin`，并 **打日志**（登录后请改密）。
- **查询日志**：多字段、永久存 DB；管理端 **查删**；Dashboard 查询量 = **客户端请求次数**。

## 常用命令

```bash
make tidy          # go mod tidy
make build         # 构建 dnstube 二进制
make dev-api       # 需本地 PostgreSQL 与 DATABASE_URL
make dev-web       # 前端开发服务器
```

环境变量见 [`.env.example`](.env.example)。

## API

- 前缀：`/api/v1/`
- 鉴权：`Authorization: Bearer <JWT>`（除 `POST /api/v1/auth/login`）
- 响应：`success`、`code`（`0` 成功，非 `0` 为 `APICodeNum` 数值）、`message`；**对象**接口 `data` 为对象或 `null`；**列表**接口另含 `page`、`total`，且 `data` 必为数组（见 `.cursor/rules/dnstube-api.mdc`）。字符串错误码仍定义在 `internal/api/errcode.go`，文案由 `msgForCode` 映射。

## 禁止

- 在 `internal/dns` 中写 SQL。
- 无必要的大范围重构；改动与 issue/计划对齐。
