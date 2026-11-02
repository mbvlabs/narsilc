package compiler

import (
	"fmt"
	"strings"

	"github.com/mbvlabs/narsilc/internal/sql/ast"
	"github.com/mbvlabs/narsilc/internal/sql/astutils"
	"github.com/mbvlabs/narsilc/internal/sql/sqlerr"
)

// refScope is what a column reference at one level of a statement can
// resolve against: the relations its FROM clause binds, the output names its
// target list assigns, and the scope of the query it is nested in, which a
// correlated reference resolves against.
type refScope struct {
	tables  []*Table
	aliases map[string]bool
	parent  *refScope

	// open marks a level whose relations are not all known — a function in
	// FROM whose columns the catalog cannot tell, or a FROM item the
	// compiler does not model — so any reference may resolve there.
	open bool
}

// validateColumnRefs reports the first column reference in a statement that
// no relation in scope offers. The output column and parameter checks only
// look at a bare column in the target list, ORDER BY and GROUP BY, or one
// compared with a parameter; this walks every expression, so a column inside
// a function call, an aggregate, CASE, COALESCE, a cast, an operator or a
// subquery is checked too.
//
// It errs on the side of accepting a query: a reference is reported only when
// every relation it could name is known.
func (c *Compiler) validateColumnRefs(qc *QueryCatalog, stmt ast.Node) error {
	return c.validateStmtRefs(qc, stmt, nil)
}

func (c *Compiler) validateStmtRefs(qc *QueryCatalog, stmt ast.Node, parent *refScope) error {
	switch n := stmt.(type) {
	case *ast.SelectStmt:
		return c.validateSelectRefs(qc, n, parent)
	case *ast.InsertStmt:
		return c.validateInsertRefs(qc, n, parent)
	case *ast.UpdateStmt:
		return c.validateUpdateRefs(qc, n, parent)
	case *ast.DeleteStmt:
		return c.validateDeleteRefs(qc, n, parent)
	}
	return nil
}

