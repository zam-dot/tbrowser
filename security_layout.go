// SecurityLayout manages the dual-mode interface (Browser ↔ Security).
// Dual-mode interface management (Browser ↔ Security Scanner).
// Handles mode switching, focus management, and component coordination.
// Provides F1/F2 keyboard shortcuts for seamless mode transitions.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"browser/security"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppMode int

const (
	ModeBrowser AppMode = iota
	ModeSecurity
	ModeExplorer // Add this line
)

// Add this struct definition near your other type definitions
type ResourceItem struct {
	Type string
	URL  string
	Size string
}

type SecurityLayout struct {
	app              *tview.Application
	securityFlex     *tview.Flex
	browserFlex      *tview.Flex
	currentRoot      tview.Primitive
	currentMode      AppMode
	config           *Config
	browserHandlers  *UIHandlers
	currentResources []ResourceItem
	currentBaseURL   string

	// Security components
	techPanel      *tview.TextView
	vulnPanel      *tview.TextView
	headersPanel   *tview.TextView
	requestsPanel  *tview.TextView
	securityInput  *tview.InputField
	securityStatus *tview.TextView

	// Browser components
	browserContent *tview.TextView
	browserInput   *tview.InputField
	browserStatus  *tview.TextView

	// Explorer components
	explorerFlex     *tview.Flex
	explorerContent  *tview.TextView
	explorerInput    *tview.InputField
	explorerStatus   *tview.TextView
	explorerSections *tview.List
}

// FILE EXPLORER LAYOUT
func (l *SecurityLayout) setupExplorerLayout() {
	// Initialize explorer components first
	l.explorerContent = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	l.explorerContent.SetBorder(true).SetTitle(" Site Resources ")
	l.explorerContent.SetBorderPadding(0, 0, 1, 1)

	l.explorerSections = tview.NewList().
		ShowSecondaryText(false)
	l.explorerSections.SetBorder(true).SetTitle(" Sections ")
	l.explorerSections.SetBorderPadding(0, 0, 1, 1)

	l.explorerInput = tview.NewInputField().
		SetLabel("🔍 ").
		SetFieldWidth(0)
	l.explorerInput.SetBorder(true).SetTitle(" Explorer Commands ")

	l.explorerStatus = tview.NewTextView().SetDynamicColors(true)
	l.explorerStatus.SetText(" Ready ")
	l.explorerStatus.SetBackgroundColor(tcell.ColorDarkBlue)

	// Two-panel layout: sections list + content
	contentRow := tview.NewFlex().
		AddItem(l.explorerSections, 30, 1, false). // 30% width for sections
		AddItem(l.explorerContent, 0, 1, false)    // 70% width for content

	l.explorerFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(contentRow, 0, 1, false).
		AddItem(l.explorerInput, 3, 1, true).
		AddItem(l.explorerStatus, 1, 1, false)

	// Initialize currentResources to avoid nil issues
	l.currentResources = []ResourceItem{}

	l.setupExplorerCommands()
	l.setupExplorerSections()
}

// Update the visitReconLink method to handle nil browserHandlers gracefully
func (l *SecurityLayout) visitReconLink(index int) {
	if index < 0 || index >= len(l.currentResources) {
		return
	}

	// Check if browserHandlers is initialized
	if l.browserHandlers == nil {
		l.explorerStatus.SetText(" [red]Browser not initialized[-] ")
		return
	}

	resource := l.currentResources[index]

	// Resolve relative URL to absolute
	fullURL, err := l.browserHandlers.contentManager.resolver.ResolveURL(resource.URL, l.currentBaseURL)
	if err != nil {
		l.explorerStatus.SetText(fmt.Sprintf(" [red]Error resolving URL:[-] %s ", err.Error()))
		return
	}

	// Switch to browser mode
	l.SwitchToBrowserMode()

	// Set the URL in input field and focus it
	l.browserInput.SetText(fullURL)
	l.app.SetFocus(l.browserInput)

	// Status message tells user to press Enter
	l.browserStatus.SetText(fmt.Sprintf(" [yellow]URL set - Press Enter to navigate to:[-] %s ", fullURL))
}

// Also update the initialization order in CreateSecurityLayout
func CreateSecurityLayout(app *tview.Application, config *Config) *SecurityLayout {
	layout := &SecurityLayout{
		app:    app,
		config: config,
	}

	// Initialize ALL layouts first
	layout.setupBrowserLayout()  // This creates browserFlex
	layout.setupSecurityLayout() // This creates securityFlex
	layout.setupExplorerLayout() // This creates explorerFlex

	// Now switch to browser mode (all flex layouts are initialized)
	layout.SwitchToBrowserMode()

	return layout
}

func (l *SecurityLayout) setupExplorerSections() {
	l.explorerSections.Clear()

	sections := []struct {
		title string
		desc  string
		cmd   string
	}{
		{"📁 Downloadable Files", "PDFs, ZIPs, documents", ":files"},
		{"🖼️ Images & Media", "Images, videos, audio", ":images"},
		{"📜 Scripts & Styles", "JS, CSS files", ":scripts"},
		{"🔗 Links & Endpoints", "All URLs and APIs", ":links"},
		{"📊 Page Resources", "Metadata and headers", ":resources"},
	}

	for _, section := range sections {
		l.explorerSections.AddItem(section.title, section.desc, 0, func() {
			l.explorerInput.SetText(section.cmd + " " + l.getCurrentURL())
			l.handleExplorerCommand()
		})
	}
}

func (l *SecurityLayout) getCurrentURL() string {
	// Get URL from current browser context
	if l.browserHandlers != nil && l.browserHandlers.contentManager != nil {
		return l.browserHandlers.contentManager.currentURL
	}
	return ""
}

func (l *SecurityLayout) setupExplorerCommands() {
	l.explorerInput.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			l.handleExplorerCommand()
		}
	})

	// Global explorer input capture for F1/F2/F3
	l.explorerFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			l.SwitchToBrowserMode()
			return nil
		case tcell.KeyF2:
			l.SwitchToSecurityMode()
			return nil
		case tcell.KeyF3:
			// Already in explorer, do nothing or could refresh
			return nil
		case tcell.KeyTab:
			// Handle Tab switching between explorer components
			l.cycleExplorerFocus()
			return nil
		}
		return event
	})

	// Input capture for explorer content
	l.explorerContent.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle number keys 1-9 for quick navigation
		if event.Rune() >= '1' && event.Rune() <= '9' {
			index := int(event.Rune() - '1') // Convert '1' to 0, '2' to 1, etc.
			if index < len(l.currentResources) {
				l.visitReconLink(index)
				return nil
			}
		}

		// Handle Tab for focus switching within explorer
		if event.Key() == tcell.KeyTab {
			l.cycleExplorerFocus()
			return nil
		}

		// Also handle F1/F2/F3 in content area
		switch event.Key() {
		case tcell.KeyF1:
			l.SwitchToBrowserMode()
			return nil
		case tcell.KeyF2:
			l.SwitchToSecurityMode()
			return nil
		}

		return event
	})

	// Input capture for explorer sections list
	l.explorerSections.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			l.cycleExplorerFocus()
			return nil
		}

		// Also handle F1/F2/F3 in sections area
		switch event.Key() {
		case tcell.KeyF1:
			l.SwitchToBrowserMode()
			return nil
		case tcell.KeyF2:
			l.SwitchToSecurityMode()
			return nil
		}

		return event
	})
}

