package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ListUpstreamGroups(ctx context.Context) ([]UpstreamGroup, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, created_at FROM upstream_groups ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UpstreamGroup
	for rows.Next() {
		var g UpstreamGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []UpstreamGroup{}, nil
	}
	return out, nil
}

func (s *Store) CreateUpstreamGroup(ctx context.Context, name string) (int32, error) {
	var id int32
	err := s.Pool.QueryRow(ctx, `INSERT INTO upstream_groups (name) VALUES ($1) RETURNING id`, name).Scan(&id)
	return id, err
}

func (s *Store) UpdateUpstreamGroup(ctx context.Context, id int32, name string) error {
	ct, err := s.Pool.Exec(ctx, `UPDATE upstream_groups SET name = $1 WHERE id = $2`, name, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteUpstreamGroup(ctx context.Context, id int32) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM upstream_groups WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s *Store) ListUpstreamServers(ctx context.Context, groupID int32) ([]UpstreamServer, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, group_id, address, port, sort_order, protocol, path, tls_server_name, created_at
		FROM upstream_servers WHERE group_id = $1 ORDER BY sort_order, id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UpstreamServer
	for rows.Next() {
		var u UpstreamServer
		if err := rows.Scan(&u.ID, &u.GroupID, &u.Address, &u.Port, &u.SortOrder, &u.Protocol, &u.Path, &u.TLSServerName, &u.CreatedAt); err != nil {
			return nil, err
		}
		if u.Protocol == "" {
			u.Protocol = "udp"
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		return []UpstreamServer{}, nil
	}
	return out, nil
}

// AddUpstreamServerInput 创建上游服务器的入参；新协议字段集中在此，便于 API 与存储解耦。
type AddUpstreamServerInput struct {
	GroupID       int32
	Address       string
	Port          int
	SortOrder     int
	Protocol      string  // udp / dot / doh；空值视为 udp
	Path          *string // 仅 DoH 使用
	TLSServerName *string // DoT/DoH 可选
}

func (s *Store) AddUpstreamServer(ctx context.Context, in AddUpstreamServerInput) (int32, error) {
	proto := in.Protocol
	if proto == "" {
		proto = "udp"
	}
	var id int32
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO upstream_servers (group_id, address, port, sort_order, protocol, path, tls_server_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		in.GroupID, in.Address, in.Port, in.SortOrder, proto, in.Path, in.TLSServerName,
	).Scan(&id)
	return id, err
}

func (s *Store) DeleteUpstreamServer(ctx context.Context, id int32) error {
	ct, err := s.Pool.Exec(ctx, `DELETE FROM upstream_servers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