func (c *Compiler) validateWithRefs(qc *QueryCatalog, with *ast.WithClause, parent *refScope) error {
	if with == nil || with.Ctes == nil {
		return nil
	}
	for _, item := range with.Ctes.Items {
		cte, ok := item.(*ast.CommonTableExpr)
		if !ok {
			continue
		}
		if err := c.validateStmtRefs(qc, cte.Ctequery, parent); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) validateSelectRefs(qc *QueryCatalog, n *ast.SelectStmt, parent *refScope) error {
	if n == nil {
		return nil
	}
	if err := c.validateWithRefs(qc, n.WithClause, parent); err != nil {
		return err
	}
	// A set operation's ORDER BY names the result's columns, so only the
	// branches are checked.
	if n.Larg != nil || n.Rarg != nil {
		if err := c.validateSelectRefs(qc, n.Larg, parent); err != nil {
			return err
		}
		return c.validateSelectRefs(qc, n.Rarg, parent)
	}

	sc, ok := c.fromScope(qc, n, n.FromClause, parent)
	if !ok {
		return nil
	}
	if err := c.validateFromRefs(qc, n.FromClause, sc); err != nil {
		return err
	}
	exprs := []ast.Node{n.WhereClause}
	for _, l := range []*ast.List{n.TargetList, n.DistinctClause} {
		if l != nil {
			exprs = append(exprs, l)
		}
	}
	if err := c.validateExprRefs(qc, sc, exprs...); err != nil {
		return err
	}
	// Output aliases are not in scope within the expressions defining them.
	// Some dialects permit them in GROUP BY, HAVING and ORDER BY, however.
	aliasScope := *sc
	aliasScope.aliases = targetAliases(n.TargetList)
	exprs = []ast.Node{n.HavingClause, n.GroupClause}
	validateOrderBy := true
	if c.conf.StrictOrderBy != nil {
		validateOrderBy = *c.conf.StrictOrderBy
	}
	if validateOrderBy {
		for _, l := range []*ast.List{n.SortClause, n.WindowClause} {
			if l != nil {
				exprs = append(exprs, l)
			}
		}
	}
	if err := c.validateExprRefs(qc, &aliasScope, exprs...); err != nil {
		return err
	}
	// A VALUES list binds no relation, but a subquery inside one is still a
	// query of its own. A bare name is left alone: MySQL reads it as the
	// value of the column being inserted.
	if n.ValuesLists != nil {
		return c.validateSubqueryRefs(qc, sc, n.ValuesLists)
	}
	return nil
}

func (c *Compiler) validateInsertRefs(qc *QueryCatalog, n *ast.InsertStmt, parent *refScope) error {
	if err := c.validateWithRefs(qc, n.WithClause, parent); err != nil {
		return err
	}
	// The rows inserted come from a query that cannot see the table they go
	// into.
	if n.SelectStmt != nil {
		if err := c.validateStmtRefs(qc, n.SelectStmt, parent); err != nil {
			return err
		}
	}
	// ON CONFLICT and ON DUPLICATE KEY UPDATE name pseudo-relations
	// (EXCLUDED, VALUES()) the compiler does not model, so they are left to
	// the checks that already cover them.
	if n.ReturningList == nil || len(n.ReturningList.Items) == 0 {
		return nil
	}
	sc, ok := c.stmtScope(qc, n, parent)
	if !ok {
		return nil
	}
	return c.validateExprRefs(qc, sc, n.ReturningList)
}

func (c *Compiler) validateUpdateRefs(qc *QueryCatalog, n *ast.UpdateStmt, parent *refScope) error {
	if err := c.validateWithRefs(qc, n.WithClause, parent); err != nil {
		return err
	}
	sc, ok := c.stmtScope(qc, n, parent)
	if !ok {
		return nil
	}
	if err := c.validateFromRefs(qc, n.FromClause, sc); err != nil {
		return err
	}
	if err := c.validateFromRefs(qc, n.TableRefs, sc); err != nil {
		return err
	}
	// The target list's names are the columns being set, which the update
	// checks already cover; their values are expressions to check.
	exprs := []ast.Node{n.WhereClause, n.ReturningList}
	if n.TargetList != nil {
		for _, item := range n.TargetList.Items {
			if res, ok := item.(*ast.ResTarget); ok {
				exprs = append(exprs, res.Val)
			}
		}
	}
	return c.validateExprRefs(qc, sc, exprs...)
}

func (c *Compiler) validateDeleteRefs(qc *QueryCatalog, n *ast.DeleteStmt, parent *refScope) error {
	if err := c.validateWithRefs(qc, n.WithClause, parent); err != nil {
		return err
	}
	sc, ok := c.stmtScope(qc, n, parent)
	if !ok {
		return nil
	}
	// DELETE ... USING binds more relations than the ones deleted from.
	if n.UsingClause != nil && len(n.UsingClause.Items) > 0 {
		using, ok := c.fromScope(qc, &ast.SelectStmt{FromClause: n.UsingClause}, n.UsingClause, parent)
		if !ok {
			return nil
		}
		sc.tables = append(sc.tables, using.tables...)
		sc.open = sc.open || using.open
		if err := c.validateFromRefs(qc, n.UsingClause, sc); err != nil {
			return err
		}
	}
	if from, ok := n.FromClause.(*ast.List); ok {
		if err := c.validateFromRefs(qc, from, sc); err != nil {
			return err
		}
	} else if n.FromClause != nil {
		if err := c.validateFromRefs(qc, &ast.List{Items: []ast.Node{n.FromClause}}, sc); err != nil {
			return err
		}
	}
	return c.validateExprRefs(qc, sc, n.WhereClause, n.ReturningList)
}

// stmtScope is the scope of an INSERT, UPDATE or DELETE: the relations it
// writes and reads, and the OLD and NEW rows its RETURNING clause may name.
func (c *Compiler) stmtScope(qc *QueryCatalog, n ast.Node, parent *refScope) (*refScope, bool) {
	tables, err := c.sourceTables(qc, n)
	if err != nil {
		return nil, false
	}
	rtables, err := c.returningTables(qc, n)
	if err != nil {
		return nil, false
	}
	sc := &refScope{tables: append(tables, rtables...), parent: parent}
	switch s := n.(type) {
	case *ast.UpdateStmt:
		sc.open = !fromIsModelled(s.FromClause) || !fromIsModelled(s.TableRefs)
	case *ast.DeleteStmt:
		if s.FromClause != nil {
			sc.open = !fromIsModelled(&ast.List{Items: []ast.Node{s.FromClause}})
		}
	}
	sc.open = sc.open || hasUnnamedColumn(sc.tables)
	return sc, true
}

// fromScope is the scope a SELECT's FROM clause binds. It reports false when
// the relations cannot be worked out, which leaves the query to the checks
// that would report why.
func (c *Compiler) fromScope(qc *QueryCatalog, n ast.Node, from *ast.List, parent *refScope) (*refScope, bool) {
	tables, err := c.sourceTables(qc, n)
	if err != nil {
		return nil, false
	}
	sc := &refScope{tables: tables, parent: parent}
	sc.open = !fromIsModelled(from) || hasUnnamedColumn(tables)
	return sc, true
}

// fromIsModelled reports whether every relation a FROM clause binds is one
// the compiler knows the columns of. A function whose columns are not named
// by an alias, and a FROM item of a kind the compiler skips, are not.
func fromIsModelled(from *ast.List) bool {
	if from == nil {
		return true
	}
	for _, item := range from.Items {
		if !fromItemIsModelled(item) {
			return false
		}
	}
	return true
}

func fromItemIsModelled(item ast.Node) bool {
	switch n := item.(type) {
	case nil:
		return true
	case *ast.RangeVar:
		return true
	case *ast.RangeSubselect:
		// The compiler names a subquery's columns after its target list, not
		// after a column list its alias gives, and cannot list the columns a
		// star expands to over a relation it does not model.
		if n.Alias != nil && n.Alias.Colnames != nil && len(n.Alias.Colnames.Items) > 0 {
			return false
		}
		return selectIsModelled(n.Subquery)
	case *ast.RangeFunction:
		return n.Alias != nil && n.Alias.Colnames != nil && len(n.Alias.Colnames.Items) > 0
	case *ast.JoinExpr:
		if n.Alias != nil {
			return false
		}
		return fromItemIsModelled(n.Larg) && fromItemIsModelled(n.Rarg)
	case *ast.List:
		return fromIsModelled(n)
	}
	return false
}

func selectIsModelled(n ast.Node) bool {
	sel, ok := n.(*ast.SelectStmt)
	if !ok {
		return false
	}
	if sel.Larg != nil {
		return selectIsModelled(sel.Larg)
	}
	return fromIsModelled(sel.FromClause)
}

// hasUnnamedColumn reports whether a relation in scope has a column the
// compiler could not name, as an unaliased expression in a subquery or CTE
// is. The database names it, so a reference to it cannot be checked.
func hasUnnamedColumn(tables []*Table) bool {
	for _, t := range tables {
		for _, col := range t.Columns {
			if col.Name == "" {
				return true
			}
		}
	}
	return false
}

func targetAliases(targets *ast.List) map[string]bool {
	if targets == nil {
		return nil
	}
	aliases := map[string]bool{}
	for _, item := range targets.Items {
		if res, ok := item.(*ast.ResTarget); ok && res.Name != nil && *res.Name != "" {
			aliases[*res.Name] = true
		}
	}
	return aliases
}

// validateFromRefs checks what a FROM clause evaluates: the ON condition of
// each join, the arguments of a function, and the subqueries it selects
// from. A subquery in FROM may be LATERAL, so it sees the whole clause.
func (c *Compiler) validateFromRefs(qc *QueryCatalog, from *ast.List, sc *refScope) error {
	if from == nil {
		return nil
	}
	for _, item := range from.Items {
		if err := c.validateFromItemRefs(qc, sc, item); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) validateFromItemRefs(qc *QueryCatalog, sc *refScope, item ast.Node) error {
	switch n := item.(type) {
	case *ast.JoinExpr:
		if err := c.validateFromItemRefs(qc, sc, n.Larg); err != nil {
			return err
		}
		if err := c.validateFromItemRefs(qc, sc, n.Rarg); err != nil {
			return err
		}
		return c.validateExprRefs(qc, sc, n.Quals)
	case *ast.RangeSubselect:
		return c.validateStmtRefs(qc, n.Subquery, sc)
	case *ast.RangeFunction:
		if n.Functions != nil {
			return c.validateExprRefs(qc, sc, n.Functions)
		}
	case *ast.List:
		return c.validateFromRefs(qc, n, sc)
	}
	return nil
}

// validateExprRefs checks every column reference in the expressions against
// the scope, and every subquery among them against a scope nested in it.
func (c *Compiler) validateExprRefs(qc *QueryCatalog, sc *refScope, exprs ...ast.Node) error {
	return c.walkExprRefs(qc, sc, true, exprs...)
}

// validateSubqueryRefs checks only the subqueries among the expressions.
func (c *Compiler) validateSubqueryRefs(qc *QueryCatalog, sc *refScope, exprs ...ast.Node) error {
	return c.walkExprRefs(qc, sc, false, exprs...)
}

func (c *Compiler) walkExprRefs(qc *QueryCatalog, sc *refScope, checkRefs bool, exprs ...ast.Node) error {
	w := &refWalker{c: c, qc: qc, scope: sc, checkRefs: checkRefs}
	for _, e := range exprs {
		if isNilNode(e) {
			continue
		}
		astutils.Walk(w, e)
		if w.err != nil {
			return w.err
		}
	}
	return nil
}

func isNilNode(n ast.Node) bool {
	switch v := n.(type) {
	case nil:
		return true
	case *ast.List:
		return v == nil
	}
	return false
}

type refWalker struct {
	c         *Compiler
	qc        *QueryCatalog
	scope     *refScope
	checkRefs bool
	err       error
}

func (w *refWalker) Visit(node ast.Node) astutils.Visitor {
	if w.err != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.ColumnRef:
		if w.checkRefs {
			w.err = w.c.checkColumnRef(w.scope, n)
		}
		return nil
	case *ast.SelectStmt:
		w.err = w.c.validateSelectRefs(w.qc, n, w.scope)
		return nil
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt:
		w.err = w.c.validateStmtRefs(w.qc, n, w.scope)
		return nil
	}
	return w
}

// checkColumnRef resolves a reference against the scope it appears in and
// then the scopes around it. It reports the reference only when no level it
// could resolve at is open.
func (c *Compiler) checkColumnRef(sc *refScope, ref *ast.ColumnRef) error {
	if ref.Fields == nil {
		return nil
	}
	for _, item := range ref.Fields.Items {
		if _, ok := item.(*ast.A_Star); ok {
			return nil
		}
	}
	if c.isImplicitColumnRef(ref) {
		return nil
	}
	parts := stringSlice(ref.Fields)
	var schema, rel, name string
	switch len(parts) {
	case 1:
		name = parts[0]
	case 2:
		rel, name = parts[0], parts[1]
	case 3:
		schema, rel, name = parts[0], parts[1], parts[2]
	default:
		return nil
	}
	// MySQL reads $1 as an identifier rather than a placeholder. Queries
	// written for another engine use it as one, and the parameter checks
	// are the ones to say so.
	if strings.HasPrefix(name, "$") {
		return nil
	}

	relFound := false
	for s := sc; s != nil; s = s.parent {
		if s.open {
			return nil
		}
		if rel == "" && s.aliases[name] {
			return nil
		}
		for _, t := range s.tables {
			if t.Rel == nil {
				continue
			}
			if rel == "" {
				// A bare name may be a whole row, as row_to_json(u) and
				// count(u) take one.
				if t.Rel.Name == name || tableHasColumn(t, name) {
					return nil
				}
				continue
			}
			if t.Rel.Name != rel {
				continue
			}
			if schema != "" && t.Rel.Schema != "" && t.Rel.Schema != schema {
				continue
			}
			relFound = true
			if tableHasColumn(t, name) {
				return nil
			}
		}
	}
	if rel != "" && !relFound {
		return &sqlerr.Error{
			Code:     "42703",
			Message:  fmt.Sprintf("table alias %q does not exist", rel),
			Location: ref.Location,
		}
	}
	return &sqlerr.Error{
		Code:     "42703",
		Message:  fmt.Sprintf("column %q does not exist", name),
		Location: ref.Location,
	}
}

func tableHasColumn(t *Table, name string) bool {
	for _, col := range t.Columns {
		if col.Name == name {
			return true
		}
	}
	return false
}
