package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/versatile-sh/go-lint/golangci"
	"github.com/versatile-sh/go-lint/internal/baseline"
	"github.com/versatile-sh/go-lint/internal/changed"
	"github.com/versatile-sh/go-lint/internal/driver"
)

func changedTier(args []string) int {
	fs := flag.NewFlagSet("changed", flag.ExitOnError)
	baseRef := fs.String("base", "origin/main", "ref whose merge base with HEAD bounds the change")
	tags := fs.String("tags", "", "comma-separated build tags, passed to package loading and golangci-lint")
	base := fs.String("baseline", "", "analyzer baseline file, relative to the repository root; only new findings fail")
	bin := fs.String("golangci-lint", "golangci-lint", "golangci-lint binary")
	list := fs.Bool("list", false, "print the changed packages and run nothing")
	skipGolangci := fs.Bool("skip-golangci", false, "run the analyzers only")
	_ = fs.Parse(args)
	res, err := changed.Resolve(context.Background(), ".", *baseRef)
	if err != nil {
		fmt.Fprintln(os.Stderr, "vlint:", err)
		return 1
	}
	if len(res.Sets) == 0 {
		fmt.Fprintf(os.Stderr, "vlint changed: no Go package changed since %s\n", *baseRef)
		return 0
	}
	if *list {
		for _, s := range res.Sets {
			for _, p := range s.Patterns {
				fmt.Println(filepath.Join(s.Dir, p))
			}
		}
		return 0
	}
	baselinePath := *base
	if baselinePath != "" && !filepath.IsAbs(baselinePath) {
		baselinePath = filepath.Join(res.Top, baselinePath)
	}
	code := 0
	for _, s := range res.Sets {
		if err := os.Chdir(s.Dir); err != nil {
			fmt.Fprintln(os.Stderr, "vlint:", err)
			return 1
		}
		if c := changedAnalyzers(s.Patterns, *tags, baselinePath); c != 0 {
			code = max(code, c)
		}
		if !*skipGolangci {
			if c := changedGolangci(*bin, res, s, *baseRef, *tags); c != 0 {
				code = max(code, c)
			}
		}
	}
	return code
}

func changedAnalyzers(patterns []string, tags, baselinePath string) int {
	findings, err := analyze(analyzers, patterns, driver.Options{Tags: tags, Tests: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, "vlint:", err)
		return 1
	}
	if baselinePath != "" {
		findings, _, err = baseline.Compare(baselinePath, findings)
		if err != nil {
			fmt.Fprintln(os.Stderr, "vlint:", err)
			return 1
		}
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		return 3
	}
	return 0
}

func changedGolangci(bin string, res changed.Result, s changed.Set, baseRef, tags string) int {
	cfg := findConfig(s.Dir, res.Top, golangci.FastConfigFile)
	if _, err := os.Stat(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "vlint: %s missing: run vlint golangci\n", golangci.FastConfigFile)
		return 1
	}
	args := []string{"run", "-c", cfg, "--new-from-merge-base", baseRef}
	if tags != "" {
		args = append(args, "--build-tags", tags)
	}
	cmd := exec.CommandContext(context.Background(), bin, append(args, s.Patterns...)...) //nolint:gosec // the binary and patterns are the caller's
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "vlint changed: golangci-lint:", strings.TrimSpace(err.Error()))
		return 3
	}
	return 0
}
