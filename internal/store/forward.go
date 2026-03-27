package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
)

// ListForwardRulesSummary 供管理 API 列表：不含 pattern_fetched_body，避免大字段。
func (s *Store) ListForwardRulesSummary(ctx context.Context, instanceID int32) ([]ForwardRule, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, instance_id, name_pattern, pattern_source, pattern_url, pattern_format,
		       target_group_id, mode, priority, hit_count, disabled, created_at,
		       pattern_rule_count, pattern_fetched_at
		FROM forward_rules WHERE instance_id = $1 ORDER BY priority, id`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanForwardRuleRows(rows, false)
}

// ListForwardRules 供 DNS 引擎加载，含 pattern_fetched_body。
func (s *Store) ListForwardRules(ctx context.Context, instanceID int32) ([]ForwardRule, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, instance_id, name_pattern, pattern_source, pattern_url, pattern_format,
		       target_group_id, mode, priority, hit_count, disabled, created_at,
		       pattern_fetched_body, pattern_rule_count, pattern_fetched_at
		FROM forward_rules WHERE instance_id = $1 ORDER BY priority, id`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanForwardRuleRows(rows, true)
}

func scanForwardRuleRows(rows pgx.Rows, withBody bool) ([]ForwardRule, error) {
	var out []ForwardRule
	for rows.Next() {
		var r ForwardRule
		var url sql.NullString
		var fetchedAt sql.NullTime
		var body sql.NullString
		var err error
		if withBody {
			err = rows.Scan(&r.ID, &r.InstanceID, &r.NamePattern, &r.PatternSource, &url, &r.PatternFormat,
				&r.TargetGroupID, &r.Mode, &r.Priority, &r.HitCount, &r.Disabled, &r.CreatedAt,
				&body, &r.PatternRuleCount, &fetchedAt)
		} else {
			err = rows.Scan(&r.ID, &r.InstanceID, &r.NamePattern, &r.PatternSource, &url, &r.PatternFormat,
				&r.TargetGroupID, &r.Mode, &r.Priority, &r.HitCount, &r.Disabled, &r.CreatedAt,
				&r.PatternRuleCount, &fetchedAt)
		}
		if err != nil {
			return nil, err
		}
		if url.Valid {
			s := url.String
			r.PatternURL = &s
		}
		if fetchedAt.Valid {
			t := fetchedAt.Time
			r.PatternFetchedAt = &t
		}
		if withBody && body.Valid {
			s := body.String
			r.PatternFetchedBody = &s
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetForwardRuleByID(ctx context.Context, id int32) (ForwardRule, error) {
	var r ForwardRule
	var url sql.NullString
	var fetchedAt sql.NullTime
	var body sql.NullString
	var resolved sql.NullString
	err := s.Pool.QueryRow(ctx, `
		SELECT id, instance_id, name_pattern, pattern_source, pattern_url, pattern_format,
		       target_group_id, mode, priority, hit_count, disabled, created_at,
		       pattern_fetched_body, pattern_rule_count, pattern_fetched_at,
		       pattern_resolved_entries::text
		FROM forward_rules WHERE id = $1`, id,
	).Scan(&r.ID, &r.InstanceID, &r.NamePattern, &r.PatternSource, &url, &r.PatternFormat,
		&r.TargetGroupID, &r.Mode, &r.Priority, &r.HitCount, &r.Disabled, &r.CreatedAt,
		&body, &r.PatternRuleCount, &fetchedAt, &resolved)
	if err != nil {
		return ForwardRule{}, err
	}
	if url.Valid {
		s := url.String
		r.PatternURL = &s
	}
	if fetchedAt.Valid {
		t := fetchedAt.Time
		r.PatternFetchedAt = &t
	}
	if body.Valid {
		s := body.String
		r.PatternFetchedBody = &s
	}
	if resolved.Valid && resolved.String != "" {
		r.PatternResolvedEntries = []byte(resolved.String)
	}
	return r, nil
}

// ListForwardRulesURLSource 返回需定期/手动拉取 URL 的规则（未禁用且配置了 URL）。
func (s *Store) ListForwardRulesURLSource(ctx context.Context) ([]ForwardRule, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, instance_id, name_pattern, pattern_source, pattern_url, pattern_format,
		       target_group_id, mode, priority, hit_count, disabled, created_at,
		       pattern_fetched_body, pattern_rule_count, pattern_fetched_at
		FROM forward_rules
		WHERE pattern_source = 'url' AND disabled = false
		  AND pattern_url IS NOT NULL AND btrim(pattern_url) <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanForwardRuleRows(rows, true)
}

func (s *Store) CreateForwardRule(ctx context.Context, instanceID int32, r ForwardRule) (int32, error) {
	var id int32
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO forward_rules (instance_id, name_pattern, pattern_source, pattern_url, pattern_format, target_group_id, mode, priority, disabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		instanceID, r.NamePattern, r.PatternSource, r.PatternURL, r.PatternFormat, r.TargetGroupID, r.Mode, r.Priority, r.Disabled,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateForwardRule(ctx context.Context, id int32, r ForwardRule) error {
	ct, err := s.Pool.Exec(ctx, `
		UPDATE forward_rules SET name_pattern = $1, pattern_source = $2, pattern_url = $3, pattern_format = $4,
			target_group_id = $5, mode = $6, priority = $7, disabled = $8 WHERE id = $9`,
		r.NamePattern, r.PatternSource, r.PatternURL, r.PatternFormat, r.TargetGroupID, r.Mode, r.Priority, r.Disabled, id,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpdateForwardRulePatternFetch 仅在拉取成功时更新缓存正文与统计；resolvedEntries 非空时写入 JSONB（AutoProxy 条目）。
func (s *Store) UpdateForwardRulePatternFetch(ctx context.Context, id int32, body string, ruleCount int, fetchedAt time.Time, resolvedEntries []byte) error {
	var ent any
	if len(resolvedEntries) > 0 {
		ent = resolvedEntries
	} else {
		ent = nil
	}
	ct, err := s.Pool.Exec(ctx, `
		UPDATE forward_rules SET pattern_fetched_body = $2, pattern_rule_count = $3, pattern_fetched_at = $4, pattern_resolved_entries = $5
		WHERE id = $1`,
		id, body, ruleCount, fetchedAt, ent)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ClearForwardRulePatternCache 清空拉取缓存（如修改 URL 或切换为 inline 时）。
func (s *Store) ClearForwardRulePatternCache(ctx context.Context, id int32) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE forward_rules SET pattern_fetched_body = NULL, pattern_rule_count = 0, pattern_fetched_at = NULL, pattern_resolved_entries = NULL
		WHERE id = $1`, id)
	return err
}

// UpdateForwardRuleInlinePatternCompile 更新多行/内联规则编译结果（不写 pattern_fetched_body）。
func (s *Store) UpdateForwardRuleInlinePatternCompile(ctx context.Context, id int32, ruleCount int, fetchedAt time.Time, resolvedEntries []byte) error {
	var ent any
	if len(resolvedEntries) > 0 {
		ent = resolvedEntries
	} else {
		ent = nil
	}
	ct, err := s.Pool.Exec(ctx, `
		UPDATE forward_rules SET pattern_rule_count = $2, pattern_fetched_at = $3, pattern_resolved_entries = $4
		WHERE id = $1`,
		id, ruleCount, fetchedAt, ent)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpdateForwardRuleDisabled 仅更新规则的禁用状态。
func (s *Store) UpdateForwardRuleDisabled(ctx context.Context, id int32, disabled bool) error {
	ct, err := s.Pool.Exec(ctx, `UPDATE forward_rules SET disabled = $1 WHERE id = $2`, disabled, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// IncrementForwardRuleHitCount 将指定转发规则的命中次数 +1（DNS 匹配到该规则时调用）。
func (s *Store) IncrementForwardRuleHitCount(ctx context.Context, id int32) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE forward_rules SET hit_count = hit_count + 1 WHERE id = $1`, id)
	return err
}

func (s *Store) DeleteForwardRule(ctx context.Context, id int32) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM forward_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
