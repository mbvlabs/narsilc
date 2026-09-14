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
	SchemaVersion  int                    `json:"schemaVersion"`
	Version        string                 `json:"version"`
	DatabaseConfig *andurelDatabaseConfig `json:"databaseConfig"`
}

type andurelDatabaseConfig struct {
	Engine   string `json:"engine"`
	NullType string `json:"nullType"`
}

type andurelSchemaHeader struct {
	SchemaVersion int `json:"schemaVersion"`
}

// FromAndurelLock builds a narsilc Config from an andurel.lock document.
// Paths, package, sql_package (pgx/v5), and Andurel row mapping are fixed.
// User choices are databaseConfig.engine and databaseConfig.nullType.
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

	engine := EnginePostgreSQL
	nullType := "pgtype.Null"
	if lock.DatabaseConfig != nil {
		if lock.DatabaseConfig.Engine != "" {
			switch lock.DatabaseConfig.Engine {
			case "postgresql", "postgres":
				engine = EnginePostgreSQL
			default:
				return Config{}, fmt.Errorf("andurel.lock databaseConfig.engine %q is not supported", lock.DatabaseConfig.Engine)
			}
		}
		if lock.DatabaseConfig.NullType != "" {
			nullType = lock.DatabaseConfig.NullType
		}
	}

	var emitPointers bool
	switch nullType {
	case "pointer":
		emitPointers = true
	case "pgtype.Null":
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
					SqlPackage:               golang.SQLPackagePGXV5,
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
