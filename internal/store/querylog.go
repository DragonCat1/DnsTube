package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type QueryLogInsert struct {
	InstanceID             int32
	ClientIP               string
	Qname                  string
	Qtype                  string
	ResponseCode           string
	CacheHit               bool
	Forwarded              bool
	UpstreamAddr           *string
	UpstreamMs             *int
	TotalMs                *int
	ErrorMessage           *string
	ResultSummary          *string
	ForwardUpstreamGroupID *int32
}

func (s *Store) InsertQueryLog(ctx context.Context, q QueryLogInsert) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO query_logs (
			instance_id, client_ip, qname, qtype, response_code, cache_hit, forwarded,
			upstream_addr, upstream_ms, total_ms, error_message, result_summary,
			forward_upstream_group_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		q.InstanceID, q.ClientIP, q.Qname, q.Qtype, q.ResponseCode, q.CacheHit, q.Forwarded,
		q.UpstreamAddr, q.UpstreamMs, q.TotalMs, q.ErrorMessage, q.ResultSummary,
		q.ForwardUpstreamGroupID,
	)
	return err
}

// queryLogBatchChunk 单条 INSERT 参数个数为 13；PostgreSQL 单语句参数上限约 65535，留余量按块写入。
const queryLogBatchChunk = 500

// InsertQueryLogsBatch 批量写入查询日志（单事务）；失败时由调用方整批丢弃，不做重试。
func (s *Store) InsertQueryLogsBatch(ctx context.Context, rows []QueryLogInsert) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for off := 0; off < len(rows); off += queryLogBatchChunk {
		end := off + queryLogBatchChunk
		if end > len(rows) {
			end = len(rows)
		}
		if err := insertQueryLogsBatchChunkTx(ctx, tx, rows[off:end]); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func insertQueryLogsBatchChunkTx(ctx context.Context, tx pgx.Tx, rows []QueryLogInsert) error {
	if len(rows) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO query_logs (
		instance_id, client_ip, qname, qtype, response_code, cache_hit, forwarded,
		upstream_addr, upstream_ms, total_ms, error_message, result_summary,
		forward_upstream_group_id
	) VALUES `)
	args := make([]any, 0, len(rows)*13)
	arg := 1
	for i, q := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			arg, arg+1, arg+2, arg+3, arg+4, arg+5, arg+6, arg+7, arg+8, arg+9, arg+10, arg+11, arg+12)
		arg += 13
		args = append(args, q.InstanceID, q.ClientIP, q.Qname, q.Qtype, q.ResponseCode, q.CacheHit, q.Forwarded,
			q.UpstreamAddr, q.UpstreamMs, q.TotalMs, q.ErrorMessage, q.ResultSummary, q.ForwardUpstreamGroupID)
	}
	_, err := tx.Exec(ctx, b.String(), args...)
	return err
}

type QueryLogFilter struct {
	InstanceID *int32
	From       *time.Time
	To         *time.Time
	// ClientIP 模糊匹配（ILIKE %...%）
	ClientIP *string
	Qname    *string
	Qtype    *string
	Rcode    *string
	// ForwardUpstreamGroupID 精确匹配；nil 表示不按转发组筛。
	ForwardUpstreamGroupID *int32
	Limit                  int
	Offset                 int
	// SortBy: id, created_at, client_ip, qtype, instance_id, forward_group, total_ms,
	// response_code, result_summary, cache_hit, forwarded, upstream_addr
	SortBy string
	// SortOrder: asc, desc
	SortOrder string
}

func (s *Store) ListQueryLogs(ctx context.Context, f QueryLogFilter) ([]QueryLogRow, int64, error) {
	// Limit==0：不分页，返回全部匹配行（仍受下面 SQL 无 LIMIT 分支处理）。
	// Limit>0：分页，单页最大 500。
	if f.Limit > 0 && f.Limit > 500 {
		f.Limit = 500
	}
	if f.Limit > 0 && f.Offset < 0 {
		f.Offset = 0
	}
	var conds []string
	var args []any
	arg := 1
	if f.InstanceID != nil {
		conds = append(conds, fmt.Sprintf("ql.instance_id = $%d", arg))
		args = append(args, *f.InstanceID)
		arg++
	}
	if f.From != nil {
		conds = append(conds, fmt.Sprintf("ql.created_at >= $%d", arg))
		args = append(args, *f.From)
		arg++
	}
	if f.To != nil {
		conds = append(conds, fmt.Sprintf("ql.created_at <= $%d", arg))
		args = append(args, *f.To)
		arg++
	}
	if f.ClientIP != nil && strings.TrimSpace(*f.ClientIP) != "" {
		conds = append(conds, fmt.Sprintf("ql.client_ip ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*f.ClientIP)+"%")
		arg++
	}
	if f.Qname != nil && strings.TrimSpace(*f.Qname) != "" {
		conds = append(conds, fmt.Sprintf("ql.qname ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*f.Qname)+"%")
		arg++
	}
	if f.Qtype != nil && strings.TrimSpace(*f.Qtype) != "" {
		conds = append(conds, fmt.Sprintf("ql.qtype ILIKE $%d", arg))
		args = append(args, strings.TrimSpace(*f.Qtype))
		arg++
	}
	if f.Rcode != nil && strings.TrimSpace(*f.Rcode) != "" {
		conds = append(conds, fmt.Sprintf("ql.response_code ILIKE $%d", arg))
		args = append(args, strings.TrimSpace(*f.Rcode))
		arg++
	}
	if f.ForwardUpstreamGroupID != nil {
		conds = append(conds, fmt.Sprintf("ql.forward_upstream_group_id = $%d", arg))
		args = append(args, *f.ForwardUpstreamGroupID)
		arg++
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	fromJoin := `FROM query_logs ql
		JOIN dns_instances di ON di.id = ql.instance_id
		LEFT JOIN upstream_groups ug ON ug.id = ql.forward_upstream_group_id`
	countSQL := "SELECT COUNT(*) " + fromJoin + " " + where
	var total int64
	if err := s.Pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderCol := "ql.created_at"
	switch f.SortBy {
	case "id":
		orderCol = "ql.id"
	case "created_at":
		orderCol = "ql.created_at"
	case "client_ip":
		orderCol = "ql.client_ip"
	case "qtype":
		orderCol = "ql.qtype"
	case "instance_id":
		orderCol = "ql.instance_id"
	case "forward_group":
		orderCol = "COALESCE(ug.name, '')"
	case "total_ms":
		orderCol = "ql.total_ms"
	case "response_code":
		orderCol = "ql.response_code"
	case "result_summary":
		orderCol = "ql.result_summary"
	case "cache_hit":
		orderCol = "ql.cache_hit"
	case "forwarded":
		orderCol = "ql.forwarded"
	case "upstream_addr":
		orderCol = "COALESCE(ql.upstream_addr, '')"
	default:
		orderCol = "ql.created_at"
	}
	dir := "DESC"
	if strings.EqualFold(f.SortOrder, "asc") {
		dir = "ASC"
	}
	orderBy := fmt.Sprintf("%s %s NULLS LAST", orderCol, dir)
	if orderCol != "ql.id" {
		orderBy = fmt.Sprintf("%s, ql.id %s", orderBy, dir)
	}
	var q string
	if f.Limit == 0 {
		q = fmt.Sprintf(`
		SELECT ql.id, ql.created_at, ql.instance_id, ql.client_ip, ql.qname, ql.qtype, ql.response_code, ql.cache_hit, ql.forwarded,
		       ql.upstream_addr, ql.upstream_ms, ql.total_ms, ql.error_message, ql.result_summary, ql.forward_upstream_group_id,
		       di.name, COALESCE(ug.name, '')
		%s
		%s
		ORDER BY %s`, fromJoin, where, orderBy)
	} else {
		args = append(args, f.Limit, f.Offset)
		limitArg := arg
		offsetArg := arg + 1
		q = fmt.Sprintf(`
		SELECT ql.id, ql.created_at, ql.instance_id, ql.client_ip, ql.qname, ql.qtype, ql.response_code, ql.cache_hit, ql.forwarded,
		       ql.upstream_addr, ql.upstream_ms, ql.total_ms, ql.error_message, ql.result_summary, ql.forward_upstream_group_id,
		       di.name, COALESCE(ug.name, '')
		%s
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, fromJoin, where, orderBy, limitArg, offsetArg)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []QueryLogRow
	for rows.Next() {
		var row QueryLogRow
		var l QueryLog
		var upAddr, errMsg, resSum sql.NullString
		var upMs, totMs sql.NullInt32
		var fwdGid sql.NullInt32
		if err := rows.Scan(
			&l.ID, &l.CreatedAt, &l.InstanceID, &l.ClientIP, &l.Qname, &l.Qtype, &l.ResponseCode,
			&l.CacheHit, &l.Forwarded, &upAddr, &upMs, &totMs, &errMsg, &resSum, &fwdGid,
			&row.InstanceName, &row.ForwardGroupName,
		); err != nil {
			return nil, 0, err
		}
		if upAddr.Valid {
			s := upAddr.String
			l.UpstreamAddr = &s
		}
		if upMs.Valid {
			v := int(upMs.Int32)
			l.UpstreamMs = &v
		}
		if totMs.Valid {
			v := int(totMs.Int32)
			l.TotalMs = &v
		}
		if errMsg.Valid {
			s := errMsg.String
			l.ErrorMessage = &s
		}
		if resSum.Valid {
			s := resSum.String
			l.ResultSummary = &s
		}
		if fwdGid.Valid {
			v := fwdGid.Int32
			l.ForwardUpstreamGroupID = &v
		}
		row.QueryLog = l
		out = append(out, row)
	}
	return out, total, rows.Err()
}

