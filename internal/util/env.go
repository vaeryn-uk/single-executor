// util provides general-case functionality used by, but not specific to, our implementations.
package util

import (
	"log"
	"os"
)

// Will exit the program if env name is empty or missing.
func MustGetEnv(name string) string {
	value := os.Getenv(name)

	if len(value) == 0 {
		log.Fatalf("Missing env var %s\n", name)
	}

	return value
}
