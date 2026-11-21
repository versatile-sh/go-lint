// Command vlint runs the Versatile Go analyzers, generates the shared golangci-lint config and applies the safe fixers.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/versatile-sh/go-lint/golangci"
	"github.com/versatile-sh/go-lint/internal/baseline"
	"github.com/versatile-sh/go-lint/internal/driver"
	"github.com/versatile-sh/go-lint/passes/adrref"
	"github.com/versatile-sh/go-lint/passes/clockseam"
	"github.com/versatile-sh/go-lint/passes/doccomment"
	"github.com/versatile-sh/go-lint/passes/envroot"
	"github.com/versatile-sh/go-lint/passes/errstring"
	"github.com/versatile-sh/go-lint/passes/layout"
	"github.com/versatile-sh/go-lint/passes/noskipenv"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/unitchecker"
)

var analyzers = []*analysis.Analyzer{adrref.Analyzer, clockseam.Analyzer, doccomment.Analyzer, envroot.Analyzer,
	errstring.Analyzer, layout.Analyzer, noskipenv.Analyzer}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "golangci":
			os.Exit(config(os.Args[2:]))
		case "fix":
			os.Exit(fix(os.Args[2:]))
		}
	}
	if len(os.Args) == 2 && strings.HasSuffix(os.Args[1], ".cfg") {
		unitchecker.Run(os.Args[1], analyzers)
	}
	os.Exit(lint(os.Args[1:]))
}

func lint(args []string) int {
	fs := flag.NewFlagSet("vlint", flag.ExitOnError)
	tags := fs.String("tags", "", "comma-separated build tags, passed to package loading")
	tests := fs.Bool("test", true, "analyze test files too")
	base := fs.String("baseline", "", "fail only on findings missing from this baseline file; list the ones fixed")
	write := fs.Bool("write", false, "with -baseline, record the current findings in the baseline file")
	only := map[*analysis.Analyzer]*bool{}
	for _, a := range analyzers {
		only[a] = fs.Bool(a.Name, false, "run only the analyzers set true, or all but those set false: "+a.Doc)
	}
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: vlint [flags] packages | vlint golangci [-check] | vlint fix [packages]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)
	if fs.NArg() == 0 || fs.Arg(0) == "help" {
		fs.Usage()
		if fs.NArg() == 0 {
			return 1
		}
		return 0
	}
	run := selected(fs, only)
	findings, err := analyze(run, fs.Args(), driver.Options{Tags: *tags, Tests: *tests})
	if err != nil {
		fmt.Fprintln(os.Stderr, "vlint:", err)
		return 1
	}
	switch {
	case *base == "" && *write:
		fmt.Fprintln(os.Stderr, "vlint: -write needs -baseline <file>")
		return 1
	case *write:
		if err := baseline.Write(*base, findings); err != nil {
			fmt.Fprintln(os.Stderr, "vlint:", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "vlint: %d findings recorded in %s\n", len(findings), *base)
		return 0
	case *base != "":
		fresh, fixed, err := baseline.Compare(*base, findings)
		if err != nil {
			fmt.Fprintln(os.Stderr, "vlint:", err)
			return 1
		}
		for _, key := range fixed {
			fmt.Fprintln(os.Stderr, "fixed:", strings.ReplaceAll(key, "\t", " "))
		}
		if len(fixed) > 0 {
			fmt.Fprintf(os.Stderr, "vlint: %d baseline entries no longer occur: rerun with -write to shrink %s\n", len(fixed), *base)
		}
		findings = fresh
		fmt.Fprintf(os.Stderr, "vlint: %d new findings against %s\n", len(fresh), *base)
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		return 3
	}
	return 0
}

func analyze(run []*analysis.Analyzer, patterns []string, opts driver.Options) ([]driver.Finding, error) {
	findings, modules, err := driver.Run(run, patterns, opts)
	if err != nil || !slices.Contains(run, adrref.Analyzer) {
		return findings, err
	}
	for _, m := range modules {
		ps, err := adrref.Proto(m)
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			findings = append(findings, driver.Finding{Module: m, File: p.File, Line: p.Line, Col: p.Col, Analyzer: adrref.Analyzer.Name, Message: adrref.Message})
		}
	}
	driver.Sort(findings)
	return findings, nil
}

func selected(fs *flag.FlagSet, only map[*analysis.Analyzer]*bool) []*analysis.Analyzer {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	anyTrue := false
	for _, a := range analyzers {
		anyTrue = anyTrue || *only[a]
	}
	var out []*analysis.Analyzer
	for _, a := range analyzers {
		if (anyTrue && *only[a]) || (!anyTrue && !set[a.Name]) {
			out = append(out, a)
		}
	}
	return out
}

func config(args []string) int {
	fs := flag.NewFlagSet("golangci", flag.ExitOnError)
	check := fs.Bool("check", false, "fail when the committed .golangci.yml differs from the generated one")
	dir := fs.String("dir", ".", "repository root")
	_ = fs.Parse(args)
	run := golangci.Write
	if *check {
		run = golangci.Check
	}
	if err := run(*dir); err != nil {
		fmt.Fprintln(os.Stderr, "vlint:", err)
		return 1
	}
	return 0
}

func fix(args []string) int {
	if len(args) == 0 {
		args = []string{"./..."}
	}
	steps := [][]string{
		append([]string{"run", "--fix", "--enable-only", "errorlint"}, args...),
		append([]string{"fmt"}, args...),
	}
	code := 0
	for _, step := range steps {
		cmd := exec.CommandContext(context.Background(), "golangci-lint", step...) //nolint:gosec // fixed binary, arguments are the caller's package patterns
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "vlint fix:", err)
			code = 1
		}
	}
	return code
}
