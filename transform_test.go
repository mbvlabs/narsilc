package narsilc

import (
	"errors"
	"strings"
	"testing"
)

type authorRow struct {
	ID   int64  `andurel:"id"`
	Name string `andurel:"name"`
}

type author struct {
	ID   int64  `andurel:"id"`
	Name string `andurel:"name"`
}

type mappedAuthor struct {
	ID   int64
	Name string
}

func (m *mappedAuthor) Transform(row authorRow) error {
	m.ID = row.ID
	m.Name = strings.ToUpper(row.Name)
	return nil
}

type failAuthor struct{}

func (f *failAuthor) Transform(authorRow) error {
	return errors.New("transform failed")
}

func TestFromRowIdentity(t *testing.T) {
	row := authorRow{ID: 7, Name: "ada"}
	got, err := FromRow[authorRow, authorRow](row)
	if err != nil {
		t.Fatal(err)
	}
	if got != row {
		t.Fatalf("identity: %+v", got)
	}
}

func TestFromRowTransformSuccess(t *testing.T) {
	got, err := FromRow[mappedAuthor, authorRow](authorRow{ID: 7, Name: "ada"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "ADA" {
		t.Fatalf("transform: %+v", got)
	}
}

func TestFromRowTransformError(t *testing.T) {
	_, err := FromRow[failAuthor, authorRow](authorRow{ID: 1, Name: "ada"})
	if err == nil || err.Error() != "transform failed" {
		t.Fatalf("err = %v", err)
	}
}

func TestFromRowMissingTransform(t *testing.T) {
	_, err := FromRow[author, authorRow](authorRow{ID: 1, Name: "ada"})
	if err == nil {
		t.Fatal("expected missing transform error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "narsilc.author") ||
		!strings.Contains(msg, "narsilc.authorRow") ||
		!strings.Contains(msg, "Transformer") {
		t.Fatalf("error should name both types, got %q", msg)
	}
}

func TestReadWithoutTransformScansT(t *testing.T) {
	got, err := Read[author, authorRow](
		assignScanner{int64(7), "ada"},
		[]string{"id", "name"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "ada" {
		t.Fatalf("scan T: %+v", got)
	}
}

func TestReadWithTransformScansRow(t *testing.T) {
	got, err := Read[mappedAuthor, authorRow](
		assignScanner{int64(7), "ada"},
		[]string{"id", "name"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || got.Name != "ADA" {
		t.Fatalf("scan row: %+v", got)
	}
}

func TestReadIdentityRow(t *testing.T) {
	got, err := Read[authorRow, authorRow](
		assignScanner{int64(3), "lin"},
		[]string{"id", "name"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 3 || got.Name != "lin" {
		t.Fatalf("scan row type: %+v", got)
	}
}
