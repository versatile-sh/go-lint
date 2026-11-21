package server

import (
	"example.com/m/cmd/srv/flags" // want `server imports cmd/srv/flags: nothing imports cmd/`
	"example.com/m/service"
)

var _ = service.X

var _ = flags.Name
