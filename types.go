// types.go
package main

import (
	"net/http"

	"github.com/rivo/tview"
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
	client *http.Client
	config *Config
}

type RSSParser struct{}

type UIHandlers struct {
	contentManager *ContentManager
	fetcher        *Fetcher
	app            *tview.Application
	config         *Config
}
