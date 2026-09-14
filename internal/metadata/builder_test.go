package metadata

import (
	"reflect"
	"testing"
)

func TestParseBuilderAnnotations(t *testing.T) {
	filters, orders, err := ParseBuilderAnnotations([]string{
		" name: ListAuthors :many",
		" @filter name eq,like",
		" @filter created_at gte,lte",
		" @order  name, created_at, id",
	})
	if err != nil {
		t.Fatal(err)
	}
	wantFilters := []Filter{
		{Column: "name", Operators: []FilterOp{FilterEq, FilterLike}},
		{Column: "created_at", Operators: []FilterOp{FilterGte, FilterLte}},
	}
	wantOrders := []Order{
		{Column: "name"},
		{Column: "created_at"},
		{Column: "id"},
	}
	if !reflect.DeepEqual(filters, wantFilters) {
		t.Errorf("filters: got %#v want %#v", filters, wantFilters)
	}
	if !reflect.DeepEqual(orders, wantOrders) {
		t.Errorf("orders: got %#v want %#v", orders, wantOrders)
	}
}

func TestParseBuilderAnnotationsErrors(t *testing.T) {
	cases := [][]string{
		{"@filter name"},
		{"@filter name bogus"},
		{"@filter name eq", "@filter name like"},
		{"@order"},
	}
	for _, comments := range cases {
		if _, _, err := ParseBuilderAnnotations(comments); err == nil {
			t.Errorf("expected error for %q", comments)
		}
	}
}
