package googlesql

import (
	"github.com/mbvlabs/narsilc/internal/sql/catalog"
)

func NewCatalog() *catalog.Catalog {
	def := "main" // GoogleSQL has no implicit schema; use "main" as the default
	return &catalog.Catalog{
		DefaultSchema: def,
		Schemas: []*catalog.Schema{
			defaultSchema(def),
		},
		Extensions: map[string]struct{}{},
	}
}
