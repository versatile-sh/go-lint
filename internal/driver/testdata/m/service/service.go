// Package service is a fixture.
package service

import "time"

// Stamp reads the clock.
func Stamp() time.Time { return time.Now() }
