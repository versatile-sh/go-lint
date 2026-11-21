package core

import (
	"example.com/m/core/rule"
	"example.com/m/internal/fixture" // want `core imports internal/fixture: core/ may import only core/, port/, schema/`
	"example.com/m/port"
	"example.com/m/store" // want `core imports store: core/ may import only core/, port/, schema/`
)

var _ port.Store

const Name = "core"

var _, _, _ = rule.OK, fixture.X, store.New
