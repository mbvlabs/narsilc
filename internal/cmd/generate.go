package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"
	"sync"

	"github.com/mbvlabs/narsilc/internal/codegen/golang"
	"github.com/mbvlabs/narsilc/internal/compiler"
	"github.com/mbvlabs/narsilc/internal/config"
	"github.com/mbvlabs/narsilc/internal/debug"
	"github.com/mbvlabs/narsilc/internal/ext"
	"github.com/mbvlabs/narsilc/internal/multierr"
	"github.com/mbvlabs/narsilc/internal/opts"
	"github.com/mbvlabs/narsilc/internal/plugin"
	"github.com/mbvlabs/narsilc/internal/sqlcdebug"
)

var debugDumpCatalog = sqlcdebug.New("dumpcatalog")

const errMessageNoVersion = `The configuration file must have a version number.
Set the version to 1 or 2 at the top of narsilc.json:

{
  "version": "1"
  ...
}
`

const errMessageUnknownVersion = `The configuration file has an invalid version number.
The supported version can only be "1" or "2".
`

const errMessageNoPackages = `No packages are configured`

func printFileErr(stderr io.Writer, dir string, fileErr *multierr.FileError) {
	filename, err := filepath.Rel(dir, fileErr.Filename)
	if err != nil {
		filename = fileErr.Filename
	}
	fmt.Fprintf(stderr, "%s:%d:%d: %s\n", filename, fileErr.Line, fileErr.Column, fileErr.Err)
}

func readConfig(stderr io.Writer, dir, filename string) (string, *config.Config, error) {
	configPath := ""
	if filename != "" {
		configPath = filepath.Join(dir, filename)
	} else {
		tomlPath := filepath.Join(dir, "andurel.toml")
		if _, err := os.Stat(tomlPath); err == nil {
			return readAndurelToml(stderr, tomlPath)
		}

		lockPath := filepath.Join(dir, "andurel.lock")
		if lockData, err := os.ReadFile(lockPath); err == nil {
			err := config.MissingAndurelManifestError(lockData)
			fmt.Fprintln(stderr, err)
			return "", nil, err
		} else if !os.IsNotExist(err) {
			return "", nil, err
		}

		var yamlMissing, jsonMissing, ymlMissing bool
		yamlPath := filepath.Join(dir, "narsilc.yaml")
		ymlPath := filepath.Join(dir, "narsilc.yml")
		jsonPath := filepath.Join(dir, "narsilc.json")

		if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
			yamlMissing = true
		}
		if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
			jsonMissing = true
		}

		if _, err := os.Stat(ymlPath); os.IsNotExist(err) {
			ymlMissing = true
		}

		if yamlMissing && ymlMissing && jsonMissing {
			fmt.Fprintln(stderr, "error parsing configuration files. andurel.toml or narsilc.(yaml|yml|json): file does not exist")
			return "", nil, errors.New("config file missing")
		}

		if (!yamlMissing || !ymlMissing) && !jsonMissing {
			fmt.Fprintln(stderr, "error: both narsilc.json and narsilc.(yaml|yml) files present")
			return "", nil, errors.New("narsilc.json and narsilc.(yaml|yml) present")
		}

		if jsonMissing {
			if yamlMissing {
				configPath = ymlPath
			} else {
				configPath = yamlPath
			}
		} else {
			configPath = jsonPath
		}
	}

	base := filepath.Base(configPath)
	if config.IsAndurelToml(base) {
		return readAndurelToml(stderr, configPath)
	}
	if config.IsAndurelLock(base) {
		err := fmt.Errorf("pass andurel.toml; andurel.lock is digests-only")
		fmt.Fprintln(stderr, err)
		return "", nil, err
	}

	file, err := os.Open(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "error parsing %s: file does not exist\n", base)
		return "", nil, err
	}
	defer file.Close()

	conf, err := config.ParseConfig(file)
	if err != nil {
		switch err {
		case config.ErrMissingVersion:
			fmt.Fprint(stderr, errMessageNoVersion)
		case config.ErrUnknownVersion:
			fmt.Fprint(stderr, errMessageUnknownVersion)
		case config.ErrNoPackages:
			fmt.Fprint(stderr, errMessageNoPackages)
		}
		fmt.Fprintf(stderr, "error parsing %s: %s\n", base, err)
		return "", nil, err
	}

	return configPath, &conf, nil
}

func readAndurelToml(stderr io.Writer, path string) (string, *config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "error parsing andurel.toml: file does not exist\n")
		return "", nil, err
	}
	conf, err := config.FromAndurelToml(data)
	if err != nil {
		fmt.Fprintf(stderr, "error parsing andurel.toml: %s\n", err)
		return "", nil, err
	}
	return path, &conf, nil
}

func Generate(ctx context.Context, dir, filename string, o *Options) (map[string]string, error) {
	e := o.Env
	stderr := o.Stderr

	configPath, conf, err := o.ReadConfig(dir, filename)
	if err != nil {
		return nil, err
	}

	base := filepath.Base(configPath)
	if err := config.Validate(conf); err != nil {
		fmt.Fprintf(stderr, "error validating %s: %s\n", base, err)
		return nil, err
	}

	if err := e.Validate(conf); err != nil {
		fmt.Fprintf(stderr, "error validating %s: %s\n", base, err)
		return nil, err
	}

	g := &generator{
		dir:    dir,
		output: map[string]string{},
	}

	if err := processQuerySets(ctx, g, conf, dir, o); err != nil {
		return nil, err
	}

	return g.output, nil
}

