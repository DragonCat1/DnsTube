package api

import (
	"errors"

	"github.com/dnstube/dnstube/internal/store"
	"github.com/jackc/pgx/v5/pgconn"
)

func mapStoreErr(err error) string {
	if err == nil {
		return CodeDatabaseError
	}
	if errors.Is(err, store.ErrNotFound) {
		return CodeNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return CodeDuplicateKey
	}
	return CodeDatabaseError
}
