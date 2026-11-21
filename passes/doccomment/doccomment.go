// Package doccomment enforces the comment rule: a doc comment of at most two lines on exported symbols, nothing else.
package doccomment

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"github.com/versatile-sh/go-lint/internal/module"
	"golang.org/x/tools/go/analysis"
)

// Analyzer reports missing, long or misplaced doc comments and every inline, trailing or block comment.
var Analyzer = &analysis.Analyzer{
	Name: "doccomment",
	Doc:  "require a doc comment of at most two lines on exported symbols and forbid every other comment",
	Run:  run,
}

const maxLines = 2

var directive = regexp.MustCompile(`^(//(go:|line |export |extern |[a-z0-9]+:\S)|// ?\+build )`)

var exampleOutput = regexp.MustCompile(`^//\s*(Unordered output|Output):`)

type checker struct {
	pass *analysis.Pass
	test bool
	seen map[*ast.CommentGroup]bool
}

func run(pass *analysis.Pass) (any, error) {
	for f, test := range module.Files(pass) {
		c := &checker{pass: pass, test: test, seen: map[*ast.CommentGroup]bool{}}
		c.file(f)
	}
	return nil, nil
}

func (c *checker) file(f *ast.File) {
	if f.Doc != nil {
		c.length(f.Doc)
	}
	for _, d := range f.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			c.symbol(x.Name, funcExported(x), x.Doc)
		case *ast.GenDecl:
			c.gen(x)
		}
	}
	for _, cg := range f.Comments {
		if c.seen[cg] || cg.End() < f.Package || allDirectives(cg) || exampleOutput.MatchString(cg.List[0].Text) {
			continue
		}
		c.pass.Reportf(cg.Pos(), "inline comment: delete it, or fold real knowledge into the doc comment of an exported symbol")
	}
}

func (c *checker) gen(d *ast.GenDecl) {
	switch d.Tok {
	case token.IMPORT:
		for _, s := range d.Specs {
			if is, ok := s.(*ast.ImportSpec); ok && is.Path.Value == `"C"` {
				c.seen[d.Doc] = true
				c.seen[is.Doc] = true
			}
		}
	case token.TYPE:
		for _, s := range d.Specs {
			ts := s.(*ast.TypeSpec)
			doc := ts.Doc
			if doc == nil && !d.Lparen.IsValid() {
				doc = d.Doc
			}
			c.symbol(ts.Name, ts.Name.IsExported(), doc)
			c.members(ts.Type)
		}
		if d.Lparen.IsValid() && d.Doc != nil {
			c.group(d)
		}
	case token.CONST, token.VAR:
		grouped := d.Lparen.IsValid()
		for _, s := range d.Specs {
			vs := s.(*ast.ValueSpec)
			exported := false
			for _, n := range vs.Names {
				exported = exported || n.IsExported()
			}
			doc := vs.Doc
			if doc == nil && !grouped {
				doc = d.Doc
			}
			if doc != nil || !grouped || d.Doc == nil || !exported {
				c.symbol(vs.Names[0], exported, doc)
			}
			c.trailing(vs.Comment)
		}
		if grouped && d.Doc != nil {
			c.group(d)
		}
	}
}

func (c *checker) group(d *ast.GenDecl) {
	for _, s := range d.Specs {
		switch x := s.(type) {
		case *ast.ValueSpec:
			for _, n := range x.Names {
				if n.IsExported() {
					c.length(d.Doc)
					return
				}
			}
		case *ast.TypeSpec:
			if x.Name.IsExported() {
				c.length(d.Doc)
				return
			}
		}
	}
	c.seen[d.Doc] = true
	c.pass.Reportf(d.Doc.Pos(), "doc comment on an unexported group: delete it")
}

func (c *checker) symbol(name *ast.Ident, exported bool, doc *ast.CommentGroup) {
	if doc != nil && allDirectives(doc) {
		c.seen[doc] = true
		doc = nil
	}
	switch {
	case doc == nil && exported && !c.test && name.Name != "_":
		c.pass.Reportf(name.Pos(), "exported %s has no doc comment", name.Name)
	case doc == nil:
	case exported:
		c.length(doc)
	default:
		c.seen[doc] = true
		c.pass.Reportf(doc.Pos(), "doc comment on unexported %s: delete it, the name and the test carry the intent", name.Name)
	}
}

func (c *checker) members(t ast.Expr) {
	ast.Inspect(t, func(n ast.Node) bool {
		var fields *ast.FieldList
		switch x := n.(type) {
		case *ast.StructType:
			fields = x.Fields
		case *ast.InterfaceType:
			fields = x.Methods
		}
		if fields == nil {
			return true
		}
		for _, fl := range fields.List {
			if fl.Doc != nil {
				c.length(fl.Doc)
			}
			c.trailing(fl.Comment)
		}
		return true
	})
}

func (c *checker) trailing(cg *ast.CommentGroup) {
	if cg != nil && !allDirectives(cg) {
		c.seen[cg] = true
		c.pass.Reportf(cg.Pos(), "trailing comment: delete it, or move it into the doc comment")
	}
}

func (c *checker) length(doc *ast.CommentGroup) {
	c.seen[doc] = true
	lines := 0
	for _, cm := range doc.List {
		if !directive.MatchString(cm.Text) {
			lines += strings.Count(strings.TrimRight(cm.Text, "\n"), "\n") + 1
		}
	}
	if lines > maxLines {
		c.pass.Reportf(doc.Pos(), "doc comment has %d lines, the limit is %d", lines, maxLines)
	}
}

func funcExported(f *ast.FuncDecl) bool {
	if !f.Name.IsExported() {
		return false
	}
	if f.Recv == nil || len(f.Recv.List) == 0 {
		return true
	}
	t := f.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	switch x := t.(type) {
	case *ast.IndexExpr:
		t = x.X
	case *ast.IndexListExpr:
		t = x.X
	}
	id, ok := t.(*ast.Ident)
	return ok && id.IsExported()
}

func allDirectives(cg *ast.CommentGroup) bool {
	for _, cm := range cg.List {
		if !directive.MatchString(cm.Text) {
			return false
		}
	}
	return true
}
