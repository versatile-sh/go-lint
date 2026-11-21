// Package errstring forbids deciding on an error by its message text.
package errstring

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer reports err.Error() compared with ==, !=, switch or a strings matcher in non-test code.
var Analyzer = &analysis.Analyzer{
	Name: "errstring",
	Doc:  "forbid matching on err.Error() text; compare with a sentinel through errors.Is or errors.As",
	Run:  run,
}

const msg = "error matched by its message text: use a sentinel with errors.Is or errors.As"

var matchers = map[string]bool{"Contains": true, "HasPrefix": true, "HasSuffix": true, "EqualFold": true, "Index": true, "Compare": true}

func run(pass *analysis.Pass) (any, error) {
	errType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	isErrorText := func(e ast.Expr) bool {
		call, ok := ast.Unparen(e).(*ast.CallExpr)
		if !ok || len(call.Args) != 0 {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Error" {
			return false
		}
		t := pass.TypesInfo.TypeOf(sel.X)
		return t != nil && types.Implements(t, errType)
	}
	for f, test := range module.Files(pass) {
		if test {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if (x.Op == token.EQL || x.Op == token.NEQ) && (isErrorText(x.X) || isErrorText(x.Y)) {
					pass.Reportf(x.Pos(), msg)
				}
			case *ast.SwitchStmt:
				if x.Tag != nil && isErrorText(x.Tag) {
					pass.Reportf(x.Tag.Pos(), msg)
				}
			case *ast.CallExpr:
				fn, ok := typeutil.Callee(pass.TypesInfo, x).(*types.Func)
				if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "strings" || !matchers[fn.Name()] {
					return true
				}
				for _, a := range x.Args {
					if isErrorText(a) {
						pass.Reportf(x.Pos(), msg)
						break
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
