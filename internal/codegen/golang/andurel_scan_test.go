package golang

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
)

type assignScanner []any

func (a assignScanner) Scan(dest ...any) error {
	if len(dest) != len(a) {
		return errors.New("dest count mismatch")
	}
	for i, v := range a {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

func TestScanAndurel(t *testing.T) {
	type author struct {
		ID   int64          `andurel:"id"`
		Name string         `andurel:"name"`
		Bio  sql.NullString `andurel:"bio"`
		Skip string
	}

	got, err := scanAndurel[author](assignScanner{int64(7), "ada", sql.NullString{String: "bio", Valid: true}}, []string{"id", "name", "bio"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "ada" || got.Bio.String != "bio" || got.Skip != "" {
		t.Fatalf("unexpected scan: %+v", got)
	}
}

func TestScanAndurelProjection(t *testing.T) {
	type author struct {
		ID        int64  `andurel:"id"`
		Name      string `andurel:"name"`
		PostCount int64  `andurel:"post_count"`
	}

	got, err := scanAndurel[author](assignScanner{int64(1), "ada"}, []string{"id", "name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || got.Name != "ada" || got.PostCount != 0 {
		t.Fatalf("unexpected projection: %+v", got)
	}
}

func TestScanAndurelMissingTag(t *testing.T) {
	type author struct {
		ID int64 `andurel:"id"`
	}
	_, err := scanAndurel[author](assignScanner{int64(1), "ada"}, []string{"id", "name"})
	if err == nil {
		t.Fatal("expected missing tag error")
	}
}

func TestScanAndurelNotStruct(t *testing.T) {
	_, err := scanAndurel[int](assignScanner{int64(1)}, []string{"id"})
	if err == nil {
		t.Fatal("expected not a struct error")
	}
}

func TestScanAndurelDuplicateTag(t *testing.T) {
	type author struct {
		ID   int64  `andurel:"id"`
		Also int64  `andurel:"id"`
		Name string `andurel:"name"`
	}
	_, err := scanAndurel[author](assignScanner{int64(1), "ada"}, []string{"id", "name"})
	if err == nil {
		t.Fatal("expected duplicate tag error")
	}
}
