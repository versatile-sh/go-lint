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
	if out, err := exec.Command(bin, "config", "verify", "-c", filepath.Join(dir, golangci.ConfigFile)).CombinedOutput(); err != nil {
		t.Fatalf("golangci-lint config verify: %v\n%s", err, out)
	}
}
