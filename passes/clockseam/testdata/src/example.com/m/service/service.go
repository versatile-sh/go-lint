package service

import "time"

func Stamp() time.Time {
	return time.Now() // want `time.Now: read time through the injected clock port`
}

var now = time.Now // want `time.Now: read time through the injected clock port`

func Age(t time.Time) time.Duration {
	_ = now
	return time.Since(t) // want `time.Since: read time through the injected clock port`
}

func Fixed() time.Time {
	return time.Unix(0, 0)
}
