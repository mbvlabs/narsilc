package config

import (
	"testing"

	golang "github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
)

func TestFromAndurelLock(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		conf, err := FromAndurelLock([]byte(`{
			"schemaVersion": 1,
			"version": "v0.1.0"
		}`))
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
		if opts.Overrides[0].DBType != "uuid" || opts.Overrides[0].Nullable || opts.Overrides[0].GoType.Spec != "uuid.UUID" {
			t.Fatalf("non-null uuid override = %+v", opts.Overrides[0])
		}
		nullUUID := opts.Overrides[1]
		if nullUUID.DBType != "uuid" || !nullUUID.Nullable || nullUUID.GoType.Path != "uuid" || nullUUID.GoType.Name != "UUID" || !nullUUID.GoType.Pointer {
			t.Fatalf("nullable uuid override = %+v", nullUUID)
		}
	})

	t.Run("pointer nullType", func(t *testing.T) {
		t.Parallel()
		conf, err := FromAndurelLock([]byte(`{
			"schemaVersion": 1,
			"version": "v0.1.0",
			"databaseConfig": {
				"engine": "postgresql",
				"nullType": "pointer"
			}
		}`))
		if err != nil {
			t.Fatal(err)
		}
		if !conf.SQL[0].Gen.Go.EmitPointersForNullTypes {
			t.Fatal("pointer nullType should emit pointers")
		}
	})

	t.Run("rejects tools-only requirements gone", func(t *testing.T) {
		t.Parallel()
		if _, err := FromAndurelLock([]byte(`{
			"schemaVersion": 1,
			"version": "v0.1.0"
		}`)); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("rejects bun.Null", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelLock([]byte(`{
			"schemaVersion": 1,
			"version": "v0.1.0",
			"databaseConfig": {"nullType": "bun.Null"}
		}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("sql.Null nullType", func(t *testing.T) {
		t.Parallel()
		conf, err := FromAndurelLock([]byte(`{
			"schemaVersion": 1,
			"version": "v0.1.0",
			"databaseConfig": {"nullType": "sql.Null"}
		}`))
		if err != nil {
			t.Fatal(err)
		}
		if conf.SQL[0].Gen.Go.EmitPointersForNullTypes {
			t.Fatal("sql.Null nullType should not emit pointers")
		}
	})

	t.Run("rejects unknown engine", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelLock([]byte(`{
			"schemaVersion": 1,
			"version": "v0.1.0",
			"databaseConfig": {"engine": "mysql"}
		}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects schemaVersion 2", func(t *testing.T) {
		t.Parallel()
		_, err := FromAndurelLock([]byte(`{
			"schemaVersion": 2,
			"version": "v0.1.0"
		}`))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
