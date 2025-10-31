// URL resolution, history management, and link following.
// Handles relative URL resolution and maintains navigation state.
// Provides forward/backward navigation with bounds checking.
package main

import (
	"fmt"
	"net/url"
	"strings"
)

func NormalizeURL(input string) string {
	input = strings.TrimSpace(input)

	// Handle .onion sites FIRST - force HTTP
	if strings.Contains(input, ".onion") {
		// Remove any existing protocol
		input = strings.TrimPrefix(input, "https://")
		input = strings.TrimPrefix(input, "http://")
		// Force HTTP for .onion
		return "http://" + input
	}

	// THEN handle regular URL normalization
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}

	if strings.HasPrefix(input, "//") {
		return "https:" + input
	}

	return "https://" + input
}

func (r *URLResolver) ResolveURL(link string, baseURL string) (string, error) {
	if baseURL == "" {
		return "", fmt.Errorf("cannot resolve relative link without base URL: %s", link)
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %s", baseURL)
	}

	linkURL, err := url.Parse(link)
	if err != nil {
		return "", fmt.Errorf("invalid link: %s", link)
	}

	resolved := base.ResolveReference(linkURL)
	return resolved.String(), nil
}

// History Management
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		history:      []string{},
		historyIndex: -1,
	}
}

func (h *HistoryManager) Add(url string) {
	if h.historyIndex < len(h.history)-1 {
		h.history = h.history[:h.historyIndex+1]
	}
	h.history = append(h.history, url)
	h.historyIndex = len(h.history) - 1

	// Add history size limit (keep last 100 entries)
	const maxHistory = 30
	if len(h.history) > maxHistory {
		// Remove oldest entries from the beginning
		removeCount := len(h.history) - maxHistory
		h.history = h.history[removeCount:]
		h.historyIndex -= removeCount
	}
}

func (h *HistoryManager) Back() (string, bool) {
	if h.historyIndex > 0 {
		h.historyIndex--
		return h.history[h.historyIndex], true
	}
	return "", false
}

func (h *HistoryManager) Forward() (string, bool) {
	if h.historyIndex < len(h.history)-1 {
		h.historyIndex++
		return h.history[h.historyIndex], true
	}
	return "", false
}

func (h *HistoryManager) CanGoBack() bool {
	return h.historyIndex > 0
}

func (h *HistoryManager) CanGoForward() bool {
	return h.historyIndex < len(h.history)-1
}

// Content Manager
func NewContentManager() *ContentManager {
	return &ContentManager{
		history:  NewHistoryManager(),
		resolver: &URLResolver{},
	}
}

func (cm *ContentManager) SetCurrentURL(url string) {
	cm.currentURL = url
}

func (cm *ContentManager) SetCurrentLinks(links []string) {
	cm.currentLinks = links
}

func (cm *ContentManager) FollowLink(linkNum int) (string, error) {
	if linkNum < 1 || linkNum > len(cm.currentLinks) {
		return "", fmt.Errorf("invalid link number: %d", linkNum)
	}

	link := cm.currentLinks[linkNum-1]
	return cm.resolver.ResolveURL(link, cm.currentURL)
}

func (cm *ContentManager) NavigateToURL(url string, addToHistory bool) string {
	if addToHistory {
		cm.history.Add(url)
	}
	cm.currentURL = url
	return url
}

func (cm *ContentManager) GoBack() (string, bool) {
	if url, ok := cm.history.Back(); ok {
		cm.currentURL = url
		return url, true
	}
	return "", false
}

func (cm *ContentManager) GoForward() (string, bool) {
	if url, ok := cm.history.Forward(); ok {
		cm.currentURL = url
		return url, true
	}
	return "", false
}

func (cm *ContentManager) CanGoBack() bool {
	return cm.history.CanGoBack()
}

func (cm *ContentManager) CanGoForward() bool {
	return cm.history.CanGoForward()
}