func (s *Store) DeleteQueryLogs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ct, err := s.Pool.Exec(ctx, `DELETE FROM query_logs WHERE id = ANY($1::bigint[])`, ids)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

type DashboardPoint struct {
	Bucket time.Time `json:"bucket"`
	Count  int64     `json:"count"`
}

type TopClient struct {
	ClientIP string `json:"client_ip"`
	Count    int64  `json:"count"`
}

// buildDashboardWhere 生成 query_logs 时间窗 + 可选实例 + 可选客户端 IP（ILIKE 子串）条件。
func buildDashboardWhere(instanceID *int32, from, to time.Time, clientIP *string) (where string, args []any) {
	var conds []string
	arg := 1
	if instanceID != nil {
		conds = append(conds, fmt.Sprintf("instance_id = $%d", arg))
		args = append(args, *instanceID)
		arg++
	}
	conds = append(conds, fmt.Sprintf("created_at >= $%d", arg))
	args = append(args, from)
	arg++
	conds = append(conds, fmt.Sprintf("created_at <= $%d", arg))
	args = append(args, to)
	arg++
	if clientIP != nil && strings.TrimSpace(*clientIP) != "" {
		conds = append(conds, fmt.Sprintf("client_ip ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*clientIP)+"%")
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func (s *Store) DashboardSeries(ctx context.Context, instanceID *int32, from, to time.Time, bucket string, clientIP *string) ([]DashboardPoint, error) {
	trunc := "day"
	switch bucket {
	case "week":
		trunc = "week"
	case "month":
		trunc = "month"
	case "year":
		trunc = "year"
	}
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	// 以数据库会话时区进行分桶，避免固定 UTC 导致本地时区展示偏移。
	q := `SELECT date_trunc('` + trunc + `', created_at) AS b, COUNT(*)::bigint
		FROM query_logs ` + where + ` GROUP BY b ORDER BY b`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DashboardPoint
	for rows.Next() {
		var p DashboardPoint
		if err := rows.Scan(&p.Bucket, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) DashboardTopClients(ctx context.Context, instanceID *int32, from, to time.Time, n int, clientIP *string) ([]TopClient, error) {
	if n <= 0 || n > 100 {
		n = 10
	}
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	limitArg := len(args) + 1
	args = append(args, n)
	q := `SELECT client_ip, COUNT(*)::bigint AS c FROM query_logs ` + where + ` GROUP BY client_ip ORDER BY c DESC LIMIT $` + strconv.Itoa(limitArg)
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopClient
	for rows.Next() {
		var t TopClient
		if err := rows.Scan(&t.ClientIP, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CountQueriesInRange(ctx context.Context, instanceID *int32, from, to time.Time, clientIP *string) (int64, error) {
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	q := `SELECT COUNT(*)::bigint FROM query_logs ` + where
	var n int64
	err := s.Pool.QueryRow(ctx, q, args...).Scan(&n)
	return n, err
}

// CountQueryLogsAll 查询日志总条数（不限时间）。instanceID 非 nil 时仅统计该实例。
func (s *Store) CountQueryLogsAll(ctx context.Context, instanceID *int32) (int64, error) {
	if instanceID != nil {
		var n int64
		err := s.Pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM query_logs WHERE instance_id = $1`, *instanceID).Scan(&n)
		return n, err
	}
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*)::bigint FROM query_logs`).Scan(&n)
	return n, err
}

// TopQName 查询日志中按次数排序的域名。
type TopQName struct {
	Qname string `json:"qname"`
	Count int64  `json:"count"`
}

// QtypeCount 按查询类型聚合。
type QtypeCount struct {
	Qtype string `json:"qtype"`
	Count int64  `json:"count"`
}

// RcodeCount 按应答码聚合。
type RcodeCount struct {
	ResponseCode string `json:"response_code"`
	Count        int64  `json:"count"`
}

// CacheForwardBreakdown 缓存命中 / 转发 / 其余（如静态命中后首次等）计数。
type CacheForwardBreakdown struct {
	Total     int64 `json:"total"`
	CacheHits int64 `json:"cache_hits"`
	Forwarded int64 `json:"forwarded"`
	Other     int64 `json:"other"`
}

func clampDashboardTopN(n int) int {
	if n <= 0 {
		return 10
	}
	if n > 50 {
		return 50
	}
	return n
}

// DashboardTopQnames 区间内 TOP 域名（按 qname 聚合）。
func (s *Store) DashboardTopQnames(ctx context.Context, instanceID *int32, from, to time.Time, n int, clientIP *string) ([]TopQName, error) {
	n = clampDashboardTopN(n)
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	limitArg := len(args) + 1
	args = append(args, n)
	q := `SELECT qname, COUNT(*)::bigint AS c FROM query_logs ` + where + ` GROUP BY qname ORDER BY c DESC LIMIT $` + strconv.Itoa(limitArg)
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TopQName
	for rows.Next() {
		var t TopQName
		if err := rows.Scan(&t.Qname, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DashboardQtypeDistribution 区间内各 qtype 条数。
func (s *Store) DashboardQtypeDistribution(ctx context.Context, instanceID *int32, from, to time.Time, clientIP *string) ([]QtypeCount, error) {
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	q := `SELECT qtype, COUNT(*)::bigint AS c FROM query_logs ` + where + ` GROUP BY qtype ORDER BY c DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QtypeCount
	for rows.Next() {
		var t QtypeCount
		if err := rows.Scan(&t.Qtype, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DashboardRcodeDistribution 区间内各 response_code 条数。
func (s *Store) DashboardRcodeDistribution(ctx context.Context, instanceID *int32, from, to time.Time, clientIP *string) ([]RcodeCount, error) {
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	q := `SELECT response_code, COUNT(*)::bigint AS c FROM query_logs ` + where + ` GROUP BY response_code ORDER BY c DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RcodeCount
	for rows.Next() {
		var t RcodeCount
		if err := rows.Scan(&t.ResponseCode, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DashboardCacheForwardBreakdown 缓存命中、转发及其余查询计数。
func (s *Store) DashboardCacheForwardBreakdown(ctx context.Context, instanceID *int32, from, to time.Time, clientIP *string) (CacheForwardBreakdown, error) {
	where, args := buildDashboardWhere(instanceID, from, to, clientIP)
	q := `
		SELECT COUNT(*)::bigint,
			COUNT(*) FILTER (WHERE cache_hit)::bigint,
			COUNT(*) FILTER (WHERE forwarded)::bigint,
			COUNT(*) FILTER (WHERE NOT cache_hit AND NOT forwarded)::bigint
		FROM query_logs ` + where
	var b CacheForwardBreakdown
	err := s.Pool.QueryRow(ctx, q, args...).Scan(
		&b.Total, &b.CacheHits, &b.Forwarded, &b.Other,
	)
	return b, err
}

// ErrNotFound for API mapping
var ErrNotFound = pgx.ErrNoRows
