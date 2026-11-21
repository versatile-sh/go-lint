package port

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	// a test comment // want `inline comment`
	_ = t
}

// fake is unexported. // want `doc comment on unexported fake`
type fake struct{}

func TestExportedWithoutDoc(t *testing.T) { _ = fake{} }

func ExampleLong() {
	fmt.Println("ok")
	// Output:
	// ok
}

func ExampleMissing() {
	fmt.Println("b")
	fmt.Println("a")
	// Unordered output:
	// a
	// b
}
