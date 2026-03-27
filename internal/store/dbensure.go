package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isDatabaseDoesNotExist(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "3D000"
}

func isDuplicateDatabase(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "42P04" {
			return true
		}
		if strings.Contains(strings.ToLower(pgErr.Message), "already exists") {
			return true
		}
	}
	return false
}

func databaseNameFromURL(databaseURL string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}
	if u.Path == "" || u.Path == "/" {
		return "", fmt.Errorf("no database name in DATABASE_URL path")
	}
	return strings.TrimPrefix(u.Path, "/"), nil
}

func replaceDatabaseInURL(databaseURL, newDBName string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database url: %w", err)
	}
	switch u.Scheme {
	case "postgres", "postgresql":
	default:
		return "", fmt.Errorf("unsupported DATABASE_URL scheme %q (need postgres or postgresql)", u.Scheme)
	}
	u.Path = "/" + newDBName
	return u.String(), nil
}

func createDatabaseIfMissing(ctx context.Context, databaseURL string) error {
	dbName, err := databaseNameFromURL(databaseURL)
	if err != nil {
		return err
	}
	switch dbName {
	case "postgres", "template0", "template1":
		return fmt.Errorf("cannot auto-create reserved database name %q", dbName)
	}

	adminURL, err := replaceDatabaseInURL(databaseURL, "postgres")
	if err != nil {
		return err
	}
	adminCfg, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		return fmt.Errorf("parse admin database url: %w", err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminCfg)
	if err != nil {
		return fmt.Errorf("connect to postgres database for CREATE DATABASE: %w", err)
	}
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres database: %w", err)
	}

	q := fmt.Sprintf(`CREATE DATABASE %s`, pgx.Identifier{dbName}.Sanitize())
	_, err = adminPool.Exec(ctx, q)
	if err != nil {
		if isDuplicateDatabase(err) {
			return nil
		}
		return fmt.Errorf("create database %q: %w", dbName, err)
	}
	slog.Info("auto-created database", "name", dbName)
	return nil
}

// ensureTargetDatabaseExists 若目标库不存在（3D000），则连到 postgres 执行 CREATE DATABASE；失败时返回原错误。
func ensureTargetDatabaseExists(ctx context.Context, databaseURL string) error {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		if isDatabaseDoesNotExist(err) {
			return createDatabaseIfMissing(ctx, databaseURL)
		}
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		if isDatabaseDoesNotExist(err) {
			return createDatabaseIfMissing(ctx, databaseURL)
		}
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
