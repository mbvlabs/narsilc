package narsilc

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

func TestScan(t *testing.T) {
	type author struct {
		ID   int64          `andurel:"id"`
		Name string         `andurel:"name"`
		Bio  sql.NullString `andurel:"bio"`
		Skip string
	}

	got, err := Scan[author](assignScanner{int64(7), "ada", sql.NullString{String: "bio", Valid: true}}, []string{"id", "name", "bio"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "ada" || got.Bio.String != "bio" || got.Skip != "" {
		t.Fatalf("unexpected scan: %+v", got)
	}
}

func TestScanProjection(t *testing.T) {
	type author struct {
		ID        int64  `andurel:"id"`
		Name      string `andurel:"name"`
		PostCount int64  `andurel:"post_count"`
	}

	got, err := Scan[author](assignScanner{int64(1), "ada"}, []string{"id", "name"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 || got.Name != "ada" || got.PostCount != 0 {
		t.Fatalf("unexpected projection: %+v", got)
	}
}

func TestScanMissingTag(t *testing.T) {
	type author struct {
		ID int64 `andurel:"id"`
	}
	_, err := Scan[author](assignScanner{int64(1), "ada"}, []string{"id", "name"})
	if err == nil {
		t.Fatal("expected missing tag error")
	}
}

func TestScanNotStruct(t *testing.T) {
	_, err := Scan[int](assignScanner{int64(1)}, []string{"id"})
	if err == nil {
		t.Fatal("expected not a struct error")
	}
}

func TestScanDuplicateTag(t *testing.T) {
	type author struct {
		ID   int64  `andurel:"id"`
		Also int64  `andurel:"id"`
		Name string `andurel:"name"`
	}
	_, err := Scan[author](assignScanner{int64(1), "ada"}, []string{"id", "name"})
	if err == nil {
		t.Fatal("expected duplicate tag error")
	}
}
