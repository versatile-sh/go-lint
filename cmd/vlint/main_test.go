package main

import (
	"path/filepath"
	"testing"

	"github.com/versatile-sh/go-lint/internal/driver"
	"github.com/versatile-sh/go-lint/passes/adrref"
	"github.com/versatile-sh/go-lint/passes/layout"
	"golang.org/x/tools/go/analysis"
)

func TestAnalyzeAddsProtoFindingsWithAdrref(t *testing.T) {
	t.Chdir(filepath.Join("..", "..", "passes", "adrref", "testdata", "protomod"))
	got, err := analyze([]*analysis.Analyzer{adrref.Analyzer}, []string{"./..."}, driver.Options{Tests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || filepath.Base(got[0].File) != "order.proto" || got[0].Analyzer != "adrref" {
		t.Fatalf("got %v, want three order.proto findings", got)
	}
	got, err = analyze([]*analysis.Analyzer{layout.Analyzer}, []string{"./..."}, driver.Options{Tests: true})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v err %v, want no proto scan without adrref", got, err)
	}
}
