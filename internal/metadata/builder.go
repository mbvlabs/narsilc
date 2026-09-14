package metadata

import (
	"fmt"
	"strings"
)

const (
	QueryFlagFilter = "@filter"
	QueryFlagOrder  = "@order"
)

// FilterOp is a v1 whitelist of optional AND predicates.
type FilterOp string

const (
	FilterEq   FilterOp = "eq"
	FilterLike FilterOp = "like"
	FilterGte  FilterOp = "gte"
	FilterLte  FilterOp = "lte"
)

var filterOps = map[string]FilterOp{
	"eq":   FilterEq,
	"like": FilterLike,
	"gte":  FilterGte,
	"lte":  FilterLte,
}

var filterSQL = map[FilterOp]string{
	FilterEq:   "=",
	FilterLike: "LIKE",
	FilterGte:  ">=",
	FilterLte:  "<=",
}

var filterMethodSuffix = map[FilterOp]string{
	FilterEq:   "Eq",
	FilterLike: "Like",
	FilterGte:  "Gte",
	FilterLte:  "Lte",
}

func (op FilterOp) SQL() string {
	return filterSQL[op]
}

func (op FilterOp) MethodSuffix() string {
	return filterMethodSuffix[op]
}

// Filter is one column and the operators the generated builder may apply.
type Filter struct {
	Column    string
	Operators []FilterOp
}

// Order is one column the generated builder may sort by.
type Order struct {
	Column string
}

// ParseBuilderAnnotations reads @filter / @order comments. Other comments are ignored.
func ParseBuilderAnnotations(comments []string) ([]Filter, []Order, error) {
	var filters []Filter
	var orders []Order
	seenFilter := map[string]struct{}{}
	seenOrder := map[string]struct{}{}

	for _, line := range comments {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case QueryFlagFilter:
			if len(fields) != 3 {
				return nil, nil, fmt.Errorf("invalid @filter: expected \"@filter <column> <op>[,<op>]\", got %q", strings.TrimSpace(line))
			}
			col := fields[1]
			if _, dup := seenFilter[col]; dup {
				return nil, nil, fmt.Errorf("duplicate @filter column %q", col)
			}
			seenFilter[col] = struct{}{}
			var ops []FilterOp
			for _, raw := range strings.Split(fields[2], ",") {
				raw = strings.TrimSpace(raw)
				if raw == "" {
					continue
				}
				op, ok := filterOps[raw]
				if !ok {
					return nil, nil, fmt.Errorf("unsupported @filter operator %q on column %q (want eq, like, gte, lte)", raw, col)
				}
				ops = append(ops, op)
			}
			if len(ops) == 0 {
				return nil, nil, fmt.Errorf("@filter %s has no operators", col)
			}
			filters = append(filters, Filter{Column: col, Operators: ops})

		case QueryFlagOrder:
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("invalid @order: expected \"@order <column>[,<column>]\"")
			}
			// Allow "@order name, created_at, id" split across fields after commas.
			rest := strings.Join(fields[1:], " ")
			for _, raw := range strings.Split(rest, ",") {
				col := strings.TrimSpace(raw)
				if col == "" {
					continue
				}
				if _, dup := seenOrder[col]; dup {
					return nil, nil, fmt.Errorf("duplicate @order column %q", col)
				}
				seenOrder[col] = struct{}{}
				orders = append(orders, Order{Column: col})
			}
			if len(orders) == 0 {
				return nil, nil, fmt.Errorf("@order has no columns")
			}
		}
	}

	return filters, orders, nil
}

// IsBuilderAnnotation reports whether a comment line is a builder flag.
func IsBuilderAnnotation(comment string) bool {
	fields := strings.Fields(comment)
	if len(fields) == 0 {
		return false
	}
	return fields[0] == QueryFlagFilter || fields[0] == QueryFlagOrder
}
