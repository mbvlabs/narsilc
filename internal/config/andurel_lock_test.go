package config

import (
	"testing"

	golang "github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
)

func TestFromAndurelToml(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		conf, err := FromAndurelToml([]byte(`
schemaVersion = 1
version = "v0.1.0"
`))
		if err != nil {
			t.Fatal(err)
		}
		if conf.SQL[0].Engine != EnginePostgreSQL {
			t.Fatalf("engine = %q, want %q", conf.SQL[0].Engine, EnginePostgreSQL)
		}
		opts := conf.SQL[0].Gen.Go
		if opts.SqlPackage != golang.SQLPackagePGXV5 {
			t.Fatalf("sql_package = %q, want %q", opts.SqlPackage, golang.SQLPackagePGXV5)
		}
		if opts.RowMapping != golang.RowMappingAndurel {
			t.Fatalf("row_mapping = %q, want %q", opts.RowMapping, golang.RowMappingAndurel)
		}
		if opts.EmitPointersForNullTypes {
			t.Fatal("default nullType should not emit pointers")
		}
		if len(opts.Overrides) != 2 {
			t.Fatalf("overrides = %d, want 2", len(opts.Overrides))
		}
		if opts.Package != "queries" {
			t.Fatalf("package = %q, want %q", opts.Package, "queries")
		}
		if got := conf.SQL[0].Schema; len(got) != 1 || got[0] != "migrations" {
			t.Fatalf("schema = %v, want [migrations]", got)
		}
		if got := conf.SQL[0].Queries; len(got) != 1 || got[0] != "domains" {
			t.Fatalf("queries = %v, want [domains]", got)
		}
		if !conf.SQL[0].QueriesRecursive {
			t.Fatal("andurel queries should be discovered recursively")
		}
		if got := conf.SQL[0].QueriesExclude; len(got) != 1 || got[0] != "internal/queries" {
			t.Fatalf("query excludes = %v, want [internal/queries]", got)
		}
		if opts.Out != "domains/internal/queries" {
			t.Fatalf("out = %q, want %q", opts.Out, "domains/internal/queries")
		}
	})

	t.Run("pointer nullType", func(t *testing.T) {
		t.Parallel()
		conf, err := FromAndurelToml([]byte(`
schemaVersion = 1
version = "v0.1.0"

[database]
engine = "postgresql"
nullType = "pointer"
`))
		if err != nil {
			t.Fatal(err)
		}
		if !conf.SQL[0].Gen.Go.EmitPointersForNullTypes {
			t.Fatal("pointer nullType should emit pointers")
		}
	})

	t.Run("rejects bun.Null", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelToml([]byte(`
schemaVersion = 1
version = "v0.1.0"

[database]
nullType = "bun.Null"
`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects sql.Null", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelToml([]byte(`
schemaVersion = 1
version = "v0.1.0"

[database]
nullType = "sql.Null"
`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ignores project and tools tables", func(t *testing.T) {
		t.Parallel()
		conf, err := FromAndurelToml([]byte(`
schemaVersion = 1
version = "v0.1.0"

[project]
name = "testapp"

[database]
engine = "postgresql"
nullType = "pgtype.Null"

[tools]
narsilc = "v0.1.0"
`))
		if err != nil {
			t.Fatal(err)
		}
		if conf.SQL[0].Engine != EnginePostgreSQL {
			t.Fatalf("engine = %q, want %q", conf.SQL[0].Engine, EnginePostgreSQL)
		}
		if conf.SQL[0].Gen.Go.EmitPointersForNullTypes {
			t.Fatal("pgtype.Null should not emit pointers")
		}
	})

	t.Run("rejects unknown engine", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelToml([]byte(`
schemaVersion = 1
version = "v0.1.0"

[database]
engine = "mysql"
`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects schemaVersion 2", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelToml([]byte(`
schemaVersion = 2
version = "v0.1.0"
`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
