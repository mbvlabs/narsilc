package narsilc

import (
	"fmt"
	"reflect"
)

// Transformer maps a generated driver-native row into T.
// Implement on *T, not on the row type.
type Transformer[Row any] interface {
	Transform(Row) error
}

func typeName[T any]() string {
	var z T
	return reflect.TypeOf(&z).Elem().String()
}

// FromRow returns row when T is Row, calls Transform when *T implements
// Transformer[Row], and otherwise errors with both type names.
func FromRow[T, Row any](row Row) (T, error) {
	var dest T
	if t, ok := any(row).(T); ok {
		return t, nil
	}
	if tr, ok := any(&dest).(Transformer[Row]); ok {
		if err := tr.Transform(row); err != nil {
			return dest, err
		}
		return dest, nil
	}
	return dest, fmt.Errorf(
		"%s does not implement narsilc.Transformer[%s]",
		typeName[T](),
		typeName[Row](),
	)
}

// Read scans once. If *T implements Transformer[Row], it scans into Row and
// calls FromRow; otherwise it scans into T (today's 1:1 path).
func Read[T, Row any](s Scanner, columns []string) (T, error) {
	var dest T
	if _, ok := any(&dest).(Transformer[Row]); ok {
		row, err := Scan[Row](s, columns)
		if err != nil {
			return dest, err
		}
		return FromRow[T, Row](row)
	}
	return Scan[T](s, columns)
}
