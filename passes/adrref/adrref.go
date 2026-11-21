// Package adrref forbids references to design records (record ids and record-folder paths) in Go source.
package adrref

import (
	"go/ast"
	"go/token"
	"regexp"

	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports design-record ids and record-folder paths in comments and string literals, tests included.
var Analyzer = &analysis.Analyzer{
	Name: "adrref",
	Doc:  "forbid design-record ids and record-folder paths in code: design records live outside the repository",
	Run:  run,
}

var pattern = regexp.MustCompile(`(?i)(\badr[- ]?[0-9]+|(^|[^a-z0-9_])adrs?/)`)

const msg = "design-record reference in code: state the behaviour, keep the record outside the repository"

func run(pass *analysis.Pass) (any, error) {
	for f := range module.Files(pass) {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if pattern.MatchString(c.Text) {
					pass.Reportf(c.Pos(), msg)
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && pattern.MatchString(lit.Value) {
				pass.Reportf(lit.Pos(), msg)
			}
			return true
		})
	}
	return nil, nil
}
