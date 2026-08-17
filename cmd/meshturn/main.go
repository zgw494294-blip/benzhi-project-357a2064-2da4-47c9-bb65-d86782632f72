package main

import (
	"encoding/json"
	"fmt"
	"os"

	"meshturn/internal/cli"
)

func main() {
	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]string{"error": err.Error()})
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
}
