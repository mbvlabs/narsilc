package narsilc

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type Scanner interface {
	Scan(dest ...any) error
}

type layout struct {
	indexes [][]int
}

var layoutCache sync.Map // map[layoutKey]layout

type layoutKey struct {
	typ     reflect.Type
	columns string
}

func Scan[T any](s Scanner, columns []string) (T, error) {
	var dest T
	ptrs, err := destPtrs(&dest, columns)
	if err != nil {
		return dest, err
	}
	if err := s.Scan(ptrs...); err != nil {
		return dest, err
	}
	return dest, nil
}

func destPtrs[T any](dest *T, columns []string) ([]any, error) {
	v := reflect.ValueOf(dest).Elem()
	l, err := layoutFor(v.Type(), columns)
	if err != nil {
		return nil, err
	}
	ptrs := make([]any, len(l.indexes))
	for i, idx := range l.indexes {
		ptrs[i] = v.FieldByIndex(idx).Addr().Interface()
	}
	return ptrs, nil
}

func layoutFor(t reflect.Type, columns []string) (layout, error) {
	key := layoutKey{typ: t, columns: strings.Join(columns, "\x00")}
	if cached, ok := layoutCache.Load(key); ok {
		return cached.(layout), nil
	}

	l, err := buildLayout(t, columns)
	if err != nil {
		return layout{}, err
	}
	layoutCache.Store(key, l)
	return l, nil
}

func buildLayout(t reflect.Type, columns []string) (layout, error) {
	if t.Kind() != reflect.Struct {
		return layout{}, fmt.Errorf("%s is not a struct", t)
	}

	byCol := make(map[string][]int, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("andurel")
		if tag == "" || tag == "-" {
			continue
		}
		if _, exists := byCol[tag]; exists {
			return layout{}, fmt.Errorf("duplicate tag %q on %s", tag, t)
		}
		byCol[tag] = f.Index
	}

	indexes := make([][]int, len(columns))
	for i, col := range columns {
		idx, ok := byCol[col]
		if !ok {
			return layout{}, fmt.Errorf("no field with andurel:%q on %s", col, t)
		}
		indexes[i] = idx
	}
	return layout{indexes: indexes}, nil
}
