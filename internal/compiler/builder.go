package compiler

import (
	"fmt"
	"strings"

	coreanalyzer "github.com/mbvlabs/narsilc/internal/core/analyzer"
	"github.com/mbvlabs/narsilc/internal/metadata"
	"github.com/mbvlabs/narsilc/internal/sql/ast"
	"github.com/mbvlabs/narsilc/internal/sql/astutils"
)

func (c *Compiler) applyBuilder(q *Query) error {
	filters, orders, err := metadata.ParseBuilderAnnotations(q.Metadata.Comments)
	if err != nil {
		return err
	}
	if len(filters) == 0 && len(orders) == 0 {
		return nil
	}
	q.Metadata.Filters = filters
	q.Metadata.Orders = orders

	if q.Metadata.Cmd != metadata.CmdMany {
		return fmt.Errorf("@filter and @order are only supported on :many queries")
	}

	if q.RawStmt == nil {
		return fmt.Errorf("query %s: missing AST for builder validation", q.Metadata.Name)
	}
	sel, ok := q.RawStmt.Stmt.(*ast.SelectStmt)
	if !ok {
		return fmt.Errorf("@filter and @order are only supported on SELECT queries")
	}
	if listHasItems(sel.SortClause) {
		return fmt.Errorf("query %s: annotated queries must not include ORDER BY; use @order", q.Metadata.Name)
	}
	if nodePresent(sel.LimitCount) {
		return fmt.Errorf("query %s: annotated queries must not include LIMIT; use the builder Limit method", q.Metadata.Name)
	}
	if nodePresent(sel.LimitOffset) {
		return fmt.Errorf("query %s: annotated queries must not include OFFSET; use the builder Offset method", q.Metadata.Name)
	}

	tables, err := c.builderSourceTables(q)
	if err != nil {
		return fmt.Errorf("query %s: %w", q.Metadata.Name, err)
	}

	for _, f := range filters {
		if err := lookupBuilderColumn(q.Metadata.Name, f.Column, tables, c.defaultSchema()); err != nil {
			return err
		}
	}
	for _, o := range orders {
		if err := lookupBuilderColumn(q.Metadata.Name, o.Column, tables, c.defaultSchema()); err != nil {
			return err
		}
	}
	return nil
}

func listHasItems(l *ast.List) bool {
	return l != nil && len(l.Items) > 0
}

func nodePresent(n ast.Node) bool {
	if n == nil {
		return false
	}
	_, todo := n.(*ast.TODO)
	return !todo
}

func (c *Compiler) defaultSchema() string {
	if c.catalog != nil && c.catalog.DefaultSchema != "" {
		return c.catalog.DefaultSchema
	}
	return "public"
}

func (c *Compiler) builderSourceTables(q *Query) ([]*Table, error) {
	if q.RawStmt == nil {
		return nil, fmt.Errorf("missing AST for builder validation")
	}
	if c.catalog != nil {
		qc, err := c.buildQueryCatalog(c.catalog, q.RawStmt.Stmt, nil)
		if err != nil {
			return nil, err
		}
		return c.sourceTables(qc, q.RawStmt.Stmt)
	}
	if c.coreCatalog != nil {
		return c.coreBuilderSourceTables(q.RawStmt.Stmt)
	}
	return nil, fmt.Errorf("catalog unavailable; cannot resolve column")
}

func lookupBuilderColumn(queryName, name string, tables []*Table, defaultSchema string) error {
	spec, err := metadata.ParseColumnRef(name)
	if err != nil {
		return fmt.Errorf("query %s: %w", queryName, err)
	}

	var matches int
	var quals []string
	var relFound bool
	seenQual := map[string]struct{}{}

	for _, t := range tables {
		if t == nil || t.Rel == nil {
			continue
		}
		if spec.Schema != "" && relationSchema(t, defaultSchema) != spec.Schema {
			continue
		}
		if spec.Rel != "" && t.Rel.Name != spec.Rel {
			continue
		}
		relFound = true
		for _, col := range t.Columns {
			if col.Name != spec.Name {
				continue
			}
			matches++
			if t.Rel.Name != "" {
				q := t.Rel.Name + "." + spec.Name
				if _, ok := seenQual[q]; !ok {
					seenQual[q] = struct{}{}
					quals = append(quals, q)
				}
			}
		}
	}

	if spec.Rel != "" && !relFound {
		if spec.Schema != "" {
			return fmt.Errorf("query %s: unknown relation %q for @filter/@order", queryName, spec.Schema+"."+spec.Rel)
		}
		return fmt.Errorf("query %s: unknown alias %q for @filter/@order", queryName, spec.Rel)
	}
	if matches == 0 {
		return fmt.Errorf("query %s: unknown column %q for @filter/@order", queryName, spec.String())
	}
	if matches > 1 {
		if spec.Rel == "" && len(quals) > 1 {
			return fmt.Errorf("query %s: column %q is ambiguous for @filter/@order; qualify as %s", queryName, spec.String(), strings.Join(quals, " or "))
		}
		return fmt.Errorf("query %s: column %q is ambiguous for @filter/@order", queryName, spec.String())
	}
	return nil
}

