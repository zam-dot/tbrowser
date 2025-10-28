// TUI Browser & Security Scanner - Entry point.
// Initializes configuration, creates application instance, and launches the TUI.
// Handles graceful shutdown and configuration loading errors.
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
