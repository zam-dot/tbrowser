// Terminal UI component creation and focus management.
// Pure UI construction without business logic - follows separation of concerns.
// Handles component styling, layout, and focus indicators.
package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UI Components - Pure UI creation, no business logic
func CreateUIComponents() (*tview.TextView, *tview.InputField, *tview.TextView, *tview.Flex) {
	content := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	content.SetBorder(true).SetTitle(" Web Content ")
	content.SetBorderPadding(0, 0, 1, 1)

	input := tview.NewInputField().
		SetLabel("🌐 ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDarkSlateGray).
		SetFieldTextColor(tcell.ColorWhite)
	input.SetBorder(true).SetTitle(" Address Bar ")

	status := tview.NewTextView().
		SetDynamicColors(true)
	status.SetText(" [gray]∘[-] Ready ") // ∘ = empty circle for Tor off
	status.SetBackgroundColor(tcell.ColorDarkOliveGreen)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, false).
		AddItem(input, 3, 1, true).
		AddItem(status, 1, 1, false)

	return content, input, status, flex
}

func UpdateFocus(app *tview.Application, input *tview.InputField, content *tview.TextView) {
	currentFocus := app.GetFocus()
	switch currentFocus {
	case input:
		input.SetTitle(" [yellow]▶ Address Bar[-] ")
		content.SetTitle(" Web Content ")
	default:
		input.SetTitle(" Address Bar ")
		content.SetTitle(" [yellow]▶ Web Content[-] ")
	}
}

func DisplayInitialMessage(content *tview.TextView, config *Config) {
	fmt.Fprintln(content, config.UI.InitialMessage)
}
