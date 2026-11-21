package changed_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/versatile-sh/go-lint/internal/changed"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "go.mod", "module m\n")
	write(t, dir, "a/a.go", "package a\n")
	write(t, dir, "b/b.go", "package b\n")
	write(t, dir, "nested/go.mod", "module n\n")
	write(t, dir, "nested/n.go", "package n\n")
	write(t, dir, "a/testdata/x/x.go", "package x\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-q", "-m", "base")
	run(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	run(t, dir, "checkout", "-q", "-b", "feature")
	return dir
}

func patterns(t *testing.T, dir string) map[string][]string {
	t.Helper()
	res, err := changed.Resolve(context.Background(), dir, "origin/main")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for _, s := range res.Sets {
		rel, _ := filepath.Rel(res.Top, s.Dir)
		out[rel] = s.Patterns
	}
	return out
}

func TestResolveMapsFilesToPackagesPerModule(t *testing.T) {
	dir := repo(t)
	write(t, dir, "a/a.go", "package a\n\nvar X = 1\n")
	write(t, dir, "a/testdata/x/x.go", "package x\n\nvar Y = 1\n")
	write(t, dir, "nested/n.go", "package n\n\nvar Z = 1\n")
	write(t, dir, "c/c.go", "package c\n")
	run(t, dir, "add", "a/a.go")
	want := map[string][]string{".": {"./a", "./c"}, "nested": {"."}}
	if got := patterns(t, dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveLintsTheWholeModuleWhenGoModChanges(t *testing.T) {
	dir := repo(t)
	write(t, dir, "go.mod", "module m\n\ngo 1.27\n")
	want := map[string][]string{".": {"./..."}}
	if got := patterns(t, dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestResolveIsEmptyWithoutChanges(t *testing.T) {
	if got := patterns(t, repo(t)); len(got) != 0 {
		t.Fatalf("got %v, want nothing", got)
	}
}
