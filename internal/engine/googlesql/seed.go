package googlesql

import (
	"embed"

	"github.com/mbvlabs/narsilc/internal/core"
	"github.com/mbvlabs/narsilc/internal/core/seed"
)

//go:embed dialect
var dialectFS embed.FS

// Dialect returns the catalog option that seeds GoogleSQL's type system.
func Dialect() core.Option {
	return seed.Dialect(dialectFS, "dialect")
}
