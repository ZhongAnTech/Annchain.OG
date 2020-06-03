package utilfuncs

import "os"

func GetHostName() string {
	// Kubernetes first
	if v, ok := os.LookupEnv("HOSTNAME"); ok {
		return v
	}
	if v, err := os.Hostname(); err == nil {
		return v
	}
	return "NoName"
}
