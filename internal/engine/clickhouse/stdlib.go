package clickhouse

import (
	"github.com/mbvlabs/narsilc/internal/sql/catalog"
)

func defaultSchema(name string) *catalog.Schema {
	return &catalog.Schema{Name: name}
}
