// Package driver loads Go packages, runs analyzers over them and returns one deduplicated finding per position.
package driver

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// Finding is one analyzer report: the absolute file, its module root, the position, the analyzer and the message.
type Finding struct {
	Module, File      string
	Line, Col         int
	Analyzer, Message string
}

// Options selects the build tags and whether test files are analyzed.
type Options struct {
	Tags  string
	Tests bool
}

// ErrLoad is returned when a package fails to load or type-check, so no result is mistaken for a clean tree.
var ErrLoad = errors.New("packages failed to load")

// Run analyzes the packages matched by patterns and returns the sorted findings and the sorted module roots seen.
func Run(analyzers []*analysis.Analyzer, patterns []string, opts Options) ([]Finding, []string, error) {
	cfg := &packages.Config{Mode: packages.LoadSyntax | packages.NeedModule, Tests: opts.Tests}
	if opts.Tags != "" {
		cfg.BuildFlags = []string{"-tags=" + opts.Tags}
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrLoad, err)
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("%w: %s matched no packages", ErrLoad, strings.Join(patterns, " "))
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, nil, fmt.Errorf("%w: %d errors", ErrLoad, n)
	}
	graph, err := checker.Analyze(analyzers, pkgs, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("analyze: %w", err)
	}
	seen := map[Finding]bool{}
	modules := map[string]bool{}
	var out []Finding
	var errs []error
	for _, act := range graph.Roots {
		if act.Err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", act, act.Err))
			continue
		}
		mod := ""
		if act.Package.Module != nil {
			mod = act.Package.Module.Dir
			modules[mod] = true
		}
		for _, d := range act.Diagnostics {
			p := act.Package.Fset.Position(d.Pos)
			f := Finding{Module: mod, File: p.Filename, Line: p.Line, Col: p.Column, Analyzer: act.Analyzer.Name, Message: d.Message}
			if !seen[f] {
				seen[f] = true
				out = append(out, f)
			}
		}
	}
	if len(errs) > 0 {
		return nil, nil, errors.Join(errs...)
	}
	Sort(out)
	roots := make([]string, 0, len(modules))
	for m := range modules {
		roots = append(roots, m)
	}
	sort.Strings(roots)
	return out, roots, nil
}

// Sort orders findings by file, position, analyzer and message.
func Sort(fs []Finding) {
	sort.Slice(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Col != b.Col {
			return a.Col < b.Col
		}
		if a.Analyzer != b.Analyzer {
			return a.Analyzer < b.Analyzer
		}
		return a.Message < b.Message
	})
}

// String formats the finding as file:line:col: message, with the file relative to the working directory when shorter.
func (f Finding) String() string {
	name := f.File
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, f.File); err == nil && !strings.HasPrefix(rel, "..") {
			name = rel
		}
	}
	return fmt.Sprintf("%s:%d:%d: %s", name, f.Line, f.Col, f.Message)
}
