package golang

import (
	"fmt"
	"regexp"

	"github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
	"github.com/mbvlabs/narsilc/internal/metadata"
	"github.com/mbvlabs/narsilc/internal/plugin"
)

var whereWord = regexp.MustCompile(`(?i)\bWHERE\b`)

type QueryBuilder struct {
	Filters  []BuilderFilter
	Orders   []BuilderOrder
	HasWhere bool
	Dollar   bool
}

type BuilderFilter struct {
	Column   string
	SQLName  string
	Method   string
	Operator string
	GoType   string
}

type BuilderOrder struct {
	Column string
	SQL    string
	Method string
}

func (q Query) HasBuilder() bool {
	return q.Builder != nil
}

func (q Query) BuilderType() string {
	return q.MethodName + "Builder"
}

func (q Query) andurelBuilder() bool {
	return q.andurel && q.AndurelGeneric()
}

func (q Query) BuilderTypeDecl() string {
	if q.andurelBuilder() {
		return q.BuilderType() + "[T any]"
	}
	return q.BuilderType()
}

func (q Query) BuilderTypeRef() string {
	if q.andurelBuilder() {
		return "*" + q.BuilderType() + "[T]"
	}
	return "*" + q.BuilderType()
}

func buildQueryBuilder(req *plugin.GenerateRequest, options *opts.Options, query *plugin.Query) (*QueryBuilder, error) {
	filters, orders, err := metadata.ParseBuilderAnnotations(query.Comments)
	if err != nil {
		return nil, err
	}
	if len(filters) == 0 && len(orders) == 0 {
		return nil, nil
	}

	engine := ""
	if req.Settings != nil {
		engine = req.Settings.Engine
	}
	qb := &QueryBuilder{
		HasWhere: whereWord.MatchString(query.Text),
		Dollar:   engine == "postgresql",
	}

	for _, f := range filters {
		col, err := findCatalogColumn(req, f.Column)
		if err != nil {
			return nil, fmt.Errorf("query %s: %w", query.Name, err)
		}
		goTyp := goType(req, options, col)
		exported := StructName(f.Column, options)
		for _, op := range f.Operators {
			placeholder := "?"
			if qb.Dollar {
				placeholder = "$%d"
			}
			qb.Filters = append(qb.Filters, BuilderFilter{
				Column:   f.Column,
				SQLName:  f.Column,
				Method:   exported + op.MethodSuffix(),
				Operator: fmt.Sprintf("%s %s %s", f.Column, op.SQL(), placeholder),
				GoType:   goTyp,
			})
		}
	}

	for _, o := range orders {
		exported := StructName(o.Column, options)
		qb.Orders = append(qb.Orders, BuilderOrder{
			Column: o.Column,
			SQL:    o.Column,
			Method: exported,
		})
	}

	return qb, nil
}

func findCatalogColumn(req *plugin.GenerateRequest, name string) (*plugin.Column, error) {
	if req.Catalog == nil {
		return nil, fmt.Errorf("unknown column %q", name)
	}
	var found []*plugin.Column
	for _, schema := range req.Catalog.Schemas {
		if schema.Name == "pg_catalog" || schema.Name == "information_schema" {
			continue
		}
		for _, table := range schema.Tables {
			for _, col := range table.Columns {
				if col.Name == name {
					found = append(found, col)
				}
			}
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("unknown column %q", name)
	}
	return found[0], nil
}

func filterBuilderComments(comments []string) []string {
	var out []string
	for _, c := range comments {
		if metadata.IsBuilderAnnotation(c) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func usesBuilder(queries []Query) bool {
	for _, q := range queries {
		if q.HasBuilder() {
			return true
		}
	}
	return false
}

func (q Query) builderFilterTypes() []string {
	if q.Builder == nil {
		return nil
	}
	var types []string
	for _, f := range q.Builder.Filters {
		types = append(types, f.GoType)
	}
	return types
}