type generator struct {
	m      sync.Mutex
	dir    string
	output map[string]string
}

func (g *generator) Pairs(ctx context.Context, conf *config.Config) []OutputPair {
	var pairs []OutputPair
	for _, sql := range conf.SQL {
		if sql.Gen.Go != nil {
			pairs = append(pairs, OutputPair{
				SQL: sql,
				Gen: config.SQLGen{Go: sql.Gen.Go},
			})
		}
	}
	return pairs
}

func (g *generator) ProcessResult(ctx context.Context, combo config.CombinedSettings, sql OutputPair, result *compiler.Result) error {
	out, resp, err := codegen(ctx, combo, sql, result)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for _, file := range resp.Files {
		files[file.Name] = string(file.Contents)
	}
	g.m.Lock()

	absout := filepath.Join(g.dir, out)

	// When the Go codegen is configured to emit the models file into a
	// separate package directory, route that file to its own absolute path.
	// This is the only file allowed to live outside of `out`.
	var (
		modelsFileName string
		modelsAbsout   string
		modelsAbsfile  string
	)
	if sql.Gen.Go != nil && sql.Gen.Go.OutputModelsPath != "" && sql.Gen.Go.ModelsEmitEnabled() {
		modelsFileName = sql.Gen.Go.OutputModelsFileName
		if modelsFileName == "" {
			modelsFileName = "models.go"
		}
		modelsAbsout = filepath.Join(g.dir, sql.Gen.Go.OutputModelsPath)
		modelsAbsfile = filepath.Join(modelsAbsout, modelsFileName)
	}

	for n, source := range files {
		if modelsFileName != "" && n == modelsFileName {
			// Models file routed to a separate package directory.
			if strings.Contains(modelsAbsfile, "..") {
				return fmt.Errorf("invalid file output path: %s", modelsAbsfile)
			}
			if !strings.HasPrefix(modelsAbsfile, modelsAbsout) {
				return fmt.Errorf("invalid file output path: %s", modelsAbsfile)
			}
			g.output[modelsAbsfile] = source
			continue
		}
		filename := filepath.Join(g.dir, out, n)
		// filepath.Join calls filepath.Clean which should remove all "..", but
		// double check to make sure
		if strings.Contains(filename, "..") {
			return fmt.Errorf("invalid file output path: %s", filename)
		}
		// The output file must be contained inside the output directory
		if !strings.HasPrefix(filename, absout) {
			return fmt.Errorf("invalid file output path: %s", filename)
		}
		g.output[filename] = source
	}
	g.m.Unlock()
	return nil
}

func parse(ctx context.Context, name, dir string, sql config.SQL, combo config.CombinedSettings, parserOpts opts.Parser, stderr io.Writer) (*compiler.Result, bool) {
	defer trace.StartRegion(ctx, "parse").End()
	var copts []compiler.Option
	if parserOpts.Experiment.CoreAnalyzer {
		copts = append(copts, compiler.WithCoreAnalysis())
	}
	c, err := compiler.NewCompiler(sql, combo, parserOpts, copts...)
	defer func() {
		if c != nil {
			c.Close(ctx)
		}
	}()
	if err != nil {
		fmt.Fprintf(stderr, "error creating compiler: %s\n", err)
		return nil, true
	}
	if err := c.ParseCatalog(sql.Schema); err != nil {
		fmt.Fprintf(stderr, "# package %s\n", name)
		if parserErr, ok := err.(*multierr.Error); ok {
			for _, fileErr := range parserErr.Errs() {
				printFileErr(stderr, dir, fileErr)
			}
		} else {
			fmt.Fprintf(stderr, "error parsing schema: %s\n", err)
		}
		return nil, true
	}
	if debugDumpCatalog.Value() == "1" {
		debug.Dump(c.Catalog())
	}
	if err := c.ParseQueries(sql.Queries, parserOpts); err != nil {
		fmt.Fprintf(stderr, "# package %s\n", name)
		if parserErr, ok := err.(*multierr.Error); ok {
			for _, fileErr := range parserErr.Errs() {
				printFileErr(stderr, dir, fileErr)
			}
		} else {
			fmt.Fprintf(stderr, "error parsing queries: %s\n", err)
		}
		return nil, true
	}
	return c.Result(), false
}

func codegen(ctx context.Context, combo config.CombinedSettings, sql OutputPair, result *compiler.Result) (string, *plugin.GenerateResponse, error) {
	defer trace.StartRegion(ctx, "codegen").End()
	if sql.Gen.Go == nil {
		return "", nil, fmt.Errorf("missing Go codegen configuration")
	}
	req := codeGenRequest(result, combo)
	opts, err := json.Marshal(sql.Gen.Go)
	if err != nil {
		return "", nil, fmt.Errorf("opts marshal failed: %w", err)
	}
	req.PluginOptions = opts

	if combo.Global.Overrides.Go != nil {
		opts, err := json.Marshal(combo.Global.Overrides.Go)
		if err != nil {
			return "", nil, fmt.Errorf("opts marshal failed: %w", err)
		}
		req.GlobalOptions = opts
	}

	client := plugin.NewCodegenServiceClient(ext.HandleFunc(golang.Generate))
	resp, err := client.Generate(ctx, req)
	return combo.Go.Out, resp, err
}
