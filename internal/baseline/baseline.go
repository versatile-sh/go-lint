// Package baseline records known findings keyed without line numbers and compares a run against them.
package baseline

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/versatile-sh/go-lint/internal/driver"
)

const header = "# vlint baseline: module-relative file, analyzer, hash of the reported source line, message.\n" +
	"# Regenerate with vlint -baseline <file> -write. Lines are sorted; a repeated line counts one finding each.\n"

// Keyer turns findings into stable keys, caching the source files it reads.
type Keyer struct {
	files map[string][]string
}

// NewKeyer returns an empty Keyer.
func NewKeyer() *Keyer { return &Keyer{files: map[string][]string{}} }

// Key returns file, analyzer, source-line hash and message joined by tabs; edits on other lines leave it unchanged.
func (k *Keyer) Key(f driver.Finding) string {
	name := f.File
	if f.Module != "" {
		if rel, err := filepath.Rel(f.Module, f.File); err == nil {
			name = rel
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(strings.Fields(k.line(f.File, f.Line)), " ")))
	return strings.Join([]string{filepath.ToSlash(name), f.Analyzer, hex.EncodeToString(sum[:4]), f.Message}, "\t")
}

func (k *Keyer) line(file string, n int) string {
	lines, ok := k.files[file]
	if !ok {
		data, _ := os.ReadFile(file) //nolint:gosec // a file the analyzers just reported on
		lines = strings.Split(string(data), "\n")
		k.files[file] = lines
	}
	if n < 1 || n > len(lines) {
		return ""
	}
	return lines[n-1]
}

// Write records the keys of findings at path, sorted.
func Write(path string, findings []driver.Finding) error {
	k := NewKeyer()
	keys := make([]string, len(findings))
	for i, f := range findings {
		keys[i] = k.Key(f)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	buf.WriteString(header)
	for _, key := range keys {
		buf.WriteString(key + "\n")
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // a committed baseline is world-readable
		return fmt.Errorf("write baseline: %w", err)
	}
	return nil
}

// Compare returns the findings beyond the baseline at path and the baseline keys that no longer occur.
func Compare(path string, findings []driver.Finding) ([]driver.Finding, []string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // the baseline the caller named
	if err != nil {
		return nil, nil, fmt.Errorf("read baseline: %w", err)
	}
	known := map[string]int{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		if line := sc.Text(); line != "" && !strings.HasPrefix(line, "#") {
			known[line]++
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, fmt.Errorf("read baseline: %w", err)
	}
	k := NewKeyer()
	var fresh []driver.Finding
	for _, f := range findings {
		key := k.Key(f)
		if known[key] > 0 {
			known[key]--
			continue
		}
		fresh = append(fresh, f)
	}
	var fixed []string
	for key, n := range known {
		for range n {
			fixed = append(fixed, key)
		}
	}
	sort.Strings(fixed)
	return fresh, fixed, nil
}
