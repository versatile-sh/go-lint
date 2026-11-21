// Package envroot forbids reading environment variables outside the composition root, tests and test kits.
package envroot

import (
	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports os.Getenv, os.LookupEnv and os.Environ used outside app/, cmd/, main packages, tests and test kits.
var Analyzer = &analysis.Analyzer{
	Name: "envroot",
	Doc:  "forbid environment reads outside the composition root; pass the value in from app/",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	_, rel, ok := module.Of(pass)
	if ok && (module.CompositionRoot(pass, rel) || module.TestKit(rel)) {
		return nil, nil
	}
	module.ReportUses(pass, "os", map[string]bool{"Getenv": true, "LookupEnv": true, "Environ": true}, "read configuration once in the composition root")
	return nil, nil
}
