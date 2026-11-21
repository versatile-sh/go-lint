package doccomment_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/doccomment"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), doccomment.Analyzer, "example.com/m/...")
}
