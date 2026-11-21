package store

import (
	"os"
	"testing"
)

func TestDSN(t *testing.T) { _ = os.Getenv("DSN") }
