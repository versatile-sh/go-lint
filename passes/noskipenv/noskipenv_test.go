package noskipenv_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/noskipenv"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), noskipenv.Analyzer, "example.com/m/...")
}
