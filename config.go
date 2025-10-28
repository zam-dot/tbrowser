// Configuration management for TUI Browser.
// Handles TOML config file loading, validation, and default values.
// Manages user preferences for extraction, network, and UI settings.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func DefaultConfig() Config {
	return Config{
		Extraction: ExtractionConfig{
			PreserveLinks:    true,
			AddLineBreaks:    true,
			ShowLinksSection: true,
		},
		Network: NetworkConfig{
			TimeoutSeconds: 10,
			UserAgent:      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
		},
		UI: UIConfig{
			InitialMessage: `[yellow]Navigation:[-]
• Type URL or search with :d <query>
• Numbers to follow links
• ←/→ for history
• Ctrl+C to quit`,
		},
		SiteOverrides: map[string]SiteConfig{
			"reddit.com": {
				RemoveElements: []string{".side", ".footer", ".promoted"},
			},
			"twitter.com": {
				RemoveElements: []string{".sidebar", "[aria-label*='Trend']"},
			},
		},
	}
}

func DefaultConfigPtr() *Config {
	config := DefaultConfig()
	return &config
}

func LoadConfig() (*Config, error) {
	// Get config path directly without function call
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return nil, fmt.Errorf("could not find home directory: %w", homeErr)
	}
	configPath := filepath.Join(home, ".config", "tui-browser", "config.toml")

	config := DefaultConfigPtr()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, create it
		dir := filepath.Dir(configPath)
		if mkdirErr := os.MkdirAll(dir, 0755); mkdirErr != nil {
			return config, fmt.Errorf("creating config directory: %w", mkdirErr)
		}

		file, createErr := os.Create(configPath)
		if createErr != nil {
			return config, fmt.Errorf("creating config file: %w", createErr)
		}

		simpleConfig := `[extraction]
preserve_links = true
add_line_breaks = true
show_links_section = true

[network]
timeout_seconds = 10
user_agent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"

[ui]
initial_message = "Welcome to the simple Browser!"


# Add site-specific removals here:
# [site_overrides."example.com"]
# remove_elements = [".ads", ".sidebar"]
`

		_, writeErr := file.WriteString(simpleConfig)
		file.Close()

		if writeErr != nil {
			return config, fmt.Errorf("writing config: %w", writeErr)
		}

		return config, nil
	}

	// Config file exists, try to load it
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return config, fmt.Errorf("loading config: %w", err)
	}

	return config, nil
}
