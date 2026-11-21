package layout_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/layout"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), layout.Analyzer, "example.com/m/...")
}
