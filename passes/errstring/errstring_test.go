package errstring_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/errstring"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), errstring.Analyzer, "example.com/m/...")
}
