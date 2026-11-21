package adrref_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/versatile-sh/go-lint/passes/adrref"
)

func TestProtoComments(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "scan.proto"))
	if err != nil {
		t.Fatal(err)
	}
	got := adrref.ProtoComments(src)
	want := []adrref.Position{{Line: 1, Col: 1}, {Line: 3, Col: 1}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestProtoWalksTheModuleOnly(t *testing.T) {
	root := filepath.Join("testdata", "protomod")
	got, err := adrref.Proto(root)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "proto", "v1", "order.proto")
	lines := []int{3, 6, 9}
	if len(got) != len(lines) {
		t.Fatalf("got %v, want lines %v of %s", got, lines, file)
	}
	for i, p := range got {
		if p.File != file || p.Line != lines[i] {
			t.Fatalf("got %v, want lines %v of %s", got, lines, file)
		}
	}
}
