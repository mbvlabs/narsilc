package ast

import "github.com/mbvlabs/narsilc/internal/sql/format"

type Float struct {
	Tag NodeTag[Float] `json:"tag"`

	Str string `json:"str"`
}

func (n *Float) Pos() int {
	return 0
}

func (n *Float) Format(buf *TrackedBuffer, d format.Dialect) {
	if n == nil {
		return
	}
	buf.WriteString(n.Str)
}
