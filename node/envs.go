package node

import "os"

func getHostname() string {
	// Kubernetes first
	if v, ok := os.LookupEnv("HOSTNAME"); ok {
		return v
	}
	if v, err := os.Hostname(); err == nil {
		return v
	}
	return "NoName"
}
