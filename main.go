package main

import "github.com/jakeraft/clier/cmd"

// version is overridden at build time via -ldflags="-X main.version=…".
var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
