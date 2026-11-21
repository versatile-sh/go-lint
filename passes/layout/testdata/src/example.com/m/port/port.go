package port

import (
	"example.com/m/core/rule" // want `port imports core/rule: port/ may import only api/, port/, schema/`
	"example.com/m/schema"
)

type Store interface{ Get() schema.Doc }

var _ = rule.OK