// Add this method to handle focus cycling in explorer mode
func (l *SecurityLayout) cycleExplorerFocus() {
	currentFocus := l.app.GetFocus()

	switch currentFocus {
	case l.explorerInput:
		l.app.SetFocus(l.explorerSections)
		l.explorerSections.SetBorderColor(tcell.ColorYellow)
		l.explorerInput.SetBorderColor(tcell.ColorWhite)
		l.explorerContent.SetBorderColor(tcell.ColorWhite)
	case l.explorerSections:
		l.app.SetFocus(l.explorerContent)
		l.explorerContent.SetBorderColor(tcell.ColorYellow)
		l.explorerSections.SetBorderColor(tcell.ColorWhite)
		l.explorerInput.SetBorderColor(tcell.ColorWhite)
	case l.explorerContent:
		l.app.SetFocus(l.explorerInput)
		l.explorerInput.SetBorderColor(tcell.ColorYellow)
		l.explorerContent.SetBorderColor(tcell.ColorWhite)
		l.explorerSections.SetBorderColor(tcell.ColorWhite)
	default:
		l.app.SetFocus(l.explorerInput)
		l.explorerInput.SetBorderColor(tcell.ColorYellow)
	}
}

// Add this method to display file content
func (l *SecurityLayout) viewFileContent(index int) {
	if index < 0 || index >= len(l.currentResources) {
		return
	}

	resource := l.currentResources[index]

	// Resolve relative URL to absolute
	fullURL, err := l.browserHandlers.contentManager.resolver.ResolveURL(resource.URL, l.currentBaseURL)
	if err != nil {
		l.explorerStatus.SetText(fmt.Sprintf(" [red]Error resolving URL:[-] %s ", err.Error()))
		return
	}

	l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Loading:[-] %s ", fullURL))

	go func() {
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Request failed:[-] %s ", err.Error()))
			})
			return
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Download failed:[-] %s ", err.Error()))
			})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]HTTP Error %d[-] ", resp.StatusCode))
			})
			return
		}

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Read failed:[-] %s ", err.Error()))
			})
			return
		}

		l.app.QueueUpdateDraw(func() {
			// Format the content based on file type
			formatted := l.formatFileContent(string(content), resource.Type, resource.URL)
			l.explorerContent.SetText(formatted)
			l.explorerStatus.SetText(fmt.Sprintf(" [green]Loaded:[-] %s (%d bytes) ", fullURL, len(content)))
		})
	}()
}

// Add method to format different file types
func (l *SecurityLayout) formatFileContent(content string, fileType string, url string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[yellow]📄 FILE VIEWER: %s[-]\n", url))
	builder.WriteString(fmt.Sprintf("[white]Type: %s | Length: %d bytes[-]\n\n", fileType, len(content)))

	switch fileType {
	case "json":
		// Try to pretty-print JSON
		var jsonData any
		if err := json.Unmarshal([]byte(content), &jsonData); err == nil {
			if pretty, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
				builder.WriteString("[green]// Formatted JSON:[-]\n")
				builder.WriteString(string(pretty))
			} else {
				builder.WriteString(content)
			}
		} else {
			builder.WriteString("[yellow]// Invalid JSON - raw content:[-]\n")
			builder.WriteString(content)
		}

	case "script":
		builder.WriteString("[green]// JavaScript:[-]\n")
		builder.WriteString(content)

	case "config":
		builder.WriteString("[green]// Configuration:[-]\n")
		builder.WriteString(content)

	default:
		builder.WriteString("[green]// Raw content:[-]\n")
		// Limit display for very large files
		if len(content) > 10000 {
			builder.WriteString(content[:10000])
			builder.WriteString(fmt.Sprintf("\n\n[yellow]... (truncated, %d more bytes)[-]", len(content)-10000))
		} else {
			builder.WriteString(content)
		}
	}

	builder.WriteString("\n\n[yellow]💡 Press F3 to return to resource list[-]")

	return builder.String()
}

// Add this method to SecurityLayout
func (l *SecurityLayout) handleExplorerCommand() {
	text := strings.TrimSpace(l.explorerInput.GetText())
	if text == "" {
		return
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]

	// Handle quick download: "d 13"
	if cmd == "d" && len(parts) >= 2 {
		l.handleDownloadByIndex(parts[1])
		l.explorerInput.SetText("")
		return
	}

	// Handle bulk download: "bulk 1,3,5-8"
	if cmd == "bulk" && len(parts) >= 2 {
		l.handleBulkDownload(parts[1])
		l.explorerInput.SetText("")
		return
	}

	// Handle bulk download all: "bulk all"
	if cmd == "bulk" && len(parts) >= 2 && parts[1] == "all" {
		l.handleBulkDownloadAll()
		l.explorerInput.SetText("")
		return
	}

	// Handle view file: ":view 13"
	if cmd == ":view" && len(parts) >= 2 {
		index, err := strconv.Atoi(parts[1])
		if err == nil && index > 0 && index <= len(l.currentResources) {
			l.viewFileContent(index - 1)
		}
		l.explorerInput.SetText("")
		return
	}

	if len(parts) < 2 {
		l.explorerStatus.SetText(" [red]Usage: <command> <url>[-] ")
		return
	}

	targetURL := strings.Join(parts[1:], " ")

	// Handle manual download
	if cmd == ":download" {
		if len(parts) >= 2 {
			filename := ""
			if len(parts) >= 3 {
				filename = parts[2]
			}
			l.DownloadFile(targetURL, filename)
			l.explorerInput.SetText("")
			return
		} else {
			l.explorerStatus.SetText(" [red]Usage: :download <url> [filename][-] ")
			return
		}
	}

	normalizedURL := NormalizeURL(targetURL)

	switch cmd {
	case ":files", ":images", ":scripts", ":links", ":resources":
		l.ScanSiteResources(normalizedURL, cmd[1:])
	case ":recon":
		l.securityReconnaissance(normalizedURL)
	default:
		l.explorerStatus.SetText(" [red]Unknown explorer command[-] ")
	}

	l.explorerInput.SetText("")
}

