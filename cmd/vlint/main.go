// Command vlint runs the Versatile Go analyzers, generates the shared golangci-lint config and applies the safe fixers.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/versatile-sh/go-lint/golangci"
	"github.com/versatile-sh/go-lint/passes/adrref"
	"github.com/versatile-sh/go-lint/passes/clockseam"
	"github.com/versatile-sh/go-lint/passes/doccomment"
	"github.com/versatile-sh/go-lint/passes/envroot"
	"github.com/versatile-sh/go-lint/passes/errstring"
	"github.com/versatile-sh/go-lint/passes/layout"
	"github.com/versatile-sh/go-lint/passes/noskipenv"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "golangci":
			os.Exit(config(os.Args[2:]))
		case "fix":
			os.Exit(fix(os.Args[2:]))
		}
	}
	multichecker.Main(adrref.Analyzer, clockseam.Analyzer, doccomment.Analyzer, envroot.Analyzer,
		errstring.Analyzer, layout.Analyzer, noskipenv.Analyzer)
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
