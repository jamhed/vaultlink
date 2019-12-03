package main

import (
	"fmt"
	"vaultlink/app"
)

// Default is `-s -w -X main.version={{.Version}} -X main.commit={{.ShortCommit}} -X main.date={{.Date}} -X main.builtBy=goreleaser`.
var version string
var commit string
var date string
var builtBy string

func main() {
	fmt.Printf("vaultlink %s %s %s %s\n", version, commit, date, builtBy)
	app := app.New().Connect()
	app.Control()
}
