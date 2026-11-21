package core

import "testing"

func TestRun(t *testing.T) {
	if Run() == "ADR-7" { // want `design-record reference in code`
		t.Fatal()
	}
}
