package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CountRecordsByInstance 返回该实例下静态记录条数。
func (s *Store) CountRecordsByInstance(ctx context.Context, instanceID int32) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM dns_records WHERE instance_id = $1`, instanceID).Scan(&n)
	return n, err
}

// ListRecordCountsPerInstance 返回各实例 id → 记录条数（无记录的实例不在 map 中）。
func (s *Store) ListRecordCountsPerInstance(ctx context.Context) (map[int32]int64, error) {
	rows, err := s.Pool.Query(ctx, `SELECT instance_id, COUNT(*)::bigint FROM dns_records GROUP BY instance_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int32]int64)
	for rows.Next() {
		var iid int32
		var n int64
		if err := rows.Scan(&iid, &n); err != nil {
			return nil, err
		}
		out[iid] = n
	}
	return out, rows.Err()
}

func (s *Store) ListRecordsByInstance(ctx context.Context, instanceID int32) ([]DNSRecord, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, instance_id, name, rtype, ttl, content, created_at, updated_at
		FROM dns_records WHERE instance_id = $1 ORDER BY name, rtype, id`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DNSRecord
	for rows.Next() {
		var r DNSRecord
		if err := rows.Scan(&r.ID, &r.InstanceID, &r.Name, &r.Rtype, &r.TTL, &r.Content, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateRecord(ctx context.Context, instanceID int32, name, rtype string, ttl int, content string) (int32, error) {
	var id int32
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO dns_records (instance_id, name, rtype, ttl, content)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		instanceID, name, rtype, ttl, content,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateRecord(ctx context.Context, id int32, name, rtype string, ttl int, content string) error {
	ct, err := s.Pool.Exec(ctx, `
		UPDATE dns_records SET name = $1, rtype = $2, ttl = $3, content = $4, updated_at = now() WHERE id = $5`,
		name, rtype, ttl, content, id,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteRecord(ctx context.Context, id int32) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM dns_records WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
