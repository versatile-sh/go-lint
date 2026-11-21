package service

import (
	"example.com/m/app" // want `service imports app: only cmd/ and main packages import the composition root app/`
	"example.com/m/core"
	"example.com/m/store"
)

var _, _ = store.New, app.Run

var _ = core.Name

const X = 1
