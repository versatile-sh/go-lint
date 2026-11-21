package clockseam_test

import (
	"testing"

	"github.com/versatile-sh/go-lint/passes/clockseam"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), clockseam.Analyzer, "example.com/m/...", "example.com/worldclock")
}
