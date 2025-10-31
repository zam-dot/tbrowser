// UI event handlers and input processing.
// Coordinates user actions with content fetching and display.
// Supports multiple search engines and navigation commands.
package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// UI Handlers
func NewUIHandlers(app *tview.Application) *UIHandlers {
	return &UIHandlers{
		contentManager: NewContentManager(),
		fetcher:        NewFetcher(),
		app:            app,
	}
}

func NewUIHandlersWithConfig(app *tview.Application, cfg *Config) *UIHandlers {
	return &UIHandlers{
		contentManager: NewContentManager(),
		fetcher:        NewFetcherWithConfig(cfg),
		app:            app,
		config:         cfg,
	}
}

// HandleURLNavigation fetches and displays web content.
// Runs asynchronously to keep UI responsive during network operations.
// If addToHistory is true, adds URL to navigation history.
func (h *UIHandlers) HandleURLNavigation(
	url string,
	content *tview.TextView,
	status *tview.TextView,
	addToHistory bool,
) {
	status.SetText(fmt.Sprintf(" [yellow]Loading:[-] %s ", url))
	content.SetText(fmt.Sprintf("Loading %s...", url))

	go func() {
		cleanText, links, err := h.fetcher.FetchURL(url)

		h.app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText(fmt.Sprintf(" [red]Error:[-] %s ", url))
				content.SetText(fmt.Sprintf("Error loading %s:\n%v", url, err))
				return
			}

			finalURL := h.contentManager.NavigateToURL(url, addToHistory)
			h.contentManager.SetCurrentLinks(links)

			// Clean excessive blank lines before displaying
			cleanedText := cleanBlankLines(cleanText, 1)

			// Add text wrapping if max width is configured
			if h.config.UI.MaxTextWidth > 0 {
				cleanedText = wrapText(cleanedText, h.config.UI.MaxTextWidth)
			}

			status.SetText(fmt.Sprintf(" [green]Loaded:[-] %s ", finalURL))
			content.SetText(fmt.Sprintf("[green]Content from:[-] %s\n[gray]%s[-]\n\n%s",
				finalURL,
				strings.Repeat("─", 50),
				cleanedText))
		})
	}()
}

func (h *UIHandlers) HandleLinkFollow(
	linkNum int,
	content *tview.TextView,
	status *tview.TextView,
) {
	if linkNum < 1 || linkNum > len(h.contentManager.currentLinks) {
		status.SetText(fmt.Sprintf(" [red]Invalid link number: %d[-] ", linkNum))
		return
	}

	url, err := h.contentManager.FollowLink(linkNum)
	if err != nil {
		status.SetText(fmt.Sprintf(" [red]%s[-] ", err.Error()))
		return
	}

	// Check if it's a magnet link
	if strings.HasPrefix(url, "magnet:?") {
		err := openMagnetLink(url)
		if err != nil {
			status.SetText(fmt.Sprintf(" [red]Error opening magnet link: %v[-] ", err))
		} else {
			status.SetText(" [green]Opening magnet link in torrent client...[-] ")
		}
		return // Don't navigate to magnet links as URLs
	}

	h.HandleURLNavigation(url, content, status, true)
}

func openMagnetLink(magnetURI string) error {
	// This will open the system's default torrent client
	return exec.Command("xdg-open", magnetURI).Start() // Linux
	// For macOS: return exec.Command("open", magnetURI).Start()
	// For Windows: return exec.Command("cmd", "/c", "start", magnetURI).Start()
}

func (h *UIHandlers) HandleBackNavigation(
	content *tview.TextView,
	status *tview.TextView,
) {
	if url, ok := h.contentManager.GoBack(); ok {
		h.HandleURLNavigation(url, content, status, false)
	} else {
		status.SetText(" [yellow]Already at beginning of history[-] ")
	}
}

func (h *UIHandlers) HandleForwardNavigation(
	content *tview.TextView,
	status *tview.TextView,
) {
	if url, ok := h.contentManager.GoForward(); ok {
		h.HandleURLNavigation(url, content, status, false)
	} else {
		status.SetText(" [yellow]Already at end of history[-] ")
	}
}

func (h *UIHandlers) HandleRSSNavigation(
	url string,
	content *tview.TextView,
	status *tview.TextView,
) {
	status.SetText(fmt.Sprintf(" [yellow]Loading RSS:[-] %s ", url))

	go func() {
		cleanText, err := h.fetcher.FetchRSS(url)

		h.app.QueueUpdateDraw(func() {
			if err != nil {
				status.SetText(fmt.Sprintf(" [red]RSS Error:[-] %s ", err))
				content.SetText(fmt.Sprintf("Failed to fetch RSS feed:\n%v", err))
				return
			}

			parser := &RSSParser{}
			formatted, links := parser.ParseRSS(cleanText)

			h.contentManager.SetCurrentLinks(links)
			h.contentManager.SetCurrentURL(url)

			content.SetText(fmt.Sprintf("[green]RSS Feed:[-] %s\n[gray]%s[-]\n\n%s",
				url,
				strings.Repeat("─", 50),
				formatted))
			content.ScrollToBeginning()
		})
	}()
}

