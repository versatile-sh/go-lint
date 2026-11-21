package k3stest

import "os"

func Kubeconfig() string { return os.Getenv("KUBECONFIG") }
