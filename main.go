package main

import (
	"os"

	"github.com/containifyci/dunebot/cmd"
	_ "github.com/containifyci/dunebot/cmd/app"
	_ "github.com/containifyci/dunebot/cmd/dispatch"
	_ "github.com/containifyci/dunebot/cmd/update"
	pkgversion "github.com/containifyci/dunebot/pkg/version"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	pkgversion.Init(version, commit, date)
	
	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
