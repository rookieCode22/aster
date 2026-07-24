package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	home, err := os.UserHomeDir()
	fmt.Printf("os.UserHomeDir() = %q (err=%v)\n", home, err)
	fmt.Printf("  len=%d, bytes: %v\n", len(home), []byte(home))

	joined := filepath.Join(home, ".aster")
	fmt.Printf("\nfilepath.Join(home, .aster) = %q\n", joined)

	cleaned := filepath.Clean(joined)
	fmt.Printf("filepath.Clean(joined) = %q\n", cleaned)

	dsn := "sqlite:///" + filepath.Join(cleaned, "aster.db")
	fmt.Printf("\nDSN = %q\n", dsn)
	fmt.Printf("filepath.Dir(connStr after TrimPrefix) = %q\n",
		filepath.Dir(filepath.Join(cleaned, "aster.db")))

	// Simulate os.MkdirAll
	fmt.Printf("\nos.MkdirAll(%q, 0755)...\n", cleaned)
	if err := os.MkdirAll(cleaned, 0755); err != nil {
		fmt.Printf("  ERROR: %v\n", err)
	} else {
		fmt.Printf("  OK\n")
	}
}