func (l *SecurityLayout) securityReconnaissance(targetURL string) {
	normalizedURL := NormalizeURL(targetURL)
	l.explorerStatus.SetText(fmt.Sprintf(" [red]Security recon:[-] %s ", normalizedURL))

	go func() {
		scanner := security.NewSecurityScanner()
		html, _, err := scanner.FetchWithHeaders(normalizedURL)
		// Add these to your reconnaissance
		sensitiveFiles := l.extractSensitiveFiles(html)
		backupFiles := l.extractBackupFiles(html)

		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.explorerContent.SetText(fmt.Sprintf("[red]Recon failed:[-]\n%s", err.Error()))
				return
			}

			exposedEndpoints := l.findExposedEndpoints(html)
			adminInterfaces := l.findAdminInterfaces(html)
			debugFiles := l.findDebugFiles(html)
			interestingParams := l.findInterestingParameters(html)

			// Store all findings for navigation
			l.currentResources = append(append(append(append(
				exposedEndpoints, adminInterfaces...),
				debugFiles...), interestingParams...),
				append(sensitiveFiles, backupFiles...)...)
			l.currentBaseURL = normalizedURL

			var builder strings.Builder
			builder.WriteString("[red]🔒 PRACTICAL SECURITY RECONNAISSANCE[-]\n")
			builder.WriteString("[yellow]Press number key to visit link, F1 to return to browser[-]\n\n")

			// Categorize and prioritize findings - FIXED TYPO: exposedEndpoints
			builder.WriteString("[yellow]🚨 HIGH PRIORITY - ADMIN & AUTH:[-]\n")
			highPriority := append(adminInterfaces, exposedEndpoints...) // FIXED: exposedEndpoints
			for i, item := range highPriority {
				risk := l.assessRiskLevel(item.URL)
				builder.WriteString(fmt.Sprintf("[[%d]] [red]%s[-] %s\n", i+1, risk, item.URL))
			}

			builder.WriteString("\n[yellow]⚠️ MEDIUM PRIORITY - DEBUG & INFO:[-]\n")
			for i, item := range debugFiles {
				risk := l.assessRiskLevel(item.URL)
				builder.WriteString(fmt.Sprintf("[[%d]] [yellow]%s[-] %s\n", len(highPriority)+i+1, risk, item.URL))
			}

			builder.WriteString("\n[yellow]🔍 INTERESTING PARAMETERS:[-]\n")
			for i, item := range interestingParams {
				risk := l.assessRiskLevel(item.URL)
				builder.WriteString(fmt.Sprintf("[[%d]] [blue]%s[-] %s\n", len(highPriority)+len(debugFiles)+i+1, risk, item.URL))
			}

			if len(l.currentResources) == 0 {
				builder.WriteString("\n[green]✅ No obvious security issues found[-]\n")
			} else {
				builder.WriteString(fmt.Sprintf("\n[yellow]💡 Found %d endpoints - Press 1-%d to visit[-]\n", len(l.currentResources), len(l.currentResources)))
			}

			l.explorerContent.SetText(builder.String())
			l.explorerStatus.SetText(fmt.Sprintf(" [green]Found %d endpoints - Press number to visit[-] ", len(l.currentResources)))
		})
	}()
}

// Add risk assessment helper
func (l *SecurityLayout) assessRiskLevel(url string) string {
	highRiskKeywords := []string{"admin", "login", "password", "auth", "token", "key", "secret"}
	mediumRiskKeywords := []string{"settings", "account", "config", "api", "upload", "export"}
	debugKeywords := []string{"debug", "test", "demo", "phpinfo", "version", "status"}

	lowerURL := strings.ToLower(url)

	for _, keyword := range highRiskKeywords {
		if strings.Contains(lowerURL, keyword) {
			return "🚨"
		}
	}

	for _, keyword := range mediumRiskKeywords {
		if strings.Contains(lowerURL, keyword) {
			return "⚠️"
		}
	}

	for _, keyword := range debugKeywords {
		if strings.Contains(lowerURL, keyword) {
			return "🔧"
		}
	}

	return "🔍"
}

func (l *SecurityLayout) findExposedEndpoints(html string) []ResourceItem {
	var endpoints []ResourceItem

	patterns := map[string]string{
		"api":    `href="([^"]*(?:api|graphql|rest|json)[^"]*)"`,
		"upload": `href="([^"]*(?:upload|import|file)[^"]*)"`,
		"export": `href="([^"]*(?:export|download|backup)[^"]*)"`,
		"config": `href="([^"]*(?:config|settings)[^"]*)"`,

		// ADD THESE FILE EXTENSION PATTERNS:
		"json": `href="([^"]*\.json[^"]*)"`,  // JSON files
		"pdf":  `href="([^"]*\.pdf[^"]*)"`,   // PDF files
		"doc":  `href="([^"]*\.docx?[^"]*)"`, // Word documents
		"xls":  `href="([^"]*\.xlsx?[^"]*)"`, // Excel files
		"zip":  `href="([^"]*\.zip[^"]*)"`,   // ZIP archives
		"sql":  `href="([^"]*\.sql[^"]*)"`,   // SQL files
	}

	for endpointType, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				endpoints = append(endpoints, ResourceItem{
					Type: endpointType,
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return endpoints
}

func (l *SecurityLayout) findAdminInterfaces(html string) []ResourceItem {
	var admins []ResourceItem

	patterns := []string{
		`href="([^"]*(?:admin|administrator|dashboard|login|signin)[^"]*)"`,
		`href="([^"]*(?:wp-admin|phpmyadmin|cpanel|webmin)[^"]*)"`,
		`href="([^"]*(?:manager|console|control)[^"]*)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				admins = append(admins, ResourceItem{
					Type: "admin",
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return admins
}

func (l *SecurityLayout) findDebugFiles(html string) []ResourceItem {
	var debugFiles []ResourceItem

	patterns := []string{
		`href="([^"]*(?:debug|test|demo|example)[^"]*)"`,
		`href="([^"]*(?:phpinfo|test\\.php)[^"]*)"`,
		`href="([^"]*(?:version|status|health)[^"]*)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				debugFiles = append(debugFiles, ResourceItem{
					Type: "debug",
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return debugFiles
}

func (l *SecurityLayout) findInterestingParameters(html string) []ResourceItem {
	var params []ResourceItem

	// Look for URLs with interesting parameters
	patterns := []string{
		`href="([^"?]*(?:id|user|key|token|auth|password|debug|admin)[^"]*)"`,
		`href="([^"?]*\?(?:[^"]*&)?(?:id|user|key|token|auth|password|debug|admin)=[^"]*)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				params = append(params, ResourceItem{
					Type: "parameter",
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return params
}

func (l *SecurityLayout) extractSensitiveFiles(html string) []ResourceItem {
	var files []ResourceItem

	patterns := map[string]string{ // FIXED: Remove the "p" at the beginning
		"database": `href="([^"]*\.(?:sql|db|mdb|sqlite|dbf)[^"]*)"`,
		"key":      `href="([^"]*\.(?:key|pem|crt|cer|pfx|p12)[^"]*)"`,
		"env":      `href="([^"]*\.[eE][nN][vV][^"]*)"`,
		"secret":   `href="([^"]*\.(?:secret|private|passwd)[^"]*)"`,
		"dump":     `href="([^"]*\.(?:dump|export|backup)[^"]*)"`,
	}

	for fileType, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				files = append(files, ResourceItem{
					Type: fileType,
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return files
}

func (l *SecurityLayout) extractBackupFiles(html string) []ResourceItem {
	var files []ResourceItem

	patterns := map[string]string{
		"backup": `href="([^"]*\.(?:bak|backup|old|save|~)[^"]*)"`,
		"temp":   `href="([^"]*\.(?:tmp|temp)[^"]*)"`,
		"swap":   `href="([^"]*\.(?:swp|swo)[^"]*)"`,
		"log":    `href="([^"]*\.[^"]*\.log[^"]*)"`, // More specific log files
	}

	for fileType, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				files = append(files, ResourceItem{
					Type: fileType,
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return files
}

// Add these methods to SecurityLayout

func (l *SecurityLayout) handleBulkDownload(rangeStr string) {
	indices, err := l.parseRange(rangeStr)
	if err != nil {
		l.explorerStatus.SetText(fmt.Sprintf(" [red]Invalid range:[-] %s ", err.Error()))
		return
	}

	if len(indices) == 0 {
		l.explorerStatus.SetText(" [red]No valid indices found[-] ")
		return
	}

	l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Bulk downloading %d files:[-] %v ", len(indices), indices))
	go l.bulkDownloadFiles(indices)
}

func (l *SecurityLayout) handleBulkDownloadAll() {
	if len(l.currentResources) == 0 {
		l.explorerStatus.SetText(" [red]No resources available[-] ")
		return
	}

	// Create slice with all indices [1, 2, 3, ..., n]
	allIndices := make([]int, len(l.currentResources))
	for i := range allIndices {
		allIndices[i] = i + 1
	}

	l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Bulk downloading ALL %d files[-] ", len(allIndices)))
	go l.bulkDownloadFiles(allIndices)
}

func (l *SecurityLayout) bulkDownloadFiles(indices []int) {
	downloadDir := l.getDownloadDirectory()
	successCount := 0
	failCount := 0

	for i, index := range indices {
		if index < 1 || index > len(l.currentResources) {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Skipping invalid index %d[-] ", index))
			})
			failCount++
			continue
		}

		resource := l.currentResources[index-1]

		// Resolve relative URLs
		fullURL, err := l.browserHandlers.contentManager.resolver.ResolveURL(resource.URL, l.currentBaseURL)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Error resolving URL %d:[-] %s ", index, err.Error()))
			})
			failCount++
			continue
		}

		// Extract filename from URL
		filename := ""
		parts := strings.Split(resource.URL, "/")
		if len(parts) > 0 {
			filename = parts[len(parts)-1]
			if filename == "" {
				filename = fmt.Sprintf("download_%d", index)
			}
		}

		// Update status for current download
		l.app.QueueUpdateDraw(func() {
			l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Bulk: [%d/%d] Downloading #%d:[-] %s ",
				i+1, len(indices), index, filename))
		})

		// Download the file
		if l.downloadSingleFile(fullURL, filename, downloadDir) {
			successCount++
		} else {
			failCount++
		}

		// Add delay between downloads (except for the last one)
		if i < len(indices)-1 {
			time.Sleep(2 * time.Second) // 2 second delay
		}
	}

	// Final status
	finalStatus := fmt.Sprintf(" [green]Bulk download complete: %d success, %d failed[-] ", successCount, failCount)
	if failCount > 0 {
		finalStatus = fmt.Sprintf(" [yellow]Bulk download complete: %d success, %d failed[-] ", successCount, failCount)
	}

	l.app.QueueUpdateDraw(func() {
		l.explorerStatus.SetText(finalStatus)
	})
}

