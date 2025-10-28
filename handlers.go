// UI event handlers and input processing.
// Coordinates user actions with content fetching and display.
// Supports multiple search engines and navigation commands.
package main

import (
	"fmt"
	"net/url"
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
			cleanedText := cleanBlankLines(cleanText, 2)

			status.SetText(fmt.Sprintf(" [green]Loaded:[-] %s ", finalURL))
			content.SetText(fmt.Sprintf("[green]Content from:[-] %s\n[gray]%s[-]\n\n%s",
				finalURL,
				strings.Repeat("─", 50),
				cleanedText))
			content.ScrollToBeginning()
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

	h.HandleURLNavigation(url, content, status, true)
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

		if strings.HasPrefix(text, ":g ") {
			input.SetText("")
			query := strings.TrimSpace(text[3:])
			searchURL := "https://gutenberg.org/ebooks/search/?query=" + url.QueryEscape(query)
			handlers.HandleURLNavigation(searchURL, content, status, true)
			return
		}

		if strings.HasPrefix(text, ":gh ") {
			input.SetText("")
			query := strings.TrimSpace(text[4:])
			searchURL := "https://github.com/" + url.QueryEscape(query)
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
