package golang

import (
	"testing"

	"github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
	"github.com/mbvlabs/narsilc/internal/plugin"
)

func TestFindCatalogColumnScopesToQueryTables(t *testing.T) {
	req := &plugin.GenerateRequest{
		Settings: &plugin.Settings{Engine: "postgresql"},
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{
				{
					Name: "public",
					Tables: []*plugin.Table{
						{
							Rel: &plugin.Identifier{Schema: "public", Name: "logs"},
							Columns: []*plugin.Column{
								{
									Name:  "created_at",
									Type:  &plugin.Identifier{Name: "timestamp"},
									Table: &plugin.Identifier{Schema: "public", Name: "logs"},
								},
							},
						},
						{
							Rel: &plugin.Identifier{Schema: "public", Name: "authors"},
							Columns: []*plugin.Column{
								{
									Name:  "created_at",
									Type:  &plugin.Identifier{Name: "timestamptz"},
									Table: &plugin.Identifier{Schema: "public", Name: "authors"},
								},
							},
						},
					},
				},
			},
		},
	}
	query := &plugin.Query{
		Name: "ListAuthors",
		Columns: []*plugin.Column{
			{
				Name:  "id",
				Table: &plugin.Identifier{Schema: "public", Name: "authors"},
			},
		},
	}
	col, err := findCatalogColumn(req, query, "created_at")
	if err != nil {
		t.Fatal(err)
	}
	if col.Table == nil || col.Table.Name != "authors" {
		t.Fatalf("expected authors.created_at, got %+v", col.Table)
	}
	if col.Type == nil || col.Type.Name != "timestamptz" {
		t.Fatalf("expected timestamptz, got %+v", col.Type)
	}
}

func TestFindCatalogColumnQualifiedJoinTypes(t *testing.T) {
	usersID := &plugin.Column{
		Name:         "id",
		OriginalName: "id",
		NotNull:      true,
		Type:         &plugin.Identifier{Name: "bigserial"},
		Table:        &plugin.Identifier{Schema: "public", Name: "users"},
		TableAlias:   "u",
	}
	tokensID := &plugin.Column{
		Name:         "id",
		OriginalName: "id",
		NotNull:      true,
		Type:         &plugin.Identifier{Name: "uuid"},
		Table:        &plugin.Identifier{Schema: "public", Name: "tokens"},
		TableAlias:   "t",
	}
	req := &plugin.GenerateRequest{
		Settings: &plugin.Settings{Engine: "postgresql"},
		Catalog: &plugin.Catalog{
			DefaultSchema: "public",
			Schemas: []*plugin.Schema{
				{
					Name: "public",
					Tables: []*plugin.Table{
						{
							Rel:     &plugin.Identifier{Schema: "public", Name: "users"},
							Columns: []*plugin.Column{usersID},
						},
						{
							Rel:     &plugin.Identifier{Schema: "public", Name: "tokens"},
							Columns: []*plugin.Column{tokensID},
						},
					},
				},
			},
		},
	}
	query := &plugin.Query{
		Name:    "ListUserTokens",
		Columns: []*plugin.Column{usersID, tokensID},
	}
	options := &opts.Options{SqlPackage: "pgx/v5"}

	userCol, err := findCatalogColumn(req, query, "u.id")
	if err != nil {
		t.Fatal(err)
	}
	if got := goType(req, options, userCol); got != "int64" {
		t.Fatalf("u.id: got %s want int64", got)
	}

	tokenCol, err := findCatalogColumn(req, query, "t.id")
	if err != nil {
		t.Fatal(err)
	}
	if got := goType(req, options, tokenCol); got != "pgtype.UUID" {
		t.Fatalf("t.id: got %s want pgtype.UUID", got)
	}
}
