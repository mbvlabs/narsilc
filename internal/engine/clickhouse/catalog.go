package clickhouse

import (
	"github.com/mbvlabs/narsilc/internal/sql/catalog"
)

func NewCatalog() *catalog.Catalog {
	def := "default" // ClickHouse default database
	return &catalog.Catalog{
		DefaultSchema: def,
		Schemas: []*catalog.Schema{
			defaultSchema(def),
		},
		Extensions: map[string]struct{}{},
	}
}
