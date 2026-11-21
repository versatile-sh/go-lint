package service

import (
	"errors"
	"testing"
)

func TestMessage(t *testing.T) {
	if errors.New("x").Error() != "x" {
		t.Fatal("message")
	}
}