func (l *SecurityLayout) downloadSingleFile(fileURL string, filename string, downloadDir string) bool {
	// Create request with proper headers
	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// Extract filename from URL more efficiently
	if filename == "" {
		if lastSlash := strings.LastIndex(fileURL, "/"); lastSlash != -1 && lastSlash < len(fileURL)-1 {
			filename = fileURL[lastSlash+1:]
		} else {
			filename = "download"
		}
	}

	fullPath := filepath.Join(downloadDir, filename)
	err = os.WriteFile(fullPath, body, 0o644)

	return err == nil
}

func (l *SecurityLayout) parseRange(rangeStr string) ([]int, error) {
	var indices []int

	// Most efficient: split and trim in a single operation
	start := 0
	for i, r := range rangeStr {
		if r == ',' {
			if start < i {
				part := strings.TrimSpace(rangeStr[start:i])
				indices = append(indices, l.parseRangePart(part)...)
			}
			start = i + 1
		}
	}
	// Don't forget the last part
	if start < len(rangeStr) {
		part := strings.TrimSpace(rangeStr[start:])
		indices = append(indices, l.parseRangePart(part)...)
	}

	return indices, nil
}

// Helper method to parse individual range parts
func (l *SecurityLayout) parseRangePart(part string) []int {
	var partIndices []int

	if strings.Contains(part, "-") {
		rangeParts := strings.Split(part, "-")
		if len(rangeParts) != 2 {
			return partIndices
		}

		start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
		if err1 != nil || err2 != nil || start > end {
			return partIndices
		}

		for i := start; i <= end; i++ {
			partIndices = append(partIndices, i)
		}
	} else {
		num, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			partIndices = append(partIndices, num)
		}
	}

	return partIndices
}

// security_layout.go - Add this helper function
func cleanBlankLines(text string, maxBlankLines int) string {
	lines := strings.Split(text, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= maxBlankLines {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (l *SecurityLayout) extractResources(html string, baseURL string, resourceType string) []ResourceItem {
	var resources []ResourceItem

	switch resourceType {
	case "files":
		resources = l.extractFiles(html)
	case "images":
		resources = l.extractImages(html, baseURL) // This should work now
	case "scripts":
		resources = l.extractScripts(html) // Only one definition now
	case "links":
		resources = l.extractLinks(html, baseURL)
	case "resources":
		resources = l.extractAllResources(html, baseURL)
	default:
		resources = []ResourceItem{}
	}

	return resources
}

func (l *SecurityLayout) extractFiles(html string) []ResourceItem {
	var files []ResourceItem

	// Enhanced patterns for security reconnaissance
	patterns := map[string]string{
		// Common documents
		"pdf": `href="([^"]*\.pdf[^"]*)"`,
		"doc": `href="([^"]*\.docx?[^"]*)"`,
		"xls": `href="([^"]*\.xlsx?[^"]*)"`,

		// Archives
		"zip": `href="([^"]*\.zip[^"]*)"`,
		"tar": `href="([^"]*\.tar[^"]*)"`,
		"gz":  `href="([^"]*\.gz[^"]*)"`,

		// Configuration and sensitive files
		"config": `href="([^"]*\.(?:config|conf|ini|yml|yaml|xml)[^"]*)"`,
		"env":    `href="([^"]*\.env[^"]*)"`,
		"sql":    `href="([^"]*\.sql[^"]*)"`,

		// Backup and temporary files
		"backup": `href="([^"]*\.(?:bak|backup|old|tmp|temp)[^"]*)"`,
		"log":    `href="([^"]*\.log[^"]*)"`,

		// Key files and certificates
		"key": `href="([^"]*\.(?:key|pem|crt|cer)[^"]*)"`,
	}

	for fileType, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				// Remove the self-assignment - just use match[1] directly
				files = append(files, ResourceItem{
					Type: fileType,
					URL:  match[1],
					Size: "unknown",
				})
			}
		}
	}
	return files
}

func (l *SecurityLayout) extractImages(html string, baseURL string) []ResourceItem {
	var images []ResourceItem

	// Enhanced image patterns including data URLs and different formats
	patterns := []string{
		`src="([^"]*\.(?:jpg|jpeg|png|gif|webp|svg|bmp|mp4|mkv|webm|avi|mpg|mpeg|wmv|flv|avif|ico)[^"]*)"`,
		`srcset="([^"]*)"`, // For responsive images
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)

		for _, match := range matches {
			if len(match) > 1 {
				// Handle srcset which may contain multiple images - more efficiently
				srcset := match[1]
				start := 0
				for i, r := range srcset {
					if r == ',' {
						if start < i {
							urlPart := strings.TrimSpace(srcset[start:i])
							l.processImageURL(urlPart, baseURL, &images)
						}
						start = i + 1
					}
				}
				// Process the last URL
				if start < len(srcset) {
					urlPart := strings.TrimSpace(srcset[start:])
					l.processImageURL(urlPart, baseURL, &images)
				}
			}
		}
	}

	return images
}

