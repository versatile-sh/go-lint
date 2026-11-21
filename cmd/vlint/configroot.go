package main

import (
	"os"
	"path/filepath"
)

func findConfig(start, top, name string) string {
	for d := start; ; d = filepath.Dir(d) {
		p := filepath.Join(d, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		if d == top || d == filepath.Dir(d) {
			return filepath.Join(top, name)
		}
	}
}
