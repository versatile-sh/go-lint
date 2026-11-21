package store

import "os"

func DSN() string {
	return os.Getenv("DSN") // want `os.Getenv: read configuration once in the composition root`
}

func Lookup() (string, bool) {
	return os.LookupEnv("DSN") // want `os.LookupEnv: read configuration once in the composition root`
}

var environ = os.Environ // want `os.Environ: read configuration once in the composition root`

func Args() []string {
	_ = environ
	return os.Args
}
