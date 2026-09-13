package compiler

import (
	"github.com/mbvlabs/narsilc/internal/sql/catalog"
)

type Result struct {
	Catalog *catalog.Catalog
	Queries []*Query
}
