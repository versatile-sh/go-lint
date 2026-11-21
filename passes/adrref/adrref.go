// Package adrref forbids references to design records (record ids and record-folder paths) in Go source and .proto comments.
package adrref

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

// Position is a 1-based line and column in a .proto file.
type Position struct {
	File      string
	Line, Col int
}

// Message is the text adrref reports, in Go files and .proto files alike.
const Message = msg

// Proto walks the module rooted at dir and returns each .proto comment that holds a design-record reference.
func Proto(dir string) ([]Position, error) {
	var out []Position
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != dir && skipDir(p, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) != ".proto" {
			return nil
		}
		src, err := os.ReadFile(p) //nolint:gosec // a .proto file of the analyzed module
		if err != nil {
			return err
		}
		for _, c := range ProtoComments(src) {
			out = append(out, Position{File: p, Line: c.Line, Col: c.Col})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan .proto files: %w", err)
	}
	return out, nil
}

func skipDir(p, name string) bool {
	if name == "testdata" || name == "vendor" || name == "node_modules" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return true
	}
	_, err := os.Stat(filepath.Join(p, "go.mod"))
	return err == nil
}

// ProtoComments returns the start of each line or block comment in proto source that holds a design-record reference.
func ProtoComments(src []byte) []Position {
	var out []Position
	line, col := 1, 1
	for i := 0; i < len(src); {
		start := Position{Line: line, Col: col}
		end := i + 1
		switch {
		case src[i] == '"' || src[i] == '\'':
			end = stringEnd(src, i)
		case bytes.HasPrefix(src[i:], []byte("//")):
			end = len(src)
			if n := bytes.IndexByte(src[i:], '\n'); n >= 0 {
				end = i + n
			}
		case bytes.HasPrefix(src[i:], []byte("/*")):
			end = len(src)
			if n := bytes.Index(src[i+2:], []byte("*/")); n >= 0 {
				end = i + 2 + n + 2
			}
		}
		if src[i] == '/' && end > i+1 && pattern.Match(src[i:end]) {
			out = append(out, start)
		}
		for _, b := range src[i:end] {
			if b == '\n' {
				line, col = line+1, 1
			} else {
				col++
			}
		}
		i = end
	}
	return out
}

func stringEnd(src []byte, i int) int {
	quote := src[i]
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case quote, '\n':
			return j + 1
		}
	}
	return len(src)
}
