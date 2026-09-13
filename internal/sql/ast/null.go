package ast

import "github.com/mbvlabs/narsilc/internal/sql/format"

type Null struct {
	Tag NodeTag[Null] `json:"tag"`
}

func (n *Null) Pos() int {
	return 0
}
func (n *Null) Format(buf *TrackedBuffer, d format.Dialect) {
	buf.WriteString("NULL")
}