func (l *SecurityLayout) processImageURL(urlPart string, baseURL string, images *[]ResourceItem) {
	// Extract just the URL from srcset (format: "image.jpg 2x")
	url := strings.TrimSpace(strings.Split(urlPart, " ")[0])

	// Skip data URLs and empty URLs
	if strings.HasPrefix(url, "data:") || url == "" {
		return
	}

	// Resolve relative image URLs
	resolvedURL := url
	if strings.HasPrefix(resolvedURL, "/") && !strings.HasPrefix(resolvedURL, "//") {
		base := strings.TrimSuffix(baseURL, "/")
		resolvedURL = base + resolvedURL
	}

	*images = append(*images, ResourceItem{
		Type: "image",
		URL:  resolvedURL,
		Size: "unknown",
	})
}

func (l *SecurityLayout) ScanSiteResources(targetURL string, resourceType string) {
	normalizedURL := NormalizeURL(targetURL)
	l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Scanning %s:[-] %s ", resourceType, normalizedURL))
	l.explorerContent.SetText("Discovering resources...")

	go func() {
		// Create request with proper User-Agent
		req, err := http.NewRequest("GET", normalizedURL, nil)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerContent.SetText(fmt.Sprintf("[red]Request failed:[-]\n%s", err.Error()))
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
			})
			return
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerContent.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
			})
			return
		}
		defer resp.Body.Close()

		html, err := io.ReadAll(resp.Body)
		if err != nil {
			l.explorerContent.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
			l.explorerStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
			return
		}

		l.app.QueueUpdateDraw(func() {
			resources := l.extractResources(string(html), normalizedURL, resourceType)
			l.currentResources = resources
			l.currentBaseURL = normalizedURL

			formatted := l.formatResources(resources, resourceType) // Make sure resourceType is passed here
			l.explorerContent.SetText(formatted)
			l.explorerStatus.SetText(fmt.Sprintf(" [green]Found %d %s:[-] %s ", len(resources), resourceType, normalizedURL))
		})
	}()
}

func (l *SecurityLayout) extractScripts(html string) []ResourceItem {
	var scripts []ResourceItem
	re := regexp.MustCompile(`src="([^"]*\.js[^"]*)"`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 {
			// Use directly:
			scripts = append(scripts, ResourceItem{
				Type: "script",
				URL:  match[1], // Use match[1] directly
				Size: "unknown",
			})
		}
	}
	return scripts
}

func (l *SecurityLayout) extractAllResources(html string, baseURL string) []ResourceItem {
	var all []ResourceItem
	all = append(all, l.extractFiles(html)...)
	all = append(all, l.extractImages(html, baseURL)...) // Fixed call
	all = append(all, l.extractScripts(html)...)         // Single definition now
	all = append(all, l.extractLinks(html, baseURL)...)
	return all
}

func (l *SecurityLayout) DownloadFile(fileURL string, filename string) {
	// Determine download directory
	downloadDir := l.getDownloadDirectory()

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		l.explorerStatus.SetText(fmt.Sprintf(" [red]Cannot create download dir:[-] %s ", err.Error()))
		return
	}

	l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Downloading to %s:[-] %s ", downloadDir, fileURL))

	go func() {
		// Create request with proper headers
		req, err := http.NewRequest("GET", fileURL, nil)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Request failed:[-] %s ", err.Error()))
			})
			return
		}

		// Set proper User-Agent for respectful scraping
		req.Header.Set("User-Agent", "TUI-Web-Browser/1.0 (+https://github.com/your-username/tui-browser)")

		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Download failed:[-] %s ", err.Error()))
			})
			return
		}
		defer resp.Body.Close()

		// Check for rate limiting or access denied
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Access denied (HTTP %d): Check robots.txt[-] ", resp.StatusCode))
			})
			return
		}

		if resp.StatusCode != 200 {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]HTTP Error %d[-] ", resp.StatusCode))
			})
			return
		}

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Read failed:[-] %s ", err.Error()))
			})
			return
		}

		// Extract filename if not provided
		if filename == "" {
			if lastSlash := strings.LastIndex(fileURL, "/"); lastSlash != -1 {
				filename = fileURL[lastSlash+1:]
			} else {
				filename = fileURL
			}
		}

		// Create full path in Download folder
		fullPath := filepath.Join(downloadDir, filename)

		// Save file
		err = os.WriteFile(fullPath, body, 0o644)

		// Now update the UI
		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.explorerStatus.SetText(fmt.Sprintf(" [red]Save failed:[-] %s ", err.Error()))
			} else {
				l.explorerStatus.SetText(fmt.Sprintf(" [green]Downloaded to:[-] %s ", fullPath))
			}
		})
	}()
} // Add this method to get the download directory
func (l *SecurityLayout) getDownloadDirectory() string {
	// Try XDG_DOWNLOAD_DIR first (standard Linux)
	if xdgDownload := os.Getenv("XDG_DOWNLOAD_DIR"); xdgDownload != "" {
		return xdgDownload
	}

	// Try HOME/Downloads (fallback)
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, "Downloads")
	}

	// Final fallback: current directory
	return "."
}

func (l *SecurityLayout) formatResources(resources []ResourceItem, resourceType string) string {
	if len(resources) == 0 {
		return "[yellow]No " + resourceType + " found[-]"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "[white]Found %d %s:[-]\n\n", len(resources), resourceType)

	for i, resource := range resources {
		fmt.Fprintf(&builder, "[%d] [blue]%s[-]\n", i+1, resource.URL)
		fmt.Fprintf(&builder, "    Type: %s | Size: %s\n\n", resource.Type, resource.Size)
	}

	// Add usage instructions
	fmt.Fprintf(&builder, "[yellow]Quick actions:[-]\n")
	fmt.Fprintf(&builder, "[::b]d 13[::-] - Download item 13\n")
	fmt.Fprintf(&builder, "[::b]:view 13[::-] - View content of item 13\n")
	fmt.Fprintf(&builder, "[::b]Press 1-9[::-] - Quick visit item 1-9\n")
	fmt.Fprintf(&builder, "[::b]bulk 1,3,5[::-] - Bulk download items\n")

	return builder.String()
}

