// Package clockseam forbids reading the wall clock outside the composition root, tests, test kits and a clock adapter.
package clockseam

import (
	"path"
	"strings"

	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports time.Now, time.Since and time.Until outside app/, cmd/, main packages, tests and clock packages.
// A clock package's folder or name starts or ends with "clock" (clock, clockwork, worldclock), module roots included.
var Analyzer = &analysis.Analyzer{
	Name: "clockseam",
	Doc:  "forbid wall-clock reads outside the composition root; inject a clock port instead",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	_, rel, ok := module.Of(pass)
	if ok && (module.CompositionRoot(pass, rel) || module.TestKit(rel) || clockName(path.Base(rel)) || clockName(pass.Pkg.Name())) {
		return nil, nil
	}
	module.ReportUses(pass, "time", map[string]bool{"Now": true, "Since": true, "Until": true}, "read time through the injected clock port")
	return nil, nil
}

func clockName(name string) bool {
	return strings.HasPrefix(name, "clock") || strings.HasSuffix(name, "clock")
}