// Input Handlers
func CreateInputHandler(
	handlers *UIHandlers,
	input *tview.InputField,
	content *tview.TextView,
	status *tview.TextView,
) func(key tcell.Key) {
	return func(key tcell.Key) {
		text := strings.TrimSpace(input.GetText())
		if text == "" {
			return
		}

		// In handlers.go - in CreateInputHandler()
		if text == ":tor" {
			input.SetText("")
			err := handlers.fetcher.EnableTor(!handlers.fetcher.IsTorEnabled())
			if err != nil {
				status.SetText(fmt.Sprintf(" [red]Tor Error: %v[-] ", err))
			} else {
				UpdateTorIndicator(input, handlers.fetcher.IsTorEnabled())
				status.SetText(fmt.Sprintf(" [yellow]Tor: %t[-] ", handlers.fetcher.IsTorEnabled()))
			}
			return
		}

		if strings.HasPrefix(text, ":d ") {
			input.SetText("")
			query := strings.TrimSpace(text[3:])
			searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(query)
			handlers.HandleURLNavigation(searchURL, content, status, true)
			return
		}

		if strings.HasPrefix(text, ":rss ") {
			input.SetText("")
			feedURL := strings.TrimSpace(text[5:])
			feedURL = NormalizeURL(feedURL)
			handlers.HandleRSSNavigation(feedURL, content, status)
			return
		}

		if strings.HasPrefix(text, ":m ") {
			input.SetText("")
			query := strings.TrimSpace(text[3:])
			searchURL := "https://www.mojeek.com/search?q=" + url.QueryEscape(
				query,
			) + "&format=html"
			handlers.HandleURLNavigation(searchURL, content, status, true)
			return
		}

		if strings.HasPrefix(text, ":w ") {
			input.SetText("")
			query := strings.TrimSpace(text[3:])
			searchURL := "https://en.wikipedia.org/wiki/" + url.QueryEscape(query)
			handlers.HandleURLNavigation(searchURL, content, status, true)
			return
		}

		if strings.HasPrefix(text, ":t ") {
			input.SetText("")
			query := strings.TrimSpace(text[3:])
			searchURL := "https://www.limetorrents.fun/search/all/" + url.QueryEscape(query)
			handlers.HandleURLNavigation(searchURL, content, status, true)
			return
		}

		if strings.HasPrefix(text, ":g ") {
			input.SetText("")
			query := strings.TrimSpace(text[3:])
			searchURL := "https://gutenberg.org/ebooks/search/?query=" + url.QueryEscape(query)
			handlers.HandleURLNavigation(searchURL, content, status, true)
			return
		}

		if linkNum, err := strconv.Atoi(text); err == nil {
			input.SetText("")
			handlers.HandleLinkFollow(linkNum, content, status)
			return
		}

		input.SetText("")
		normalizedURL := NormalizeURL(text)
		handlers.HandleURLNavigation(normalizedURL, content, status, true)
	}
}

func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var result strings.Builder

	for _, line := range lines {
		// Check visible length (without color codes)
		visibleLength := visibleLength(line)
		if visibleLength <= maxWidth {
			result.WriteString(line + "\n")
			continue
		}

		// Word wrap considering color codes
		result.WriteString(wrapLine(line, maxWidth) + "\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// Helper to calculate visible text length (ignoring color codes)
func visibleLength(text string) int {
	// Simple regex to remove color codes like [blue], [red], [-]
	re := regexp.MustCompile(`\[[^\]]*\]`)
	clean := re.ReplaceAllString(text, "")
	return len(clean)
}

// Wrap a single line with color code preservation
func wrapLine(line string, maxWidth int) string {
	var result strings.Builder
	words := strings.Fields(line)
	currentLine := ""
	currentVisibleLength := 0

	for _, word := range words {
		wordVisibleLength := visibleLength(word)

		if currentVisibleLength+wordVisibleLength+1 > maxWidth {
			if currentLine != "" {
				result.WriteString(currentLine + "\n")
			}
			currentLine = word
			currentVisibleLength = wordVisibleLength
		} else {
			if currentLine != "" {
				currentLine += " "
				currentVisibleLength++
			}
			currentLine += word
			currentVisibleLength += wordVisibleLength
		}
	}

	if currentLine != "" {
		result.WriteString(currentLine)
	}

	return result.String()
}

func CreateInputCapture(
	handlers *UIHandlers,
	input *tview.InputField,
	content *tview.TextView,
	status *tview.TextView,
	app *tview.Application,
	updateFocus func(),
) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		updateFocus()
		switch event.Key() {
		case tcell.KeyLeft:
			handlers.HandleBackNavigation(content, status)
			return nil
		case tcell.KeyRight:
			handlers.HandleForwardNavigation(content, status)
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyEscape:
			input.SetText("")
			return nil
		}
		return event
	}
}

func CreateContentInputCapture(
	handlers *UIHandlers,
	input *tview.InputField,
	content *tview.TextView,
	status *tview.TextView,
	app *tview.Application,
	updateFocus func(),
) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		updateFocus()
		switch event.Key() {
		case tcell.KeyPgUp:
			row, _ := content.GetScrollOffset()
			newRow := max(0, row-15)
			content.ScrollTo(newRow, 0)
			return nil
		case tcell.KeyPgDn:
			row, _ := content.GetScrollOffset()
			content.ScrollTo(row+15, 0)
			return nil
		case tcell.KeyHome:
			content.ScrollToBeginning()
			return nil
		case tcell.KeyEnd:
			content.ScrollToEnd()
			return nil
		case tcell.KeyUp:
			row, _ := content.GetScrollOffset()
			newRow := max(0, row-2)
			content.ScrollTo(newRow, 0)
			return nil
		case tcell.KeyDown:
			row, _ := content.GetScrollOffset()
			content.ScrollTo(row+2, 0)
			return nil
		case tcell.KeyLeft:
			handlers.HandleBackNavigation(content, status)
			return nil
		case tcell.KeyRight:
			handlers.HandleForwardNavigation(content, status)
			return nil
		case tcell.KeyCtrlC:
			app.Stop()
			return nil
		case tcell.KeyRune:
			app.SetFocus(input)
			return nil
		}
		return event
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
