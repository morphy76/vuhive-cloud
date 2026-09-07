package main

import (
	"os"

	"github.com/morphy76/vuhive-cloud/internal/cli"
)

func main() {
	app := cli.NewApp(nil)
	code := app.Execute(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
