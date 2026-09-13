package config

import (
	"encoding/json"
	"fmt"

	golang "github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
)

const (
	andurelLockName = "andurel.lock"

	andurelQueriesDir  = "models/queries"
	andurelSchemaDir   = "migrations"
	andurelOutDir      = "models/internal/queries"
	andurelPackageName = "queries"
)

// andurelLock is the subset of andurel.lock that narsilc reads.
type andurelLock struct {
	SchemaVersion  int                        `json:"schemaVersion"`
	Version        string                     `json:"version"`
	Tools          map[string]json.RawMessage `json:"tools"`
	ScaffoldConfig *andurelScaffoldConfig     `json:"scaffoldConfig"`
	DatabaseConfig *andurelDatabaseConfig     `json:"databaseConfig"`
}

type andurelScaffoldConfig struct {
	Database string `json:"database"`
}

type andurelDatabaseConfig struct {
	NullType string `json:"nullType"`
}

type andurelSchemaHeader struct {
	SchemaVersion int `json:"schemaVersion"`
}

// FromAndurelLock builds a narsilc Config from an andurel.lock document.
// Paths, engine, package, and Andurel row mapping are fixed. The only
// user choice is databaseConfig.nullType.
func FromAndurelLock(data []byte) (Config, error) {
	var header andurelSchemaHeader
	if err := json.Unmarshal(data, &header); err != nil {
		return Config{}, fmt.Errorf("andurel.lock: %w", err)
	}
	if header.SchemaVersion == 0 {
		return Config{}, fmt.Errorf("andurel.lock schemaVersion is required")
	}
	if header.SchemaVersion != 1 {
		return Config{}, fmt.Errorf("andurel.lock schemaVersion %d is newer than this narsilc supports; upgrade narsilc to read it", header.SchemaVersion)
	}

	var lock andurelLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return Config{}, fmt.Errorf("andurel.lock: %w", err)
	}
	if lock.Version == "" {
		return Config{}, fmt.Errorf("andurel.lock version is required")
	}
	if lock.Tools == nil {
		return Config{}, fmt.Errorf("andurel.lock tools is required")
	}

	engine := EnginePostgreSQL
	if lock.ScaffoldConfig != nil && lock.ScaffoldConfig.Database != "" {
		switch lock.ScaffoldConfig.Database {
		case "postgresql", "postgres":
			engine = EnginePostgreSQL
		default:
			return Config{}, fmt.Errorf("andurel.lock scaffoldConfig.database %q is not supported", lock.ScaffoldConfig.Database)
		}
	}

	nullType := "sql.Null"
	if lock.DatabaseConfig != nil && lock.DatabaseConfig.NullType != "" {
		nullType = lock.DatabaseConfig.NullType
	}
	var emitPointers bool
	switch nullType {
	case "pointer":
		emitPointers = true
	case "sql.Null", "bun.Null":
		emitPointers = false
	default:
		return Config{}, fmt.Errorf("andurel.lock databaseConfig.nullType %q is not supported", nullType)
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
					RowMapping:               golang.RowMappingAndurel,
					OmitSqlcVersion:          true,
					OmitUnusedStructs:        true,
					EmitPointersForNullTypes: emitPointers,
				},
			},
		}},
	}, nil
}

// IsAndurelLock reports whether name is the Andurel project lock file.
func IsAndurelLock(name string) bool {
	return name == andurelLockName
}
