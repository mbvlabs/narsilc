package golang

import (
	"go/ast"
	goimporter "go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestAndurelGenericMethodTypechecks(t *testing.T) {
	const src = `package p

import "context"

type Queries struct{}

func (q *Queries) GetAuthor[T any](ctx context.Context, id int64) (T, error) {
	var z T
	return z, nil
}

type User struct {
	ID   int64  ` + "`andurel:\"id\"`" + `
	Name string ` + "`andurel:\"name\"`" + `
}

func use(q *Queries) (User, error) {
	return q.GetAuthor[User](context.Background(), 1)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Skipf("generic methods require go1.27: %v", err)
	}
	conf := types.Config{Importer: goimporter.Default()}
	if _, err := conf.Check("p", fset, []*ast.File{f}, nil); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
}
