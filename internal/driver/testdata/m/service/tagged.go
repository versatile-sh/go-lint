//go:build integration

package service

import "time"

// Tagged reads the clock behind a build tag.
func Tagged() time.Time { return time.Now() }
