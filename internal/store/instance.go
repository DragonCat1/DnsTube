package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CountInstancesOverview 返回实例总数与暂停中的数量。
func (s *Store) CountInstancesOverview(ctx context.Context) (total int32, paused int32, err error) {
	err = s.Pool.QueryRow(ctx, `
		SELECT COUNT(*)::int, COUNT(*) FILTER (WHERE paused)::int FROM dns_instances`,
	).Scan(&total, &paused)
	return total, paused, err
}

func (s *Store) ListInstances(ctx context.Context) ([]DNSInstance, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, listen_addr, listen_port, paused, default_upstream_group_id, created_at, updated_at
		FROM dns_instances ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DNSInstance
	for rows.Next() {
		var d DNSInstance
		if err := rows.Scan(&d.ID, &d.Name, &d.ListenAddr, &d.ListenPort, &d.Paused, &d.DefaultUpstreamGroupID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetInstance(ctx context.Context, id int32) (*DNSInstance, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id, name, listen_addr, listen_port, paused, default_upstream_group_id, created_at, updated_at
		FROM dns_instances WHERE id = $1`, id)
	var d DNSInstance
	if err := row.Scan(&d.ID, &d.Name, &d.ListenAddr, &d.ListenPort, &d.Paused, &d.DefaultUpstreamGroupID, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Store) CreateInstance(ctx context.Context, name, listenAddr string, listenPort int, defaultGroup *int32) (int32, error) {
	var id int32
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO dns_instances (name, listen_addr, listen_port, default_upstream_group_id)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		name, listenAddr, listenPort, defaultGroup,
	).Scan(&id)
	return id, err
}

func (s *Store) UpdateInstance(ctx context.Context, id int32, name *string, listenAddr *string, listenPort *int, paused *bool, defaultGroup *int32) error {
	// simple: fetch-merge or use COALESCE - use dynamic update
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	cur, err := s.GetInstance(ctx, id)
	if err != nil {
		return err
	}
	n := cur.Name
	la := cur.ListenAddr
	lp := cur.ListenPort
	p := cur.Paused
	dg := cur.DefaultUpstreamGroupID
	if name != nil {
		n = *name
	}
	if listenAddr != nil {
		la = *listenAddr
	}
	if listenPort != nil {
		lp = *listenPort
	}
	if paused != nil {
		p = *paused
	}
	if defaultGroup != nil {
		dg = defaultGroup
	}
	_, err = tx.Exec(ctx, `
		UPDATE dns_instances SET name = $1, listen_addr = $2, listen_port = $3, paused = $4, default_upstream_group_id = $5, updated_at = now()
		WHERE id = $6`,
		n, la, lp, p, dg, id,
	)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeleteInstance(ctx context.Context, id int32) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM dns_instances WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
