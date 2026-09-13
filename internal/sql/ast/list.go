package ast

import "github.com/mbvlabs/narsilc/internal/sql/format"

type List struct {
	Tag NodeTag[List] `json:"tag"`

	Items []Node `json:"items,omitempty"`
}

func (n *List) Pos() int {
	return 0
}

func (n *List) Format(buf *TrackedBuffer, d format.Dialect) {
	if n == nil {
		return
	}
	buf.join(n, d, ", ")
}
