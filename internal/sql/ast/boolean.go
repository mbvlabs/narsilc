package ast

import (
	"fmt"

	"github.com/mbvlabs/narsilc/internal/sql/format"
)

type Boolean struct {
	Tag NodeTag[Boolean] `json:"tag"`

	Boolval bool `json:"boolval"`
}

func (n *Boolean) Pos() int {
	return 0
}

func (n *Boolean) Format(buf *TrackedBuffer, d format.Dialect) {
	if n == nil {
		return
	}
	if n.Boolval {
		fmt.Fprintf(buf, "true")
	} else {
		fmt.Fprintf(buf, "false")
	}
}
