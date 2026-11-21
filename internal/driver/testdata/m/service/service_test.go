package service

import "testing"

func TestStamp(t *testing.T) {
	if Stamp().IsZero() {
		t.Fatal("zero")
	}
}
