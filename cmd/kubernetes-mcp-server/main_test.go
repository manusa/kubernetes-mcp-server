package main

import (
	"os"
)

func Example_version() {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"kubernetes-mcp-server", "version", "--short"}
	main()
	// Output: 0.0.0
}
