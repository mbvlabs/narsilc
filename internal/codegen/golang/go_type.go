package golang

import (
	"maps"
	"strings"

	"github.com/mbvlabs/narsilc/internal/codegen/golang/opts"
	"github.com/mbvlabs/narsilc/internal/codegen/sdk"
	"github.com/mbvlabs/narsilc/internal/plugin"
)

func addExtraGoStructTags(tags map[string]string, req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) {
	for _, override := range options.Overrides {
		oride := override.ShimOverride
		if oride.GoType.StructTags == nil {
			continue
		}
		if override.MatchesColumn(col) {
			maps.Copy(tags, oride.GoType.StructTags)
			continue
		}
		if !override.Matches(col.Table, req.Catalog.DefaultSchema) {
			// Different table.
			continue
		}
		cname := col.Name
		if col.OriginalName != "" {
			cname = col.OriginalName
		}
		if !sdk.MatchString(oride.ColumnName, cname) {
			// Different column.
			continue
		}
		// Add the extra tags.
		maps.Copy(tags, oride.GoType.StructTags)
	}
}

func goType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	// Check if the column's type has been overridden
	for _, override := range options.Overrides {
		oride := override.ShimOverride

		if oride.GoType.TypeName == "" {
			continue
		}
		cname := col.Name
		if col.OriginalName != "" {
			cname = col.OriginalName
		}
		sameTable := override.Matches(col.Table, req.Catalog.DefaultSchema)
		if oride.Column != "" && sdk.MatchString(oride.ColumnName, cname) && sameTable {
			if col.IsSqlcSlice {
				return "[]" + oride.GoType.TypeName
			}
			return oride.GoType.TypeName
		}
	}
	typ := goInnerType(req, options, col)
	if col.IsSqlcSlice {
		return "[]" + typ
	}
	if col.IsArray {
		return strings.Repeat("[]", int(col.ArrayDims)) + typ
	}
	return typ
}

// goParamType is goType for query parameters. Under Andurel row mapping a
// NOT NULL PostgreSQL date/time value is passed as time.Time: pgx encodes it
// for these columns, and a value the schema says can never be NULL does not
// need a pgtype validity flag at the call site. Result rows keep the
// representation database.nullType selects.
func goParamType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	if options != nil && options.AndurelRowMapping() && col != nil && col.Type != nil &&
		col.NotNull && !col.IsArray && !col.IsSqlcSlice &&
		req.Settings != nil && req.Settings.Engine == "postgresql" {
		if typ, ok := postgresNonNullParamType(sdk.DataType(col.Type)); ok {
			return typ
		}
	}
	return goType(req, options, col)
}

func goInnerType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	// package overrides have a higher precedence
	for _, override := range options.Overrides {
		oride := override.ShimOverride
		if oride.GoType.TypeName == "" {
			continue
		}
		if override.MatchesColumn(col) {
			return oride.GoType.TypeName
		}
	}

	// TODO: Extend the engine interface to handle types
	switch req.Settings.Engine {
	case "mysql":
		return mysqlType(req, options, col)
	case "postgresql":
		return postgresType(req, options, col)
	case "sqlite":
		return sqliteType(req, options, col)
	default:
		return "any"
	}
}
