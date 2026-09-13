package dolphin

import (
	"github.com/mbvlabs/narsilc/internal/sql/catalog"
)

// defaultSchema is MySQL's standard library, read from the dialect directory's
// functions.jsonl.
func defaultSchema(name string) *catalog.Schema {
	return &catalog.Schema{Name: name, Funcs: stdlib()}
}
