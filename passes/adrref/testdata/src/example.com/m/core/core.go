// Package core does things.
package core

// Run follows ADR-0005. // want `design-record reference in code`
func Run() string {
	return "see docs/adr/0003-x.md" // want `design-record reference in code`
}

// Plan cites adr 12 and PL-ADR-001. // want `design-record reference in code`
func Plan() string { return "address padre radr/ adrenaline" }

// Note mentions records in general.
func Note() string { return "ADR" }
