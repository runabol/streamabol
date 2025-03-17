package env

import (
	"os"
)

// Get returns the value of an environment variable or a default value if it's not set.
func Get(key, defaultValue string) string {
	value := defaultValue
	if v, ok := os.LookupEnv(key); ok {
		value = v
	}
	return value
}

// Set sets the value of an environment variable.
func Set(key, value string) error {
	return os.Setenv(key, value)
}
