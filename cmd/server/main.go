package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/morphy76/vuhive-cloud/internal/version"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vuhive-cloud server %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
		os.Exit(0)
	}

	fmt.Printf("Starting vuhive-cloud server %s...\n", version.Version)
}
