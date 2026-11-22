// Package changed resolves the Go packages touched since a git merge base, grouped by module.
package changed

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Set is the changed packages of one module: Dir is the absolute module root and Patterns are ./-relative package
// patterns, or the single pattern ./... when go.mod, go.sum or a lint config changed.
type Set struct {
	Dir      string
	Patterns []string
}

// Result is what changed since the merge base: the repository root, the merge base commit and the sets by module.
type Result struct {
	Top       string
	MergeBase string
	Sets      []Set
}

// Resolve lists the changed packages of the repository containing dir against base (a ref such as origin/main).
// It counts committed, staged, unstaged and untracked files.
func Resolve(ctx context.Context, dir, base string) (Result, error) {
	top, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Result{}, err
	}
	top = strings.TrimSpace(top)
	mb, err := git(ctx, top, "merge-base", "HEAD", base)
	if err != nil {
		return Result{}, fmt.Errorf("merge base with %s: %w (fetch it first)", base, err)
	}
	mb = strings.TrimSpace(mb)
	diff, err := git(ctx, top, "diff", "--name-only", "-z", mb)
	if err != nil {
		return Result{}, err
	}
	untracked, err := git(ctx, top, "ls-files", "-o", "--exclude-standard", "-z")
	if err != nil {
		return Result{}, err
	}
	files := splitZ(diff + untracked)
	all, err := git(ctx, top, "ls-files", "-z", "--", ":(glob)go.mod", ":(glob)**/go.mod")
	if err != nil {
		return Result{}, err
	}
	return Result{Top: top, MergeBase: mb, Sets: group(top, files, splitZ(all))}, nil
}

func group(top string, files, gomods []string) []Set {
	modules := map[string]bool{}
	for _, g := range gomods {
		if skipped(g) {
			continue
		}
		modules[filepath.Join(top, path.Dir(g))] = true
	}
	everything := false
	whole := map[string]bool{}
	pkgs := map[string]map[string]bool{}
	for _, f := range files {
		base := path.Base(f)
		switch {
		case base == "go.mod" || base == "go.sum" || base == "go.work":
			whole[moduleOf(top, path.Dir(f), modules)] = true
		case strings.HasPrefix(base, ".golangci") || base == ".revive.toml" || strings.HasPrefix(f, "lint/") || strings.HasPrefix(f, ".config/lint/"):
			everything = true
		case strings.HasSuffix(f, ".go") && !skipped(f):
			dir := filepath.Join(top, path.Dir(f))
			if !hasGo(dir) {
				continue
			}
			mod := moduleOf(top, path.Dir(f), modules)
			if mod == "" {
				continue
			}
			rel, _ := filepath.Rel(mod, dir)
			if pkgs[mod] == nil {
				pkgs[mod] = map[string]bool{}
			}
			pkgs[mod]["./"+filepath.ToSlash(rel)] = true
		}
	}
	if everything {
		for m := range modules {
			whole[m] = true
		}
	}
	var sets []Set
	for m := range modules {
		switch {
		case whole[m]:
			sets = append(sets, Set{Dir: m, Patterns: []string{"./..."}})
		case len(pkgs[m]) > 0:
			var ps []string
			for p := range pkgs[m] {
				ps = append(ps, strings.Replace(p, "/.", "", 1))
			}
			sort.Strings(ps)
			sets = append(sets, Set{Dir: m, Patterns: ps})
		}
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].Dir < sets[j].Dir })
	return sets
}

func moduleOf(top, dir string, modules map[string]bool) string {
	for d := filepath.Join(top, dir); ; d = filepath.Dir(d) {
		if modules[d] {
			return d
		}
		if d == top || d == filepath.Dir(d) {
			return ""
		}
	}
}

func skipped(f string) bool {
	for _, seg := range strings.Split(path.Dir(f), "/") {
		if seg == "." {
			continue
		}
		if seg == "testdata" || seg == "vendor" || seg == "node_modules" || strings.HasPrefix(seg, ".") || strings.HasPrefix(seg, "_") {
			return true
		}
	}
	return false
}

func hasGo(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}

func splitZ(s string) []string {
	var out []string
	for _, f := range strings.Split(s, "\x00") {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // fixed binary, the arguments are built here
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
