package sqlpath

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mbvlabs/narsilc/internal/migrations"
)

// Return a list of SQL files in the listed paths.
//
// Only includes files ending in .sql. Omits hidden files, directories, and
// down migrations.

// If a path contains *, ?, [, or ], treat the path as a pattern and expand it
// filepath.Glob.
func Glob(patterns []string) ([]string, error) {
	paths, err := expand(patterns)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, path := range paths {
		f, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("path error: %w", err)
		}
		if f.IsDir() {
			listing, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}
			for _, f := range listing {
				files = append(files, filepath.Join(path, f.Name()))
			}
		} else {
			files = append(files, filepath.Clean(path))
		}
	}
	return sqlFiles(files), nil
}

// GlobRecursive is like Glob, but descends into the subdirectories of every
// directory path. exclude lists directories to skip, relative to each
// directory path, so generated output is not read back as query input.
func GlobRecursive(patterns []string, exclude []string) ([]string, error) {
	paths, err := expand(patterns)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, path := range paths {
		f, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("path error: %w", err)
		}
		if !f.IsDir() {
			files = append(files, filepath.Clean(path))
			continue
		}

		root := filepath.Clean(path)
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if excludedDir(root, path, exclude) {
					return fs.SkipDir
				}
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return sqlFiles(files), nil
}

func expand(patterns []string) ([]string, error) {
	var paths []string
	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, "*?[]") {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, err
			}
			paths = append(paths, matches...)
		} else {
			paths = append(paths, pattern)
		}
	}
	return paths, nil
}

func excludedDir(root, path string, exclude []string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	for _, dir := range exclude {
		dir = filepath.Clean(dir)
		if rel == dir || filepath.Clean(path) == dir {
			return true
		}
	}
	return false
}

func sqlFiles(files []string) []string {
	var sqlFiles []string
	for _, file := range files {
		if !strings.HasSuffix(file, ".sql") {
			continue
		}
		if strings.HasPrefix(filepath.Base(file), ".") {
			continue
		}
		if migrations.IsDown(filepath.Base(file)) {
			continue
		}
		sqlFiles = append(sqlFiles, file)
	}
	return sqlFiles
}
