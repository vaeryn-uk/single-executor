package util

import (
	"log"
	"os"
)

func MustGetEnv(name string) string {
	value := os.Getenv(name)

	if len(value) == 0 {
		log.Fatalf("Missing env var %s\n", name)
	}

	return value
}
