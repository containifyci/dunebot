package main

import (
	"os"

	"github.com/containifyci/dunebot/cmd"
	_ "github.com/containifyci/dunebot/cmd/app"
	_ "github.com/containifyci/dunebot/cmd/dispatch"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
