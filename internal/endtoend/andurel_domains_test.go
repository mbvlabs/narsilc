package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbvlabs/narsilc/internal/cmd"
)

func TestGenerateAndurelDomains(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	files := map[string]string{
		"andurel.toml": `schemaVersion = 1
version = "v0.1.0"
`,
		"migrations/schema.sql": `CREATE TABLE widgets (
  id   BIGSERIAL PRIMARY KEY,
  name text NOT NULL
);

CREATE TABLE invoices (
  id     BIGSERIAL PRIMARY KEY,
  amount bigint NOT NULL
);
`,
		"domains/widget.sql": `-- name: GetWidget :one
SELECT * FROM widgets
WHERE id = $1 LIMIT 1;
`,
		"domains/widget.go": `package domains

// This file sits next to the queries and must not be read as query input.
`,
		"domains/billing/invoice.sql": `-- name: ListInvoices :many
SELECT * FROM invoices
ORDER BY id;
`,
		// Generated output is written here. A stale query would fail analysis
		// if the directory were scanned as input.
		"domains/internal/queries/stale.sql": `-- name: Stale :one
SELECT * FROM missing_table
WHERE id = $1 LIMIT 1;
`,
	}
	for name, contents := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var stderr bytes.Buffer
	output, err := cmd.Generate(context.Background(), dir, "", &cmd.Options{
		Env:    cmd.Env{},
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("narsilc generate failed: %s", stderr.String())
	}

	for _, name := range []string{
		"domains/internal/queries/db.go",
		"domains/internal/queries/models.go",
		"domains/internal/queries/widget.sql.go",
		"domains/internal/queries/invoice.sql.go",
	} {
		path := filepath.Join(dir, name)
		if _, ok := output[path]; !ok {
			t.Errorf("missing generated file %s", name)
		}
	}

	widget, ok := output[filepath.Join(dir, "domains/internal/queries/widget.sql.go")]
	if !ok {
		t.Fatal("widget query output missing")
	}
	if !strings.Contains(widget, "package queries") {
		t.Errorf("widget output package = %q, want queries", widget)
	}
	if !strings.Contains(widget, "GetWidget") {
		t.Errorf("widget output does not contain GetWidget:\n%s", widget)
	}

	if _, ok := output[filepath.Join(dir, "domains/widget.go")]; ok {
		t.Error("domains/widget.go was treated as generated output")
	}
	if _, ok := output[filepath.Join(dir, "domains/internal/queries/stale.sql")]; ok {
		t.Error("domains/internal/queries was treated as query input")
	}
}
