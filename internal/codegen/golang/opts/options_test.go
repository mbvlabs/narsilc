package opts

import "testing"

func TestValidateRowMapping(t *testing.T) {
	limit := int32(1)
	base := func() *Options {
		return &Options{
			QueryParameterLimit: &limit,
		}
	}

	t.Run("empty", func(t *testing.T) {
		if err := ValidateOpts(base()); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("andurel requires pgx/v5", func(t *testing.T) {
		opts := base()
		opts.RowMapping = RowMappingAndurel
		if err := ValidateOpts(opts); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unknown", func(t *testing.T) {
		opts := base()
		opts.RowMapping = "bun"
		if err := ValidateOpts(opts); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("emit_interface", func(t *testing.T) {
		opts := base()
		opts.RowMapping = RowMappingAndurel
		opts.EmitInterface = true
		if err := ValidateOpts(opts); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("pgx/v5", func(t *testing.T) {
		opts := base()
		opts.RowMapping = RowMappingAndurel
		opts.SqlPackage = SQLPackagePGXV5
		if err := ValidateOpts(opts); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("pgx/v4", func(t *testing.T) {
		opts := base()
		opts.RowMapping = RowMappingAndurel
		opts.SqlPackage = SQLPackagePGXV4
		if err := ValidateOpts(opts); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("database/sql", func(t *testing.T) {
		opts := base()
		opts.RowMapping = RowMappingAndurel
		opts.SqlPackage = SQLPackageStandard
		if err := ValidateOpts(opts); err == nil {
			t.Fatal("expected error")
		}
	})
}
