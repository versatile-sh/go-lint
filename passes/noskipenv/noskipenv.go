// Package noskipenv forbids tests that skip silently, for example when their environment is missing.
package noskipenv

import (
	"go/ast"
	"go/constant"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer reports every t.Skip not guarded by testing.Short, a *_SKIP_* opt-out variable or runtime.GOOS/GOARCH.
// The re-exec idiom (a test function named *HelperProcess) is exempt.
var Analyzer = &analysis.Analyzer{
	Name: "noskipenv",
	Doc:  "forbid silent test skips: fail, or skip only under testing.Short, an explicit SKIP opt-out variable or a platform check",
	Run:  run,
}

var optOut = regexp.MustCompile(`(^|_)SKIP(_|$)`)

const msg = "silent test skip: fail instead, or guard the skip with testing.Short, a *_SKIP_* opt-out variable or runtime.GOOS"

type envRead struct {
	any    bool
	optOut bool
}

func run(pass *analysis.Pass) (any, error) {
	vars := map[types.Object]envRead{}
	readOf := func(e ast.Node) envRead {
		var r envRead
		ast.Inspect(e, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				fn, ok := typeutil.Callee(pass.TypesInfo, x).(*types.Func)
				if !ok || fn.Pkg() == nil {
					return true
				}
				if fn.Pkg().Path() == "os" && (fn.Name() == "Getenv" || fn.Name() == "LookupEnv") && len(x.Args) == 1 {
					r.any = true
					if v := pass.TypesInfo.Types[x.Args[0]].Value; v != nil && v.Kind() == constant.String && optOut.MatchString(constant.StringVal(v)) {
						r.optOut = true
					}
				}
				if fn.Pkg().Path() == "testing" && fn.Name() == "Short" {
					r.optOut = true
				}
			case *ast.Ident:
				if v, ok := vars[pass.TypesInfo.Uses[x]]; ok {
					r.any = r.any || v.any
					r.optOut = r.optOut || v.optOut
				}
			}
			return true
		})
		return r
	}
	track := func(lhs []*ast.Ident, rhs []ast.Expr) {
		var r envRead
		for _, e := range rhs {
			got := readOf(e)
			r.any, r.optOut = r.any || got.any, r.optOut || got.optOut
		}
		if !r.any {
			return
		}
		for _, id := range lhs {
			if obj := pass.TypesInfo.ObjectOf(id); obj != nil {
				vars[obj] = r
			}
		}
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				var ids []*ast.Ident
				for _, l := range x.Lhs {
					if id, ok := l.(*ast.Ident); ok {
						ids = append(ids, id)
					}
				}
				track(ids, x.Rhs)
			case *ast.ValueSpec:
				track(x.Names, x.Values)
			}
			return true
		})
	}
	allows := func(stmt *ast.IfStmt) bool {
		for _, n := range []ast.Node{stmt.Init, stmt.Cond} {
			if n != nil && (readOf(n).optOut || platform(pass, n)) {
				return true
			}
		}
		return false
	}
	for _, f := range pass.Files {
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if fd, ok := n.(*ast.FuncDecl); ok && strings.Contains(fd.Name.Name, "HelperProcess") {
				return false
			}
			if call, ok := n.(*ast.CallExpr); ok && isSkip(pass, call) && !guarded(stack, allows) {
				pass.Reportf(call.Pos(), msg)
			}
			stack = append(stack, n)
			return true
		})
	}
	return nil, nil
}

func guarded(stack []ast.Node, allows func(*ast.IfStmt) bool) bool {
	for _, n := range stack {
		if stmt, ok := n.(*ast.IfStmt); ok && allows(stmt) {
			return true
		}
	}
	return false
}

func platform(pass *analysis.Pass, n ast.Node) bool {
	found := false
	ast.Inspect(n, func(m ast.Node) bool {
		if id, ok := m.(*ast.Ident); ok {
			if v, ok := pass.TypesInfo.Uses[id].(*types.Const); ok && v.Pkg() != nil && v.Pkg().Path() == "runtime" && (v.Name() == "GOOS" || v.Name() == "GOARCH") {
				found = true
			}
		}
		return !found
	})
	return found
}

func isSkip(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Skip" && sel.Sel.Name != "Skipf" && sel.Sel.Name != "SkipNow") {
		return false
	}
	t := pass.TypesInfo.TypeOf(sel.X)
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == "testing"
}
