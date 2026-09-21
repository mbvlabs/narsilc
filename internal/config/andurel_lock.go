package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	golang "github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
	toml "github.com/pelletier/go-toml/v2"
)

const (
	andurelTomlName = "andurel.toml"
	andurelLockName = "andurel.lock"

	andurelQueriesDir  = "models/queries"
	andurelSchemaDir   = "migrations"
	andurelOutDir      = "models/internal/queries"
	andurelPackageName = "queries"
)

// andurelToml is the subset of andurel.toml that narsilc reads.
type andurelToml struct {
	SchemaVersion int                    `toml:"schemaVersion"`
	Version       string                 `toml:"version"`
	Database      *andurelDatabaseConfig `toml:"database"`
}

type andurelDatabaseConfig struct {
	Engine   string `toml:"engine" json:"engine"`
	NullType string `toml:"nullType" json:"nullType"`
}

// FromAndurelToml builds a narsilc Config from an andurel.toml document.
// Paths, package, sql_package (pgx/v5), Andurel row mapping, and stdlib UUID
// overrides are fixed. The user choice is database.nullType
// (pgtype.Null, pointer, or sql.Null).
func FromAndurelToml(data []byte) (Config, error) {
	var doc andurelToml
	if err := toml.Unmarshal(data, &doc); err != nil {
		return Config{}, fmt.Errorf("andurel.toml: %w", err)
	}
	if doc.SchemaVersion == 0 {
		return Config{}, fmt.Errorf("andurel.toml schemaVersion is required")
	}
	if doc.SchemaVersion != 1 {
		return Config{}, fmt.Errorf("andurel.toml schemaVersion %d is newer than this narsilc supports; upgrade narsilc to read it", doc.SchemaVersion)
	}
	if doc.Version == "" {
		return Config{}, fmt.Errorf("andurel.toml version is required")
	}

	engine := EnginePostgreSQL
	nullType := "pgtype.Null"
	if doc.Database != nil {
		if doc.Database.Engine != "" {
			switch doc.Database.Engine {
			case "postgresql", "postgres":
				engine = EnginePostgreSQL
			default:
				return Config{}, fmt.Errorf("andurel.toml database.engine %q is not supported", doc.Database.Engine)
			}
		}
		if doc.Database.NullType != "" {
			nullType = doc.Database.NullType
		}
	}

	return andurelConfigFromEngineNull(engine, nullType, "andurel.toml")
}

func andurelConfigFromEngineNull(engine Engine, nullType, source string) (Config, error) {
	var emitPointers bool
	switch nullType {
	case "pointer":
		emitPointers = true
	case "pgtype.Null", "sql.Null":
		emitPointers = false
	default:
		return Config{}, fmt.Errorf("%s database.nullType %q is not supported", source, nullType)
	}

	return Config{
		Version: "2",
		SQL: []SQL{{
			Engine:  engine,
			Schema:  Paths{andurelSchemaDir},
			Queries: Paths{andurelQueriesDir},
			Gen: SQLGen{
				Go: &golang.Options{
					Package:                  andurelPackageName,
					Out:                      andurelOutDir,
					SqlPackage:               golang.SQLPackagePGXV5,
					RowMapping:               golang.RowMappingAndurel,
					OmitSqlcVersion:          true,
					OmitUnusedStructs:        true,
					EmitPointersForNullTypes: emitPointers,
					Overrides: []golang.Override{
						{
							DBType: "uuid",
							GoType: golang.GoType{Spec: "uuid.UUID"},
						},
						{
							DBType:   "uuid",
							Nullable: true,
							GoType: golang.GoType{
								Path:    "uuid",
								Name:    "UUID",
								Pointer: true,
							},
						},
					},
				},
			},
		}},
	}, nil
}

// IsAndurelToml reports whether name is the Andurel project manifest.
func IsAndurelToml(name string) bool {
	return name == andurelTomlName
}

// IsAndurelLock reports whether name is the Andurel digest lock file.
// Kept for path discovery compatibility; digests are not read by narsilc.
func IsAndurelLock(name string) bool {
	return name == andurelLockName
}

// ReadAndurelProjectConfig loads Andurel generation settings from dir.
// Prefers andurel.toml; rejects legacy JSON andurel.lock documents.
func ReadAndurelProjectConfig(dir string) (Config, error) {
	tomlPath := filepath.Join(dir, andurelTomlName)
	data, err := os.ReadFile(tomlPath)
	if err == nil {
		return FromAndurelToml(data)
	}
	if !os.IsNotExist(err) {
		return Config{}, err
	}

	lockPath := filepath.Join(dir, andurelLockName)
	lockData, lockErr := os.ReadFile(lockPath)
	if lockErr == nil {
		trimmed := bytes.TrimSpace(lockData)
		if len(trimmed) > 0 && trimmed[0] == '{' {
			return Config{}, fmt.Errorf("V2 projects require %s; found legacy JSON %s", andurelTomlName, andurelLockName)
		}
	}
	return Config{}, fmt.Errorf("%s not found", andurelTomlName)
}
