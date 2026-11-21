package adrref_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/adrref"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), adrref.Analyzer, "example.com/m/...")
}
