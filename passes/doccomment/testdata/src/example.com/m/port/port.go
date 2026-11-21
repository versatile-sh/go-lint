// Copyright header before the package clause is allowed.

// Package port is the contract.
package port

import "errors"

// ErrNotFound is returned when the record does not exist.
var ErrNotFound = errors.New("not found")

var ErrUndocumented = errors.New("x") // want `exported ErrUndocumented has no doc comment` `trailing comment`

// Codes are the wire codes.
const (
	CodeA = "a"
	CodeB = "b"
)

const (
	// CodeC is documented on its spec.
	CodeC = "c"
	CodeD = "d" // want `exported CodeD has no doc comment` `trailing comment`
)

// limits are private. // want `doc comment on an unexported group`
const (
	low  = 1
	high = 2
)

// Store keeps records.
type Store interface {
	// Get returns one record.
	Get(id string) (Record, error)
	Put(r Record) error // want `trailing comment`
}

// Record is one stored row.
type Record struct {
	// ID is the key.
	ID   string
	Name string // want `trailing comment`
}

// Long explains too much. // want `doc comment has 3 lines, the limit is 2`
// It goes on.
// And on.
func Long() {}

func Missing() {} // want `exported Missing has no doc comment` `inline comment`

// helper is documented but unexported. // want `doc comment on unexported helper`
func helper() int {
	// inside a body // want `inline comment`
	return low + high
}

type private struct{}

// Exported on an unexported type is not exported. // want `doc comment on unexported Method`
func (private) Method() {}

// Method on an exported type is documented.
func (Record) Method() {}

//go:generate echo ok
func directive() {}

/* block comment */ // want `inline comment`

func use() { _ = helper(); directive() }

//nolint:unused // reason
var _ = use
