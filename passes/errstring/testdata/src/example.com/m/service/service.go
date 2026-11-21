package service

import (
	"errors"
	"strings"
)

var ErrGone = errors.New("gone")

type named struct{}

func (named) Error() string { return "named" }

func Classify(err error) string {
	if err.Error() == "gone" { // want `error matched by its message text`
		return "a"
	}
	if "gone" != err.Error() { // want `error matched by its message text`
		return "b"
	}
	if strings.Contains(err.Error(), "timeout") { // want `error matched by its message text`
		return "c"
	}
	switch err.Error() { // want `error matched by its message text`
	case "x":
		return "d"
	}
	if (named{}).Error() == "named" { // want `error matched by its message text`
		return "e"
	}
	if errors.Is(err, ErrGone) {
		return "f"
	}
	return strings.ToUpper(err.Error())
}