func relationSchema(t *Table, defaultSchema string) string {
	if t.Rel.Schema != "" {
		return t.Rel.Schema
	}
	return defaultSchema
}

func (c *Compiler) coreBuilderSourceTables(node ast.Node) ([]*Table, error) {
	ctes, err := c.coreBuilderCTEs(node)
	if err != nil {
		return nil, err
	}

	list := &ast.List{}
	switch n := node.(type) {
	case *ast.SelectStmt:
		var tv tableVisitor
		astutils.Walk(&tv, n.FromClause)
		list = &tv.list
	default:
		return nil, fmt.Errorf("@filter and @order are only supported on SELECT queries")
	}

	var tables []*Table
	for _, item := range list.Items {
		switch n := item.(type) {
		case *ast.RangeVar:
			table, err := c.coreBindRangeVar(n, ctes)
			if err != nil {
				return nil, err
			}
			tables = append(tables, table)
		case *ast.RangeSubselect:
			table, err := c.coreBindSubselect(n)
			if err != nil {
				return nil, err
			}
			tables = append(tables, table)
		case *ast.RangeFunction:
			continue
		default:
			return nil, fmt.Errorf("sourceTable: unsupported list item type: %T", n)
		}
	}
	return tables, nil
}

func (c *Compiler) coreBuilderCTEs(node ast.Node) (map[string]*Table, error) {
	var with *ast.WithClause
	switch n := node.(type) {
	case *ast.SelectStmt:
		with = n.WithClause
	case *ast.InsertStmt:
		with = n.WithClause
	case *ast.UpdateStmt:
		with = n.WithClause
	case *ast.DeleteStmt:
		with = n.WithClause
	}
	ctes := map[string]*Table{}
	if with == nil || with.Ctes == nil {
		return ctes, nil
	}
	for _, item := range with.Ctes.Items {
		cte, ok := item.(*ast.CommonTableExpr)
		if !ok || cte.Ctename == nil {
			continue
		}
		res, err := coreanalyzer.Prepare(c.coreCatalog, cte.Ctequery)
		if err != nil {
			return nil, err
		}
		rel := &ast.TableName{Name: *cte.Ctename}
		cols := make([]*Column, 0, len(res.Columns))
		for i, col := range res.Columns {
			name := col.Name
			if cte.Aliascolnames != nil && i < len(cte.Aliascolnames.Items) {
				if val, ok := cte.Aliascolnames.Items[i].(*ast.String); ok {
					name = val.Str
				}
			}
			cols = append(cols, &Column{Name: name, Table: rel})
		}
		ctes[*cte.Ctename] = &Table{Rel: rel, Columns: cols}
	}
	return ctes, nil
}

func (c *Compiler) coreBindRangeVar(n *ast.RangeVar, ctes map[string]*Table) (*Table, error) {
	if n.Relname == nil || *n.Relname == "" {
		return nil, fmt.Errorf("range var: missing relation name")
	}
	name := *n.Relname
	if cte, ok := ctes[name]; ok && (n.Schemaname == nil || *n.Schemaname == "") {
		table := &Table{Rel: &ast.TableName{Name: name}, Columns: cte.Columns}
		if n.Alias != nil && n.Alias.Aliasname != nil {
			table.Rel = &ast.TableName{Name: *n.Alias.Aliasname}
		}
		return table, nil
	}
	schema := ""
	if n.Schemaname != nil {
		schema = *n.Schemaname
	}
	if schema == "" {
		schema = "public"
	}
	nsOID, err := c.coreCatalog.NamespaceOID(schema)
	if err != nil {
		return nil, fmt.Errorf("schema %q: %w", schema, err)
	}
	classOID, err := c.coreCatalog.ClassOID(nsOID, name)
	if err != nil {
		return nil, fmt.Errorf("relation %q.%q: %w", schema, name, err)
	}
	cols, err := c.coreCatalog.ClassColumns(classOID)
	if err != nil {
		return nil, err
	}
	table := &Table{
		Rel: &ast.TableName{Schema: schema, Name: name},
	}
	for _, col := range cols {
		if col.Hidden {
			continue
		}
		table.Columns = append(table.Columns, &Column{Name: col.Name, Table: table.Rel})
	}
	if n.Alias != nil && n.Alias.Aliasname != nil {
		table.Rel = &ast.TableName{
			Schema: table.Rel.Schema,
			Name:   *n.Alias.Aliasname,
		}
	}
	return table, nil
}

func (c *Compiler) coreBindSubselect(n *ast.RangeSubselect) (*Table, error) {
	res, err := coreanalyzer.Prepare(c.coreCatalog, n.Subquery)
	if err != nil {
		return nil, err
	}
	tableName := ""
	if n.Alias != nil && n.Alias.Aliasname != nil {
		tableName = *n.Alias.Aliasname
	}
	rel := &ast.TableName{Name: tableName}
	cols := make([]*Column, 0, len(res.Columns))
	for _, col := range res.Columns {
		cols = append(cols, &Column{Name: col.Name, Table: rel})
	}
	return &Table{Rel: rel, Columns: cols}, nil
}
