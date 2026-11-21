package core

import (
	"testing"

	"example.com/m/internal/fixture"
	"example.com/m/store"
)

func TestCore(t *testing.T) { _, _ = fixture.X, store.New }
