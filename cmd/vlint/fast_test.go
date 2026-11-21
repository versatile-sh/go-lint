package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPositionParsesUnixFormat(t *testing.T) {
	file, n, ok := position("service/live.go:42:7: [var-naming] type fakeIdP should be fakeIDP")
	if !ok || file != "service/live.go" || n != 42 {
		t.Fatalf("got %q %d %v", file, n, ok)
	}
	if _, _, ok := position("not a finding"); ok {
		t.Fatal("a line without a position must be skipped")
	}
}

func TestWithinCountsEveryLineOfANewFile(t *testing.T) {
	if !within(nil, 10) {
		t.Fatal("a nil span list means the whole file is new")
	}
	spans := []span{{3, 4}, {10, 10}}
	for n, want := range map[int]bool{2: false, 3: true, 4: true, 5: false, 10: true, 11: false} {
		if got := within(spans, n); got != want {
			t.Errorf("line %d: got %v, want %v", n, got, want)
		}
	}
}

func TestFindConfigPrefersTheNearestModuleRoot(t *testing.T) {
	top := t.TempDir()
	nested := filepath.Join(top, "server", "pkg")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatal(err)
	}
	if got, want := findConfig(nested, top, ".revive.toml"), filepath.Join(top, ".revive.toml"); got != want {
		t.Fatalf("no config anywhere: got %s, want %s", got, want)
	}
	for _, dir := range []string{top, filepath.Join(top, "server")} {
		if err := os.WriteFile(filepath.Join(dir, ".revive.toml"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := findConfig(nested, top, ".revive.toml"), filepath.Join(top, "server", ".revive.toml"); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