func (l *SecurityLayout) extractLinks(html string, baseURL string) []ResourceItem {
	var links []ResourceItem
	re := regexp.MustCompile(`href="([^"]*)"`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, match := range matches {
		if len(match) > 1 && !strings.HasPrefix(match[1], "#") && !strings.HasPrefix(match[1], "javascript:") {
			// Categorize links as internal/external
			linkType := "external"
			if strings.HasPrefix(match[1], "/") || strings.Contains(match[1], baseURL) {
				linkType = "internal"
			}

			links = append(links, ResourceItem{
				Type: linkType,
				URL:  match[1],
				Size: "unknown",
			})
		}
	}
	return links
}

func (l *SecurityLayout) SwitchToSecurityMode() {
	if l.securityFlex == nil {
		fmt.Println("ERROR: securityFlex is nil - calling setupSecurityLayout")
		l.setupSecurityLayout()
	}

	l.currentMode = ModeSecurity
	l.currentRoot = l.securityFlex
	l.app.SetRoot(l.securityFlex, true)
	l.app.SetFocus(l.securityInput)
	l.securityStatus.SetText(" [yellow]🔒 Security Scanner[-] | F1: Browser | F3: Explorer ")
	l.updateSecurityPanelFocus(-1)
}

func (l *SecurityLayout) setupBrowserLayout() {
	// Recreate original browser layout
	l.browserContent = tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			l.app.Draw()
		}).
		SetRegions(true).
		SetScrollable(true)
	l.browserContent.SetBorder(true).SetTitle(" Web Content ")
	l.browserContent.SetBorderPadding(0, 0, 1, 1)

	l.browserInput = tview.NewInputField().
		SetLabel("🌐 ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	l.browserInput.SetBorder(true).SetTitle(" Address Bar ")

	l.browserStatus = tview.NewTextView().SetDynamicColors(true)
	l.browserStatus.SetText(" Ready ")
	l.browserStatus.SetBackgroundColor(tcell.ColorDarkOliveGreen)

	// Original browser flex layout
	l.browserFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(l.browserContent, 0, 1, false).
		AddItem(l.browserInput, 3, 1, true).
		AddItem(l.browserStatus, 1, 1, false)

	// === FIX: Initialize browser handlers FIRST and verify ===
	l.browserHandlers = NewUIHandlersWithConfig(l.app, l.config)
	if l.browserHandlers == nil {
		panic("browserHandlers is nil after initialization!")
	}
	if l.browserHandlers.contentManager == nil {
		panic("browserHandlers.contentManager is nil!")
	}

	// ADD THIS LINE - Display the initial welcome message
	DisplayInitialMessage(l.browserContent, l.config)

	// NOW set up content input capture (after handlers are initialized)
	contentInputCapture := CreateContentInputCapture(
		l.browserHandlers,
		l.browserInput,
		l.browserContent,
		l.browserStatus,
		l.app,
		func() {
			// updateFocus function - could update border color or status if needed
			l.browserContent.SetBorderColor(tcell.ColorYellow)
		},
	)

	// Apply the input capture to content
	l.browserContent.SetInputCapture(contentInputCapture)

	// Global app input capture for everything in browser mode
	l.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle mode switching globally
		switch event.Key() {
		case tcell.KeyF1:
			switch l.currentMode {
			case ModeSecurity:
				l.SwitchToBrowserMode()
				return nil
			case ModeExplorer:
				l.SwitchToBrowserMode()
				return nil
			}
		case tcell.KeyF2:
			switch l.currentMode {
			case ModeBrowser:
				l.SwitchToSecurityMode()
				return nil
			case ModeExplorer:
				l.SwitchToSecurityMode()
				return nil
			}
		case tcell.KeyF3:
			switch l.currentMode {
			case ModeBrowser:
				l.SwitchToExplorerMode()
				return nil
			case ModeSecurity:
				l.SwitchToExplorerMode()
				return nil
			}
		}

		// Handle navigation in browser mode regardless of focus
		if l.currentMode == ModeBrowser {
			switch event.Key() {
			case tcell.KeyUp:
				row, _ := l.browserContent.GetScrollOffset()
				newRow := max(0, row-1)
				l.browserContent.ScrollTo(newRow, 0)
				return nil
			case tcell.KeyDown:
				row, _ := l.browserContent.GetScrollOffset()
				l.browserContent.ScrollTo(row+1, 0)
				return nil
			case tcell.KeyPgUp:
				row, _ := l.browserContent.GetScrollOffset()
				newRow := max(0, row-10)
				l.browserContent.ScrollTo(newRow, 0)
				return nil
			case tcell.KeyPgDn:
				row, _ := l.browserContent.GetScrollOffset()
				l.browserContent.ScrollTo(row+10, 0)
				return nil
			case tcell.KeyHome:
				l.browserContent.ScrollToBeginning()
				return nil
			case tcell.KeyEnd:
				l.browserContent.ScrollToEnd()
				return nil
			case tcell.KeyLeft:
				// Add nil check for safety
				if l.browserHandlers != nil && l.browserHandlers.contentManager != nil {
					l.browserHandlers.HandleBackNavigation(l.browserContent, l.browserStatus)
				}
				return nil
			case tcell.KeyRight:
				// Add nil check for safety
				if l.browserHandlers != nil && l.browserHandlers.contentManager != nil {
					l.browserHandlers.HandleForwardNavigation(l.browserContent, l.browserStatus)
				}
				return nil
			}
		}

		return event
	})

	// Browser input field - only handle its own input, no focus switching
	l.browserInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyF2 {
			l.SwitchToSecurityMode()
			return nil
		}
		return event
	})

	l.setupBrowserCommands()
}

func (l *SecurityLayout) setupBrowserCommands() {
	// Use your original browser input handler with safety check
	inputHandler := CreateInputHandler(
		l.browserHandlers,
		l.browserInput,
		l.browserContent,
		l.browserStatus,
	)

	l.browserInput.SetDoneFunc(func(key tcell.Key) {
		// Only handle input if handlers are initialized
		if l.browserHandlers != nil && l.browserHandlers.contentManager != nil {
			inputHandler(key)
		} else {
			l.browserStatus.SetText(" [red]Browser not ready - please wait[-] ")
		}
	})
}

func (l *SecurityLayout) SwitchToExplorerMode() {
	l.currentMode = ModeExplorer
	l.currentRoot = l.explorerFlex
	l.app.SetRoot(l.explorerFlex, true)
	l.app.SetFocus(l.explorerInput)

	// Set initial border colors to show focus
	l.explorerInput.SetBorderColor(tcell.ColorYellow)
	l.explorerSections.SetBorderColor(tcell.ColorWhite)
	l.explorerContent.SetBorderColor(tcell.ColorWhite)

	l.explorerStatus.SetText(
		" [blue]🔍 Site Explorer[-] | F1: Browser | F2: Security | Tab: Navigate (Input → Sections → Content) ",
	)

	// Auto-populate with current site if available
	currentURL := l.getCurrentURL()
	if currentURL != "" {
		l.explorerInput.SetText(":files " + currentURL)
	}
}

