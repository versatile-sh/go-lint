package driver_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/versatile-sh/go-lint/internal/driver"
	"github.com/versatile-sh/go-lint/passes/clockseam"
	"golang.org/x/tools/go/analysis"
)

func run(t *testing.T, opts driver.Options, patterns ...string) []driver.Finding {
	t.Helper()
	t.Chdir(filepath.Join("testdata", "m"))
	got, _, err := driver.Run([]*analysis.Analyzer{clockseam.Analyzer}, patterns, opts)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func lines(fs []driver.Finding) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = filepath.Base(f.File) + ":" + f.Analyzer
	}
	return out
}

func TestRunDeduplicatesTestVariants(t *testing.T) {
	got := run(t, driver.Options{Tests: true}, "./...")
	if len(got) != 1 || lines(got)[0] != "service.go:clockseam" {
		t.Fatalf("got %v, want one service.go finding", lines(got))
	}
}

func TestRunPassesTagsToLoading(t *testing.T) {
	got := run(t, driver.Options{Tags: "integration", Tests: true}, "./...")
	if len(got) != 2 || lines(got)[1] != "tagged.go:clockseam" {
		t.Fatalf("got %v, want service.go and tagged.go", lines(got))
	}
}

func TestRunFailsWhenNothingLoads(t *testing.T) {
	t.Chdir(filepath.Join("testdata", "m"))
	_, _, err := driver.Run([]*analysis.Analyzer{clockseam.Analyzer}, []string{"./missing/..."}, driver.Options{})
	if !errors.Is(err, driver.ErrLoad) {
		t.Fatalf("err = %v, want ErrLoad", err)
	}
}
