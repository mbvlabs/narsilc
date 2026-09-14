package compiler

import (
	"fmt"

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

	for _, f := range filters {
		if err := c.lookupBuilderColumn(q, f.Column); err != nil {
			return err
		}
	}
	for _, o := range orders {
		if err := c.lookupBuilderColumn(q, o.Column); err != nil {
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

func (c *Compiler) lookupBuilderColumn(q *Query, name string) error {
	if c.catalog == nil {
		return fmt.Errorf("query %s: catalog unavailable; cannot resolve column %q", q.Metadata.Name, name)
	}

	tables := fromRangeVars(q.RawStmt)
	if len(tables) == 0 {
		for _, schema := range c.catalog.Schemas {
			if schema.Name == "pg_catalog" || schema.Name == "information_schema" {
				continue
			}
			for _, table := range schema.Tables {
				tables = append(tables, table.Rel)
			}
		}
	}

	var matches int
	for _, rel := range tables {
		table, err := c.catalog.GetTable(rel)
		if err != nil {
			continue
		}
		for _, col := range table.Columns {
			if col.Name == name {
				matches++
			}
		}
	}
	if matches == 0 {
		return fmt.Errorf("query %s: unknown column %q for @filter/@order", q.Metadata.Name, name)
	}
	if matches > 1 {
		return fmt.Errorf("query %s: column %q is ambiguous for @filter/@order", q.Metadata.Name, name)
	}
	return nil
}

func fromRangeVars(root ast.Node) []*ast.TableName {
	found := astutils.Search(root, func(n ast.Node) bool {
		_, ok := n.(*ast.RangeVar)
		return ok
	})
	var out []*ast.TableName
	for _, item := range found.Items {
		rv, ok := item.(*ast.RangeVar)
		if !ok || rv.Relname == nil || *rv.Relname == "" {
			continue
		}
		t := &ast.TableName{Name: *rv.Relname}
		if rv.Schemaname != nil {
			t.Schema = *rv.Schemaname
		}
		out = append(out, t)
	}
	return out
}
