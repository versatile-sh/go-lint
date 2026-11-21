// Package module places an analyzed package inside its Go module: the module path, the module-relative directory
// and the roles the layout standard derives from them.
package module

import (
	"go/ast"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
)

// Module is a Go module root found by walking up from a package directory.
type Module struct {
	Path string
	Dir  string
}

var cache sync.Map

// Of returns the module of the analyzed package and the package directory relative to the module root ("" at the root).
func Of(pass *analysis.Pass) (Module, string, bool) {
	if len(pass.Files) == 0 {
		return Module{}, "", false
	}
	dir := filepath.Dir(pass.Fset.Position(pass.Files[0].Package).Filename)
	m, ok := find(dir)
	if !ok {
		return Module{}, "", false
	}
	rel, err := filepath.Rel(m.Dir, dir)
	if err != nil {
		return Module{}, "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		rel = ""
	}
	return m, rel, true
}

func find(dir string) (Module, bool) {
	if v, ok := cache.Load(dir); ok {
		m := v.(Module)
		return m, m.Dir != ""
	}
	var m Module
	if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil { //nolint:gosec // go.mod of the analyzed tree
		m = Module{Path: modfile.ModulePath(data), Dir: dir}
	} else if parent := filepath.Dir(dir); parent != dir {
		m, _ = find(parent)
	}
	cache.Store(dir, m)
	return m, m.Dir != ""
}

// Rel returns the module-relative path of an import path, or false when the import is outside the module.
func (m Module) Rel(importPath string) (string, bool) {
	if importPath == m.Path {
		return "", true
	}
	rel, ok := strings.CutPrefix(importPath, m.Path+"/")
	return rel, ok
}

// Top returns the first element of a module-relative path.
func Top(rel string) string {
	top, _, _ := strings.Cut(rel, "/")
	return top
}

// CompositionRoot reports whether the package is a main package or lives under app/ or cmd/.
func CompositionRoot(pass *analysis.Pass, rel string) bool {
	if pass.Pkg.Name() == "main" {
		return true
	}
	top := Top(rel)
	return top == "app" || top == "cmd"
}

// TestKit reports whether the package is a test-support package, named with a "test" suffix.
func TestKit(rel string) bool {
	return rel != "" && strings.HasSuffix(path.Base(rel), "test")
}

// IsTest reports whether file is a _test.go file.
func IsTest(pass *analysis.Pass, file *ast.File) bool {
	return strings.HasSuffix(pass.Fset.Position(file.Package).Filename, "_test.go")
}

// Files returns the analyzed files that are not generated, with whether each is a test file.
func Files(pass *analysis.Pass) map[*ast.File]bool {
	out := make(map[*ast.File]bool, len(pass.Files))
	for _, f := range pass.Files {
		if !ast.IsGenerated(f) {
			out[f] = IsTest(pass, f)
		}
	}
	return out
}

// ReportUses reports every non-test use, call or function value, of the named functions of package pkg.
func ReportUses(pass *analysis.Pass, pkg string, names map[string]bool, msg string) {
	for f, test := range Files(pass) {
		if test {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			fn, ok := pass.TypesInfo.Uses[id].(*types.Func)
			if ok && fn.Pkg() != nil && fn.Pkg().Path() == pkg && names[fn.Name()] && fn.Signature().Recv() == nil {
				pass.Reportf(id.Pos(), "%s.%s: %s", pkg, fn.Name(), msg)
			}
			return true
		})
	}
}
