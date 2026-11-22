package golangci_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/versatile-sh/go-lint/golangci"
)

const overlay = `exclusions:
  rules:
    - path: 'legacy/'
      linters: [gosec]
depguard:
  rules:
    core-no-sql:
      files: ["**/core/**"]
      deny:
        - pkg: database/sql
          desc: core does no I/O
`

func TestGenerateMergesOverlay(t *testing.T) {
	out, err := golangci.Generate([]byte(overlay))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{"go-lint. DO NOT EDIT.", "path: 'legacy/'", "core-no-sql:", "one-logger:", "checked-types:"} {
		if !strings.Contains(text, want) {
			t.Errorf("generated config lacks %q", want)
		}
	}
	if strings.Contains(text, "os\\.(Getenv") || strings.Contains(text, "time\\.(Now") {
		t.Error("env and clock belong to the vlint analyzers, not forbidigo")
	}
}

func TestGenerateAddsMergeBaseOnlyWhenSet(t *testing.T) {
	plain, err := golangci.Generate([]byte(overlay))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "new-from-merge-base") {
		t.Fatal("an overlay without issues must not set a merge base")
	}
	gated, err := golangci.Generate([]byte(overlay + "issues:\n  new-from-merge-base: origin/main\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(string(plain), "  max-same-issues: 0\n", "  max-same-issues: 0\n  new-from-merge-base: origin/main\n", 1)
	if string(gated) != want {
		t.Fatalf("gated config differs from the plain one by more than the merge base:\n%s", gated)
	}
}

func TestGenerateRejectsUnknownOverlayKeys(t *testing.T) {
	if _, err := golangci.Generate([]byte("linters:\n  enable: [wsl]\n")); err == nil {
		t.Fatal("an overlay that enables linters must be refused")
	}
}

func TestGenerateRejectsRedefinedDepguardRule(t *testing.T) {
	if _, err := golangci.Generate([]byte("depguard:\n  rules:\n    one-logger: {deny: []}\n")); err == nil {
		t.Fatal("an overlay that redefines a shared depguard rule must be refused")
	}
}

func TestCheckDetectsDrift(t *testing.T) {
	dir := t.TempDir()
	if err := golangci.Check(dir); !errors.Is(err, golangci.ErrDrift) {
		t.Fatalf("missing config: got %v, want ErrDrift", err)
	}
	if err := golangci.Write(dir); err != nil {
		t.Fatal(err)
	}
	if err := golangci.Check(dir); err != nil {
		t.Fatalf("fresh config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, golangci.ConfigFile), []byte("version: \"2\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := golangci.Check(dir); !errors.Is(err, golangci.ErrDrift) {
		t.Fatalf("hand edit: got %v, want ErrDrift", err)
	}
	if err := golangci.Write(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lint"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, golangci.OverlayFile), []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := golangci.Check(dir); !errors.Is(err, golangci.ErrDrift) {
		t.Fatalf("overlay change: got %v, want ErrDrift", err)
	}
}

func TestGeneratedConfigVerifies(t *testing.T) {
	bin, err := exec.LookPath("golangci-lint")
	if err != nil {
		t.Fatal("golangci-lint is required on PATH for this test")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lint"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, golangci.OverlayFile), []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := golangci.Write(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{golangci.ConfigFile, golangci.FastConfigFile} {
		if out, err := exec.Command(bin, "config", "verify", "-c", filepath.Join(dir, name)).CombinedOutput(); err != nil {
			t.Fatalf("golangci-lint config verify %s: %v\n%s", name, err, out)
		}
	}
}

func TestGenerateFastDropsOnlyTheHeavyLinters(t *testing.T) {
	out, err := golangci.GenerateFast([]byte("exclusions:\n  rules:\n    - path: 'x/'\n      linters: [gosec]\n"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, gone := range []string{"- gosec", "- contextcheck", "- sloglint", "- testifylint", "sloglint:", "gosec:", "path: 'x/'"} {
		if strings.Contains(text, gone) {
			t.Errorf("fast config still has %q", gone)
		}
	}
	for _, kept := range []string{"- staticcheck", "- govet", "staticcheck:", "one-logger:"} {
		if !strings.Contains(text, kept) {
			t.Errorf("fast config lost %q", kept)
		}
	}
}

func TestWriteToPlacesFilesAndReadsOverlayInOut(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(".config", "lint")
	if err := golangci.CheckTo(dir, out); !errors.Is(err, golangci.ErrDrift) {
		t.Fatalf("missing config: got %v, want ErrDrift", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, out), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, out, golangci.OverlayName), []byte(overlay), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := golangci.WriteTo(dir, out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{golangci.ConfigFile, golangci.FastConfigFile, golangci.ReviveFile} {
		if _, err := os.Stat(filepath.Join(dir, out, name)); err != nil {
			t.Fatalf("%s not written in out: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Fatalf("%s leaked to the root", name)
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, out, golangci.ConfigFile))
	if err != nil || !strings.Contains(string(got), "legacy/") {
		t.Fatalf("overlay in out not merged: %v", err)
	}
	if !strings.Contains(string(got), "Add repository rules to .config/lint/golangci.local.yml and regenerate.") {
		t.Fatalf("header does not name the overlay in out:\n%s", got[:200])
	}
	if err := golangci.CheckTo(dir, out); err != nil {
		t.Fatalf("fresh config: %v", err)
	}
}
