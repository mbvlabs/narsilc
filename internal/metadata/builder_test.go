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

func TestParseColumnRef(t *testing.T) {
	cases := []struct {
		in   string
		want ColumnRef
	}{
		{"id", ColumnRef{Name: "id"}},
		{"u.id", ColumnRef{Rel: "u", Name: "id"}},
		{"public.users.id", ColumnRef{Schema: "public", Rel: "users", Name: "id"}},
	}
	for _, tc := range cases {
		got, err := ParseColumnRef(tc.in)
		if err != nil {
			t.Fatalf("%s: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("%s: got %+v want %+v", tc.in, got, tc.want)
		}
	}
	for _, in := range []string{"", "u.", ".id", "a.b.c.d"} {
		if _, err := ParseColumnRef(in); err == nil {
			t.Errorf("expected error for %q", in)
		}
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
