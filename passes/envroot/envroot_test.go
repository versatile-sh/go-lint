package envroot_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/envroot"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), envroot.Analyzer, "example.com/m/...")
}
