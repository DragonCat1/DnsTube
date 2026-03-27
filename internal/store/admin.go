package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CountAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_user`).Scan(&n)
	return n, err
}

func (s *Store) CreateAdmin(ctx context.Context, username, passwordHash string) (int32, error) {
	var id int32
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO admin_user (username, password_hash) VALUES ($1, $2) RETURNING id`,
		username, passwordHash,
	).Scan(&id)
	return id, err
}

func (s *Store) GetAdminByUsername(ctx context.Context, username string) (*AdminUser, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM admin_user WHERE username = $1`,
		username,
	)
	var u AdminUser
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetAdminByID(ctx context.Context, id int32) (*AdminUser, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM admin_user WHERE id = $1`,
		id,
	)
	var u AdminUser
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateAdminPassword(ctx context.Context, id int32, hash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE admin_user SET password_hash = $1, updated_at = now() WHERE id = $2`, hash, id)
	return err
}

func (s *Store) UpdateAdminUsername(ctx context.Context, id int32, username string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE admin_user SET username = $1, updated_at = now() WHERE id = $2`, username, id)
	return err
}

func (s *Store) GetFirstAdmin(ctx context.Context) (*AdminUser, error) {
	row := s.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at, updated_at FROM admin_user ORDER BY id LIMIT 1`,
	)
	var u AdminUser
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}
