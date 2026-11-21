// Package clockseam forbids reading the wall clock outside the composition root, tests, test kits and a clock adapter.
package clockseam

import (
	"path"
	"strings"

	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports time.Now, time.Since and time.Until used outside app/, cmd/, main packages, tests and clock* packages.
var Analyzer = &analysis.Analyzer{
	Name: "clockseam",
	Doc:  "forbid wall-clock reads outside the composition root; inject a clock port instead",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	_, rel, ok := module.Of(pass)
	if ok && (module.CompositionRoot(pass, rel) || module.TestKit(rel) || strings.HasPrefix(path.Base(rel), "clock")) {
		return nil, nil
	}
	module.ReportUses(pass, "time", map[string]bool{"Now": true, "Since": true, "Until": true}, "read time through the injected clock port")
	return nil, nil
}
