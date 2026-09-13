package sqlite

import "github.com/mbvlabs/narsilc/internal/sql/catalog"

func NewCatalog() *catalog.Catalog {
	def := "main"
	return &catalog.Catalog{
		DefaultSchema: def,
		Schemas: []*catalog.Schema{
			defaultSchema(def),
		},
		LoadExtension: loadExtension,
		Extensions:    map[string]struct{}{},
	}
}
