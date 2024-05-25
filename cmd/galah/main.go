package main

import (
	"github.com/0x4d31/galah/internal/app"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	app := app.App{}
	app.Run()
}