func (l *SecurityLayout) setupSecurityLayout() {
	// Security panels
	l.techPanel = createPanel("Technology Stack")
	l.vulnPanel = createPanel("Vulnerabilities")
	l.headersPanel = createPanel("Security Headers")
	l.requestsPanel = createPanel("Requests")
	l.securityInput = createInputField()
	l.securityStatus = createStatusBar()

	securityPanels := []*tview.TextView{l.techPanel, l.vulnPanel, l.headersPanel, l.requestsPanel}

	// Set up each panel with proper closure
	for i := range securityPanels {
		panel := securityPanels[i] // Local variable for closure
		panel.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			// Handle scrolling
			switch event.Key() {
			case tcell.KeyUp:
				row, _ := panel.GetScrollOffset()
				newRow := max(0, row-1)
				panel.ScrollTo(newRow, 0)
				return nil
			case tcell.KeyDown:
				row, _ := panel.GetScrollOffset()
				panel.ScrollTo(row+1, 0)
				return nil
			case tcell.KeyPgUp:
				row, _ := panel.GetScrollOffset()
				newRow := max(0, row-10)
				panel.ScrollTo(newRow, 0)
				return nil
			case tcell.KeyPgDn:
				row, _ := panel.GetScrollOffset()
				panel.ScrollTo(row+10, 0)
				return nil
			case tcell.KeyHome:
				panel.ScrollToBeginning()
				return nil
			case tcell.KeyEnd:
				panel.ScrollToEnd()
				return nil
			case tcell.KeyTab:
				// Find current panel index and focus next
				currentIndex := -1
				for idx, p := range securityPanels {
					if p == panel {
						currentIndex = idx
						break
					}
				}
				if currentIndex != -1 {
					nextIndex := (currentIndex + 1) % len(securityPanels)
					l.app.SetFocus(securityPanels[nextIndex])
					l.updateSecurityPanelFocus(nextIndex)
				}
				return nil
			case tcell.KeyEscape:
				// Escape to go back to security input field
				l.app.SetFocus(l.securityInput)
				l.updateSecurityPanelFocus(-1) // Clear panel focus indicators
				return nil
			case tcell.KeyF1:
				l.SwitchToBrowserMode()
				return nil
			}
			return event
		})
	}

	// Set input capture for security input field
	l.securityInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			// Tab from input to first panel
			l.app.SetFocus(securityPanels[0])
			l.updateSecurityPanelFocus(0)
			return nil
		}
		if event.Key() == tcell.KeyF1 {
			l.SwitchToBrowserMode()
			return nil
		}
		return event
	})

	// 4-panel security layout
	topRow := tview.NewFlex().
		AddItem(l.techPanel, 0, 1, false).
		AddItem(l.vulnPanel, 0, 1, false)
	bottomRow := tview.NewFlex().
		AddItem(l.headersPanel, 0, 1, false).
		AddItem(l.requestsPanel, 0, 1, false)
	content := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topRow, 0, 1, false).
		AddItem(bottomRow, 0, 1, false)

	l.securityFlex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, false).
		AddItem(l.securityInput, 3, 1, true).
		AddItem(l.securityStatus, 1, 1, false)

		// === ADD THIS: Global security mode input capture ===
	l.securityFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyF1:
			l.SwitchToBrowserMode()
			return nil
		case tcell.KeyF2:
			// Already in security, do nothing
			return nil
		case tcell.KeyF3:
			l.SwitchToExplorerMode()
			return nil
		}
		return event
	})
	// === END ADDED CODE ===

	l.setupSecurityCommands()
}

func (l *SecurityLayout) updateSecurityPanelFocus(activePanel int) {
	panels := []*tview.TextView{l.techPanel, l.vulnPanel, l.headersPanel, l.requestsPanel}
	titles := []string{"Technology Stack", "Vulnerabilities", "Security Headers", "Requests"}

	for i, panel := range panels {
		if i == activePanel {
			panel.SetTitle(" [yellow]▶ " + titles[i] + "[-] ")
		} else {
			panel.SetTitle(" " + titles[i] + " ")
		}
	}
}

func createPanel(title string) *tview.TextView {
	panel := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	panel.SetBorder(true).SetTitle(title)
	panel.SetBorderPadding(0, 0, 1, 1)
	return panel
}

func createInputField() *tview.InputField {
	input := tview.NewInputField().
		SetLabel("🔒 ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDarkSlateGray)
	input.SetBorder(true).SetTitle(" Security Commands ")
	return input
}

func createStatusBar() *tview.TextView {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetText(" Ready ")
	status.SetBackgroundColor(tcell.ColorDarkOliveGreen)
	return status
}

func (l *SecurityLayout) SwitchToBrowserMode() {
	if l.browserFlex == nil {
		fmt.Println("CRITICAL: browserFlex is nil, attempting recovery...")
		l.setupBrowserLayout()

		// Check if it worked
		if l.browserFlex == nil {
			panic("browserFlex is still nil after recovery attempt!")
		}
	}

	l.currentMode = ModeBrowser
	l.currentRoot = l.browserFlex
	l.app.SetRoot(l.browserFlex, true)

	// Always keep address bar focused
	l.app.SetFocus(l.browserInput)
	l.browserStatus.SetText(
		" [green]🌐 Browser Mode[-] | Arrows: Scroll/History | F2: Security | F3: Explorer ",
	)
}

func (l *SecurityLayout) setupSecurityCommands() {
	l.securityInput.SetDoneFunc(func(key tcell.Key) {
		text := strings.TrimSpace(l.securityInput.GetText())
		if text == "" {
			return
		}

		if strings.HasPrefix(text, ":robots ") {
			target := strings.TrimSpace(text[len(":robots "):])
			l.securityInput.SetText("")
			l.ScanRobotsTxt(target)
			return
		}

		if strings.HasPrefix(text, ":tech ") {
			target := strings.TrimSpace(text[len(":tech "):])
			l.securityInput.SetText("")
			l.ScanTechnology(target)
			return
		}

		if strings.HasPrefix(text, ":headers ") {
			target := strings.TrimSpace(text[len(":headers "):])
			l.securityInput.SetText("")
			l.ScanSecurityHeaders(target)
			return
		}

		if strings.HasPrefix(text, ":vuln ") {
			target := strings.TrimSpace(text[len(":vuln "):])
			l.securityInput.SetText("")
			l.ScanVulnerabilities(target)
			return
		}

		if strings.HasPrefix(text, ":scan ") {
			target := strings.TrimSpace(text[len(":scan "):])
			l.securityInput.SetText("")
			l.FullScan(target)
			return
		}

		if strings.HasPrefix(text, ":export ") {
			parts := strings.SplitN(text, " ", 3)
			if len(parts) >= 3 {
				format := parts[1]
				target := parts[2]
				l.securityInput.SetText("")
				l.ExportScan(target, format)
			} else {
				l.securityStatus.SetText(" [red]Usage: :export <format> <url>[-] ")
			}
			return
		}

		if text == ":browser" || text == ":b" {
			l.securityInput.SetText("")
			l.SwitchToBrowserMode()
			return
		}

		l.securityInput.SetText("")
		l.securityStatus.SetText(" [red]Unknown security command[-] ")
	})
}

// Your security scanning methods...
func (l *SecurityLayout) ScanRobotsTxt(targetURL string) {
	normalizedURL := NormalizeURL(targetURL)
	l.securityStatus.SetText(fmt.Sprintf(" [yellow]Scanning robots.txt:[-] %s ", normalizedURL))
	l.requestsPanel.SetText("Fetching robots.txt...")

	go func() {
		robots, err := security.FetchRobotsTxt(normalizedURL)
		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.requestsPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.securityStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
				return
			}
			formatted := robots.FormatForPanel()
			l.requestsPanel.SetText(formatted)
			l.securityStatus.SetText(fmt.Sprintf(" [green]Found robots.txt:[-] %s ", normalizedURL))
		})
	}()
}

func (l *SecurityLayout) ScanTechnology(targetURL string) {
	normalizedURL := NormalizeURL(targetURL)
	l.securityStatus.SetText(fmt.Sprintf(" [yellow]Fingerprinting:[-] %s ", normalizedURL))
	l.techPanel.SetText("Scanning for technologies...")

	go func() {
		scanner := security.NewSecurityScanner()
		html, headers, err := scanner.FetchWithHeaders(normalizedURL)
		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.techPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.securityStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
				return
			}
			fingerprint := security.FingerprintTechnology(normalizedURL, html, headers)
			formatted := fingerprint.FormatForPanel()
			l.techPanel.SetText(formatted)
			l.securityStatus.SetText(fmt.Sprintf(" [green]Tech identified:[-] %s ", normalizedURL))
		})
	}()
}

