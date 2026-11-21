// Package layout enforces the Go folder layout: the allowed folder names and the direction of imports between layers.
package layout

import (
	"strconv"
	"strings"

	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports folders outside the port/core/service/app layout and imports that point against the layers.
var Analyzer = &analysis.Analyzer{
	Name: "layout",
	Doc:  "enforce the port/core/service/app folder layout and inward dependency direction",
	Run:  run,
}

var forbidden = set("domain", "ports", "application", "adapters", "adapter", "bootstrap", "infrastructure", "infra",
	"usecase", "usecases", "pkg", "util", "utils", "common", "helpers", "shared", "lib")

var layers = set("app", "core", "port", "service")

var allowed = map[string]map[string]bool{
	"core":   set("core", "port", "schema"),
	"port":   set("port", "schema", "api"),
	"schema": set("schema"),
	"api":    set("api", "schema"),
}

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

func run(pass *analysis.Pass) (any, error) {
	mod, rel, ok := module.Of(pass)
	if !ok {
		return nil, nil
	}
	folder(pass, rel)
	from := module.Top(rel)
	main := pass.Pkg.Name() == "main"
	for f, test := range module.Files(pass) {
		for _, spec := range f.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			target, inside := mod.Rel(path)
			if !inside {
				continue
			}
			if why := denied(rel, from, module.Top(target), test, main); why != "" {
				pass.Reportf(spec.Pos(), "%s imports %s: %s", display(rel), display(target), why)
			}
		}
	}
	return nil, nil
}

func folder(pass *analysis.Pass, rel string) {
	parts := strings.Split(rel, "/")
	bad := ""
	switch {
	case forbidden[parts[0]]:
		bad = parts[0]
	case parts[0] == "internal" && len(parts) > 1 && (forbidden[parts[1]] || layers[parts[1]]):
		bad = parts[0] + "/" + parts[1]
	}
	if bad == "" || len(pass.Files) == 0 {
		return
	}
	pass.Reportf(pass.Files[0].Name.Pos(), "folder %s/ is outside the layout: use app/, cmd/, port/, core/, service/, schema/, api/ or a flat adapter folder at the module root", bad)
}

func denied(rel, from, to string, test, main bool) string {
	switch {
	case to == "cmd" && from != "cmd":
		return "nothing imports cmd/"
	case to == "app" && from != "app" && from != "cmd" && rel != "" && !main:
		return "only cmd/ and main packages import the composition root app/"
	case test:
		return ""
	case allowed[from] != nil && !allowed[from][to]:
		return from + "/ may import only " + strings.Join(sorted(allowed[from]), ", ")
	}
	return ""
}

func sorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for _, n := range []string{"api", "core", "port", "schema"} {
		if m[n] {
			out = append(out, n+"/")
		}
	}
	return out
}

func display(rel string) string {
	if rel == "" {
		return "module root"
	}
	return rel
}
