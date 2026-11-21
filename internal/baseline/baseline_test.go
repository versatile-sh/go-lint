package baseline_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/versatile-sh/go-lint/internal/baseline"
	"github.com/versatile-sh/go-lint/internal/driver"
)

const src = "package core\n\nfunc A() { _ = 1 }\n\nfunc B() { _ = 1 }\n"

func module(t *testing.T, content string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "core", "core.go")
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, file
}

func finding(dir, file string, line int) driver.Finding {
	return driver.Finding{Module: dir, File: file, Line: line, Col: 1, Analyzer: "doccomment", Message: "exported A has no doc comment"}
}

func TestKeyIsModuleRelativeAndLineFree(t *testing.T) {
	dir, file := module(t, src)
	key := baseline.NewKeyer().Key(finding(dir, file, 3))
	if !strings.HasPrefix(key, "core/core.go\tdoccomment\t") || strings.Contains(key, ":3") {
		t.Fatalf("key = %q", key)
	}
}

func TestCompareSurvivesLinesInsertedAbove(t *testing.T) {
	dir, file := module(t, src)
	path := filepath.Join(dir, "baseline.txt")
	if err := baseline.Write(path, []driver.Finding{finding(dir, file, 3)}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("// x\n// y\n"+src), 0o600); err != nil {
		t.Fatal(err)
	}
	fresh, fixed, err := baseline.Compare(path, []driver.Finding{finding(dir, file, 5)})
	if err != nil || len(fresh) != 0 || len(fixed) != 0 {
		t.Fatalf("fresh %v fixed %v err %v, want none", fresh, fixed, err)
	}
}

func TestCompareReportsNewAndFixed(t *testing.T) {
	dir, file := module(t, src)
	path := filepath.Join(dir, "baseline.txt")
	if err := baseline.Write(path, []driver.Finding{finding(dir, file, 3)}); err != nil {
		t.Fatal(err)
	}
	fresh, fixed, err := baseline.Compare(path, []driver.Finding{finding(dir, file, 5)})
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh) != 1 || fresh[0].Line != 5 || len(fixed) != 1 {
		t.Fatalf("fresh %v fixed %v, want line 5 new and one fixed", fresh, fixed)
	}
}

func TestCompareCountsRepeatedKeys(t *testing.T) {
	dir, file := module(t, "package core\n\nfunc A() {}\nfunc A() {}\nfunc A() {}\n")
	path := filepath.Join(dir, "baseline.txt")
	two := []driver.Finding{finding(dir, file, 3), finding(dir, file, 4)}
	if err := baseline.Write(path, two); err != nil {
		t.Fatal(err)
	}
	fresh, _, err := baseline.Compare(path, append(two, finding(dir, file, 5)))
	if err != nil || len(fresh) != 1 {
		t.Fatalf("fresh %v err %v, want the third occurrence", fresh, err)
	}
}

func TestWriteIsSorted(t *testing.T) {
	dir, file := module(t, src)
	path := filepath.Join(dir, "baseline.txt")
	f := finding(dir, file, 5)
	f.Analyzer = "adrref"
	if err := baseline.Write(path, []driver.Finding{finding(dir, file, 3), f}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for line := range strings.Lines(string(data)) {
		if !strings.HasPrefix(line, "#") {
			keys = append(keys, line)
		}
	}
	if len(keys) != 2 || !slices.IsSorted(keys) {
		t.Fatalf("keys %q, want two sorted", keys)
	}
}

func TestCompareFailsWithoutBaseline(t *testing.T) {
	if _, _, err := baseline.Compare(filepath.Join(t.TempDir(), "none.txt"), nil); err == nil {
		t.Fatal("want an error for a missing baseline")
	}
}
