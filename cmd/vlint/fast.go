package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/versatile-sh/go-lint/golangci"
	"golang.org/x/tools/imports"
)

const reviveVersion = "v1.17.1"

func fastTier(args []string) int {
	fs := flag.NewFlagSet("fast", flag.ExitOnError)
	fix := fs.Bool("fix", false, "rewrite misformatted files instead of listing them")
	all := fs.Bool("all", false, "report every revive finding of the package, not only those on lines changed since HEAD")
	_ = fs.Parse(args)
	var files, dirs []string
	for _, f := range fs.Args() {
		if strings.HasSuffix(f, ".go") && !strings.Contains("/"+filepath.ToSlash(f), "/testdata/") {
			files = append(files, f)
			if d := filepath.Dir(f); !slices.Contains(dirs, d) {
				dirs = append(dirs, d)
			}
		}
	}
	if len(files) == 0 {
		return 0
	}
	code := 0
	for _, f := range files {
		src, err := os.ReadFile(f) //nolint:gosec // a staged file the caller named
		if err != nil {
			fmt.Fprintln(os.Stderr, "vlint:", err)
			return 1
		}
		out, err := imports.Process(f, src, &imports.Options{Comments: true, TabIndent: true, TabWidth: 8, FormatOnly: true})
		if err != nil {
			fmt.Fprintln(os.Stderr, "vlint:", err)
			return 1
		}
		if bytes.Equal(src, out) {
			continue
		}
		if *fix {
			if err := os.WriteFile(f, out, 0o644); err != nil { //nolint:gosec // rewrites the staged file in place
				fmt.Fprintln(os.Stderr, "vlint:", err)
				return 1
			}
			continue
		}
		fmt.Println(f + ": needs goimports")
		code = 3
	}
	if c := runRevive(dirs, *all); c != 0 {
		code = c
	}
	return code
}

func runRevive(dirs []string, all bool) int {
	top, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "vlint: not in a git repository")
		return 1
	}
	cfg := filepath.Join(strings.TrimSpace(string(top)), golangci.ReviveFile)
	if _, err := os.Stat(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "vlint: %s missing: run vlint golangci\n", golangci.ReviveFile)
		return 1
	}
	args := append([]string{"-config", cfg, "-formatter", "unix"}, dirs...)
	name, pre := "revive", []string(nil)
	if _, err := exec.LookPath(name); err != nil {
		name, pre = "go", []string{"run", "github.com/mgechev/revive@" + reviveVersion}
	}
	cmd := exec.CommandContext(context.Background(), name, append(pre, args...)...) //nolint:gosec // fixed binary, the directories are the caller's
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, os.Stderr
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		return 1
	}
	var keep []string
	touched := map[string][]span{}
	for _, line := range strings.Split(out.String(), "\n") {
		file, n, ok := position(line)
		if !ok {
			continue
		}
		if !all {
			if _, seen := touched[file]; !seen {
				touched[file] = addedLines(file)
			}
			if !within(touched[file], n) {
				continue
			}
		}
		keep = append(keep, line)
	}
	for _, line := range keep {
		fmt.Println(line)
	}
	if len(keep) > 0 {
		return 3
	}
	return 0
}

type span struct{ from, to int }

func position(line string) (string, int, bool) {
	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 4 {
		return "", 0, false
	}
	n, err := strconv.Atoi(parts[1])
	return parts[0], n, err == nil
}

func within(spans []span, n int) bool {
	if spans == nil {
		return true
	}
	return slices.ContainsFunc(spans, func(s span) bool { return n >= s.from && n <= s.to })
}

var hunk = regexp.MustCompile(`^@@ -\S+ \+(\d+)(?:,(\d+))? @@`)

func addedLines(file string) []span {
	tracked := exec.CommandContext(context.Background(), "git", "ls-files", "--error-unmatch", "--", file) //nolint:gosec // fixed binary, the path is a file the caller named
	if err := tracked.Run(); err != nil {
		return nil
	}
	diff := exec.CommandContext(context.Background(), "git", "diff", "-U0", "HEAD", "--", file) //nolint:gosec // fixed binary, the path is a file the caller named
	out, err := diff.Output()
	if err != nil {
		return nil
	}
	spans := []span{}
	for _, line := range strings.Split(string(out), "\n") {
		m := hunk.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		from, _ := strconv.Atoi(m[1])
		count := 1
		if m[2] != "" {
			count, _ = strconv.Atoi(m[2])
		}
		if count > 0 {
			spans = append(spans, span{from, from + count - 1})
		}
	}
	return spans
}
