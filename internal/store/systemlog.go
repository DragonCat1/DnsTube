package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	LogKindSystem = "system"
	LogKindAudit  = "audit"
)

// InsertLogEntry 写入一条系统或审计日志。
func (s *Store) InsertLogEntry(ctx context.Context, kind, level string, event *string, message string, username, clientIP *string, meta map[string]any) error {
	if kind != LogKindSystem && kind != LogKindAudit {
		kind = LogKindSystem
	}
	if level == "" {
		level = "info"
	}
	var metaBytes []byte
	if len(meta) > 0 {
		var err error
		metaBytes, err = json.Marshal(meta)
		if err != nil {
			return err
		}
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO system_logs (kind, level, event, message, username, client_ip, meta)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		kind, level, event, message, username, clientIP, metaBytes,
	)
	return err
}

type SystemLogFilter struct {
	Kind      *string
	Level     *string
	Event     *string
	From      *time.Time
	To        *time.Time
	Message   *string
	Username  *string
	ClientIP  *string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

func (s *Store) ListSystemLogs(ctx context.Context, f SystemLogFilter) ([]SystemLog, int64, error) {
	if f.Limit > 0 && f.Limit > 500 {
		f.Limit = 500
	}
	if f.Limit > 0 && f.Offset < 0 {
		f.Offset = 0
	}
	var conds []string
	var args []any
	arg := 1
	if f.Kind != nil && strings.TrimSpace(*f.Kind) != "" {
		conds = append(conds, fmt.Sprintf("kind = $%d", arg))
		args = append(args, strings.TrimSpace(*f.Kind))
		arg++
	}
	if f.Level != nil && strings.TrimSpace(*f.Level) != "" {
		conds = append(conds, fmt.Sprintf("level = $%d", arg))
		args = append(args, strings.TrimSpace(*f.Level))
		arg++
	}
	if f.Event != nil && strings.TrimSpace(*f.Event) != "" {
		conds = append(conds, fmt.Sprintf("event ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*f.Event)+"%")
		arg++
	}
	if f.From != nil {
		conds = append(conds, fmt.Sprintf("created_at >= $%d", arg))
		args = append(args, *f.From)
		arg++
	}
	if f.To != nil {
		conds = append(conds, fmt.Sprintf("created_at <= $%d", arg))
		args = append(args, *f.To)
		arg++
	}
	if f.Message != nil && strings.TrimSpace(*f.Message) != "" {
		conds = append(conds, fmt.Sprintf("message ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*f.Message)+"%")
		arg++
	}
	if f.Username != nil && strings.TrimSpace(*f.Username) != "" {
		conds = append(conds, fmt.Sprintf("username ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*f.Username)+"%")
		arg++
	}
	if f.ClientIP != nil && strings.TrimSpace(*f.ClientIP) != "" {
		conds = append(conds, fmt.Sprintf("client_ip ILIKE $%d", arg))
		args = append(args, "%"+strings.TrimSpace(*f.ClientIP)+"%")
		arg++
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	countSQL := "SELECT COUNT(*) FROM system_logs " + where
	var total int64
	if err := s.Pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderCol := "created_at"
	switch f.SortBy {
	case "id":
		orderCol = "id"
	case "created_at":
		orderCol = "created_at"
	case "kind":
		orderCol = "kind"
	case "level":
		orderCol = "level"
	case "event":
		orderCol = "COALESCE(event, '')"
	case "message":
		orderCol = "message"
	case "username":
		orderCol = "COALESCE(username, '')"
	case "client_ip":
		orderCol = "COALESCE(client_ip, '')"
	default:
		orderCol = "created_at"
	}
	dir := "DESC"
	if strings.EqualFold(f.SortOrder, "asc") {
		dir = "ASC"
	}
	orderBy := fmt.Sprintf("%s %s NULLS LAST", orderCol, dir)
	if orderCol != "id" {
		orderBy = fmt.Sprintf("%s, id %s", orderBy, dir)
	}
	var q string
	if f.Limit == 0 {
		q = fmt.Sprintf(`
		SELECT id, created_at, kind, level, event, message, username, client_ip, meta
		FROM system_logs
		%s
		ORDER BY %s`, where, orderBy)
	} else {
		args = append(args, f.Limit, f.Offset)
		limitArg := arg
		offsetArg := arg + 1
		q = fmt.Sprintf(`
		SELECT id, created_at, kind, level, event, message, username, client_ip, meta
		FROM system_logs
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, orderBy, limitArg, offsetArg)
	}
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []SystemLog
	for rows.Next() {
		var l SystemLog
		var ev, un, ip sql.NullString
		var metaBytes []byte
		if err := rows.Scan(&l.ID, &l.CreatedAt, &l.Kind, &l.Level, &ev, &l.Message, &un, &ip, &metaBytes); err != nil {
			return nil, 0, err
		}
		if ev.Valid {
			s := ev.String
			l.Event = &s
		}
		if un.Valid {
			s := un.String
			l.Username = &s
		}
		if ip.Valid {
			s := ip.String
			l.ClientIP = &s
		}
		if len(metaBytes) > 0 {
			l.Meta = json.RawMessage(append([]byte(nil), metaBytes...))
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

func (s *Store) DeleteSystemLogs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	ct, err := s.Pool.Exec(ctx, `DELETE FROM system_logs WHERE id = ANY($1::bigint[])`, ids)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}
