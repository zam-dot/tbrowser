// Type definitions for TUI Browser data structures.
// Contains all struct definitions for configuration, UI components, and business logic.
// Centralized type definitions ensure consistency across the application.
package main

import (
	"net/http"

	"github.com/rivo/tview"
	"golang.org/x/net/proxy"
)

// All your existing struct definitions go here...
type Config struct {
	Extraction    ExtractionConfig      `toml:"extraction"`
	Network       NetworkConfig         `toml:"network"`
	UI            UIConfig              `toml:"ui"`
	SiteOverrides map[string]SiteConfig `toml:"site_overrides"`
}

type SiteConfig struct {
	RemoveElements []string `toml:"remove_elements"`
	LinkFormat     string   `toml:"link_format"` // Add this
}

type ExtractionConfig struct {
	PreserveLinks    bool `toml:"preserve_links"`
	AddLineBreaks    bool `toml:"add_line_breaks"`
	ShowLinksSection bool `toml:"show_links_section"`
}

type NetworkConfig struct {
	TimeoutSeconds int    `toml:"timeout_seconds"`
	UserAgent      string `toml:"user_agent"`
}

type UIConfig struct {
	InitialMessage string `toml:"initial_message"`
	MaxTextWidth   int    `toml:"max_text_width"` // Add this
}

type LinkProcessor struct{}

type Extractor struct {
	config *Config
	links  *LinkProcessor
}

type URLResolver struct{}

type HistoryManager struct {
	history      []string
	historyIndex int
}

type ContentManager struct {
	currentURL   string
	currentLinks []string
	history      *HistoryManager
	resolver     *URLResolver
}

type Fetcher struct {
	client    *http.Client
	config    *Config
	useTor    bool
	torDialer proxy.Dialer
}

type RSSParser struct{}

type UIHandlers struct {
	contentManager *ContentManager
	fetcher        *Fetcher
	app            *tview.Application
	config         *Config
}
