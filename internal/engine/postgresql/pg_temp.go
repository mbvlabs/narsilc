package postgresql

import (
	"github.com/mbvlabs/narsilc/internal/sql/catalog"
)

func pgTemp() *catalog.Schema {
	return &catalog.Schema{Name: "pg_temp"}
}