func (l *SecurityLayout) handleDownloadByIndex(indexStr string) {
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		l.explorerStatus.SetText(" [red]Invalid index[-] ")
		return
	}

	if index < 1 || index > len(l.currentResources) {
		l.explorerStatus.SetText(fmt.Sprintf(" [red]Index out of range (1-%d)[-] ", len(l.currentResources)))
		return
	}

	resource := l.currentResources[index-1]

	// Resolve relative URLs
	fullURL, err := l.browserHandlers.contentManager.resolver.ResolveURL(resource.URL, l.currentBaseURL)
	if err != nil {
		l.explorerStatus.SetText(fmt.Sprintf(" [red]Error resolving URL:[-] %s ", err.Error()))
		return
	}

	// Extract filename from URL more efficiently
	filename := ""
	if lastSlash := strings.LastIndex(resource.URL, "/"); lastSlash != -1 && lastSlash < len(resource.URL)-1 {
		filename = resource.URL[lastSlash+1:]
	}
	if filename == "" {
		filename = fmt.Sprintf("download_%d", index)
	}

	downloadDir := l.getDownloadDirectory()
	l.explorerStatus.SetText(fmt.Sprintf(" [yellow]Downloading #[%d] to %s:[-] %s ", index, downloadDir, filename))
	l.DownloadFile(fullURL, filename)
}

func (l *SecurityLayout) ScanSecurityHeaders(targetURL string) {
	normalizedURL := NormalizeURL(targetURL)
	l.securityStatus.SetText(fmt.Sprintf(" [yellow]Analyzing headers:[-] %s ", normalizedURL))
	l.headersPanel.SetText("Checking security headers...")

	go func() {
		scanner := security.NewSecurityScanner()
		html, headers, err := scanner.FetchWithHeaders(normalizedURL)
		_ = html
		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.headersPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.securityStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
				return
			}
			securityHeaders := security.AnalyzeSecurityHeaders(normalizedURL, headers)
			formatted := securityHeaders.FormatForPanel()
			l.headersPanel.SetText(formatted)
			l.securityStatus.SetText(fmt.Sprintf(" [green]Headers analyzed:[-] %s ", normalizedURL))
		})
	}()
}

func (l *SecurityLayout) ScanVulnerabilities(targetURL string) {
	normalizedURL := NormalizeURL(targetURL)
	l.securityStatus.SetText(
		fmt.Sprintf(" [yellow]Scanning vulnerabilities:[-] %s ", normalizedURL),
	)
	l.vulnPanel.SetText("Running vulnerability checks...")

	go func() {
		scanner := security.NewSecurityScanner()
		html, headers, err := scanner.FetchWithHeaders(normalizedURL)
		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.vulnPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.securityStatus.SetText(fmt.Sprintf(" [red]Failed:[-] %s ", normalizedURL))
				return
			}
			vulnScan := security.ScanForVulnerabilities(normalizedURL, html, headers)
			formatted := vulnScan.FormatForPanel()
			l.vulnPanel.SetText(formatted)
			l.securityStatus.SetText(
				fmt.Sprintf(" [green]Vuln scan complete:[-] %s ", normalizedURL),
			)
		})
	}()
}

func (l *SecurityLayout) FullScan(targetURL string) {
	normalizedURL := NormalizeURL(targetURL)
	l.securityStatus.SetText(fmt.Sprintf(" [yellow]Full security scan:[-] %s ", normalizedURL))

	l.techPanel.SetText("Scanning technology stack...")
	l.headersPanel.SetText("Analyzing security headers...")
	l.vulnPanel.SetText("Checking for vulnerabilities...")
	l.requestsPanel.SetText("Fetching robots.txt...")

	go func() {
		scanner := security.NewSecurityScanner()
		html, headers, err := scanner.FetchWithHeaders(normalizedURL)
		l.app.QueueUpdateDraw(func() {
			if err != nil {
				l.securityStatus.SetText(fmt.Sprintf(" [red]Scan failed:[-] %s ", normalizedURL))
				l.techPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.headersPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.vulnPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				l.requestsPanel.SetText(fmt.Sprintf("[red]Error:[-]\n%s", err.Error()))
				return
			}

			fingerprint := security.FingerprintTechnology(normalizedURL, html, headers)
			l.techPanel.SetText(fingerprint.FormatForPanel())

			securityHeaders := security.AnalyzeSecurityHeaders(normalizedURL, headers)
			l.headersPanel.SetText(securityHeaders.FormatForPanel())

			vulnScan := security.ScanForVulnerabilities(normalizedURL, html, headers)
			l.vulnPanel.SetText(vulnScan.FormatForPanel())

			go func() {
				robots, err := security.FetchRobotsTxt(normalizedURL)
				l.app.QueueUpdateDraw(func() {
					if err != nil {
						l.requestsPanel.SetText(
							fmt.Sprintf("[red]Error fetching robots.txt:[-]\n%s", err.Error()),
						)
					} else if robots != nil {
						l.requestsPanel.SetText(robots.FormatForPanel())
					}
				})
			}()

			l.securityStatus.SetText(
				fmt.Sprintf(" [green]Full scan complete:[-] %s ", normalizedURL),
			)
		})
	}()
}

func (l *SecurityLayout) ExportScan(targetURL string, format string) {
	normalizedURL := NormalizeURL(targetURL)
	l.securityStatus.SetText(fmt.Sprintf(" [yellow]Exporting:[-] %s ", normalizedURL))

	go func() {
		scanner := security.NewSecurityScanner()
		html, headers, err := scanner.FetchWithHeaders(normalizedURL)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.securityStatus.SetText(fmt.Sprintf(" [red]Export failed:[-] %s ", normalizedURL))
			})
			return
		}

		tech := security.FingerprintTechnology(normalizedURL, html, headers)
		headersAnalysis := security.AnalyzeSecurityHeaders(normalizedURL, headers)
		vulns := security.ScanForVulnerabilities(normalizedURL, html, headers)
		robots, _ := security.FetchRobotsTxt(normalizedURL)

		report := security.GenerateScanReport(normalizedURL, tech, headersAnalysis, vulns, robots)

		parsed, err := url.Parse(normalizedURL)
		if err != nil {
			l.app.QueueUpdateDraw(func() {
				l.securityStatus.SetText(fmt.Sprintf(" [red]Invalid URL:[-] %s ", err.Error()))
			})
			return
		}

		domain := parsed.Host
		timestamp := "export" // Simplified for now

		var filename string
		var exportErr error

		if format == "json" {
			filename = fmt.Sprintf("scans/%s_%s.json", domain, timestamp)
			exportErr = report.SaveAsJSON(filename)
		} else {
			filename = fmt.Sprintf("scans/%s_%s.txt", domain, timestamp)
			exportErr = report.SaveAsText(filename)
		}

		l.app.QueueUpdateDraw(func() {
			if exportErr != nil {
				l.securityStatus.SetText(
					fmt.Sprintf(" [red]Export error:[-] %s ", exportErr.Error()),
				)
			} else {
				l.securityStatus.SetText(fmt.Sprintf(" [green]Exported to:[-] %s ", filename))
			}
		})
	}()
}
