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

	seenMethods := map[string]string{}
	for _, f := range filters {
		col, err := findCatalogColumn(req, query, f.Column)
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
			method := exported + op.MethodSuffix()
			if prev, ok := seenMethods[method]; ok {
				return nil, fmt.Errorf("query %s: @filter %q and %q both generate method %s", query.Name, prev, f.Column, method)
			}
			seenMethods[method] = f.Column
			qb.Filters = append(qb.Filters, BuilderFilter{
				Column:   f.Column,
				SQLName:  f.Column,
				Method:   method,
				Operator: fmt.Sprintf("%s %s %s", f.Column, op.SQL(), placeholder),
				GoType:   goTyp,
			})
		}
	}

	for _, o := range orders {
		exported := StructName(o.Column, options)
		if prev, ok := seenMethods[exported]; ok {
			return nil, fmt.Errorf("query %s: @order %q and %q both generate method OrderBy%s", query.Name, prev, o.Column, exported)
		}
		seenMethods[exported] = o.Column
		qb.Orders = append(qb.Orders, BuilderOrder{
			Column: o.Column,
			SQL:    o.Column,
			Method: exported,
		})
	}

	return qb, nil
}

func findCatalogColumn(req *plugin.GenerateRequest, query *plugin.Query, name string) (*plugin.Column, error) {
	spec, err := metadata.ParseColumnRef(name)
	if err != nil {
		return nil, err
	}

	aliasToTable, queryTables := queryRelationIndex(query)
	wantSchema, wantTable := spec.Schema, spec.Rel
	if spec.Rel != "" && spec.Schema == "" {
		if t, ok := aliasToTable[spec.Rel]; ok {
			wantTable = t.Name
			wantSchema = t.Schema
		}
	}

	var found []*plugin.Column
	if req.Catalog != nil {
		defaultSchema := req.Catalog.DefaultSchema
		for _, schema := range req.Catalog.Schemas {
			if schema.Name == "pg_catalog" || schema.Name == "information_schema" {
				continue
			}
			for _, table := range schema.Tables {
				if table.Rel == nil {
					continue
				}
				if !catalogTableInScope(table.Rel, wantSchema, wantTable, queryTables, defaultSchema) {
					continue
				}
				for _, col := range table.Columns {
					if col.Name == spec.Name {
						found = append(found, col)
					}
				}
			}
		}
	}

	if len(found) == 0 {
		for _, col := range query.GetColumns() {
			if !outputColumnMatches(col, spec) {
				continue
			}
			found = append(found, col)
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("unknown column %q", name)
	}
	if ambiguousCatalogColumns(found) {
		return nil, fmt.Errorf("column %q is ambiguous", name)
	}
	return found[0], nil
}

func queryRelationIndex(query *plugin.Query) (map[string]*plugin.Identifier, map[string]*plugin.Identifier) {
	aliasToTable := map[string]*plugin.Identifier{}
	queryTables := map[string]*plugin.Identifier{}
	if query == nil {
		return aliasToTable, queryTables
	}
	for _, col := range query.GetColumns() {
		if col.Table == nil {
			continue
		}
		queryTables[catalogTableKey(col.Table)] = col.Table
		if col.TableAlias != "" {
			aliasToTable[col.TableAlias] = col.Table
		}
	}
	return aliasToTable, queryTables
}

func catalogTableKey(id *plugin.Identifier) string {
	if id == nil {
		return ""
	}
	return id.Schema + "." + id.Name
}

func catalogTableInScope(rel *plugin.Identifier, wantSchema, wantTable string, queryTables map[string]*plugin.Identifier, defaultSchema string) bool {
	relSchema := rel.Schema
	if relSchema == "" {
		relSchema = defaultSchema
	}
	if wantTable != "" {
		if rel.Name != wantTable {
			return false
		}
		if wantSchema != "" && relSchema != wantSchema {
			return false
		}
		return true
	}
	// Bare names: only tables that already appear in the query's output.
	if _, ok := queryTables[catalogTableKey(rel)]; ok {
		return true
	}
	if _, ok := queryTables["."+rel.Name]; ok {
		return true
	}
	if defaultSchema != "" {
		if _, ok := queryTables[defaultSchema+"."+rel.Name]; ok {
			return true
		}
	}
	return false
}

func outputColumnMatches(col *plugin.Column, spec metadata.ColumnRef) bool {
	colName := col.OriginalName
	if colName == "" {
		colName = col.Name
	}
	if colName != spec.Name {
		return false
	}
	if spec.Rel == "" {
		return true
	}
	if col.TableAlias == spec.Rel {
		return spec.Schema == "" || (col.Table != nil && (col.Table.Schema == spec.Schema || col.Table.Schema == ""))
	}
	return col.Table != nil && col.Table.Name == spec.Rel && (spec.Schema == "" || col.Table.Schema == spec.Schema || col.Table.Schema == "")
}

func ambiguousCatalogColumns(found []*plugin.Column) bool {
	if len(found) < 2 {
		return false
	}
	key := func(c *plugin.Column) string {
		if c.Table == nil {
			return c.Name
		}
		return c.Table.Schema + "." + c.Table.Name + "." + c.Name
	}
	first := key(found[0])
	for _, c := range found[1:] {
		if key(c) != first {
			return true
		}
	}
	return false
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
