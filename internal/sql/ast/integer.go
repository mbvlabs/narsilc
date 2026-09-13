package ast

import (
	"strconv"

	"github.com/mbvlabs/narsilc/internal/sql/format"
)

type Integer struct {
	Tag NodeTag[Integer] `json:"tag"`

	Ival int64 `json:"ival"`
}

func (n *Integer) Pos() int {
	return 0
}

func (n *Integer) Format(buf *TrackedBuffer, d format.Dialect) {
	if n == nil {
		return
	}
	buf.WriteString(strconv.FormatInt(n.Ival, 10))
}
