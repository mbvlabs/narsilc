package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/mbvlabs/narsilc/internal/sqltest/local"
)

func TestJoinOrderByQualifiedColumnExecutes(t *testing.T) {
	ctx := context.Background()
	uri := local.PostgreSQL(t, []string{"testdata/query_builder_join/postgresql/pgx/v5/schema.sql"})
	db, err := pgx.Connect(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)

	_, err = db.Exec(ctx, `
SELECT u.id, u.email, t.id, t.scope
FROM users u
JOIN tokens t ON t.user_id = u.id
ORDER BY u.id
`)
	if err != nil {
		t.Fatalf("ORDER BY u.id: %v", err)
	}

	_, err = db.Exec(ctx, `
WITH u AS (
  SELECT id, email FROM users
)
SELECT id, email FROM u
ORDER BY u.id
`)
	if err != nil {
		t.Fatalf("CTE ORDER BY u.id: %v", err)
	}

	_, err = db.Exec(ctx, `
SELECT id, email FROM users
ORDER BY public.users.id
`)
	if err != nil {
		t.Fatalf("ORDER BY public.users.id: %v", err)
	}
}
