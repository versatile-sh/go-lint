package app

import "os"

func Load() string { return os.Getenv("DSN") }
