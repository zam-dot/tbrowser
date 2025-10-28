// main.go
package main

import (
	"github.com/rivo/tview"
)

// main.go
func main() {

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		config = DefaultConfigPtr()
	}

	app := tview.NewApplication()

	// Create the unified layout (starts in browser mode)
	_ = CreateSecurityLayout(app, config) // lowercase 'c'

	if err := app.Run(); err != nil {
		panic(err)
	}
}
