// Command vlint runs the Versatile Go analyzers, generates the shared golangci-lint config and applies the safe fixers.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/versatile-sh/go-lint/golangci"
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
	findings, err := driver.Run(run, fs.Args(), driver.Options{Tags: *tags, Tests: *tests})
	if err != nil {
		fmt.Fprintln(os.Stderr, "vlint:", err)
		return 1
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		return 3
	}
	return 0
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
