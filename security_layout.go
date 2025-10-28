// security_layout.go - Fixed version
package main

import (
	"browser/security"
	"fmt"
	"net/url"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppMode int

const (
	ModeBrowser AppMode = iota
	ModeSecurity
)

type SecurityLayout struct {
	app             *tview.Application
	securityFlex    *tview.Flex
	browserFlex     *tview.Flex
	currentRoot     tview.Primitive
	currentMode     AppMode
	config          *Config
	browserHandlers *UIHandlers

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

	// Initialize browser handlers FIRST
	l.browserHandlers = NewUIHandlersWithConfig(l.app, l.config)

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
		// Handle mode switching
		if event.Key() == tcell.KeyF2 && l.currentMode == ModeBrowser {
			l.SwitchToSecurityMode()
			return nil
		}
		if event.Key() == tcell.KeyF1 && l.currentMode == ModeSecurity {
			l.SwitchToBrowserMode()
			return nil
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
				l.browserHandlers.HandleBackNavigation(l.browserContent, l.browserStatus)
				return nil
			case tcell.KeyRight:
				l.browserHandlers.HandleForwardNavigation(l.browserContent, l.browserStatus)
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

	// Initialize browser handlers
	l.browserHandlers = NewUIHandlersWithConfig(l.app, l.config)
	l.setupBrowserCommands()
}

func CreateSecurityLayout(app *tview.Application, config *Config) *SecurityLayout {
	layout := &SecurityLayout{
		app:    app,
		config: config,
	}

	layout.setupBrowserLayout()
	layout.setupSecurityLayout()
	layout.SwitchToBrowserMode()

	return layout
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
	l.currentMode = ModeBrowser
	l.currentRoot = l.browserFlex
	l.app.SetRoot(l.browserFlex, true)

	// Always keep address bar focused
	l.app.SetFocus(l.browserInput)
	l.browserStatus.SetText(
		" [green]🌐 Browser Mode[-] | Arrows: Scroll/History | F2: Security Scanner ",
	)
}

func (l *SecurityLayout) SwitchToSecurityMode() {
	l.currentMode = ModeSecurity
	l.currentRoot = l.securityFlex
	l.app.SetRoot(l.securityFlex, true)
	l.app.SetFocus(l.securityInput)
	l.securityStatus.SetText(" [yellow]🔒 Security Scanner[-] | F1: Browser ")
	// Reset panel focus indicators
	l.updateSecurityPanelFocus(-1)
}

func (l *SecurityLayout) setupBrowserCommands() {
	// Use your original browser input handler
	inputHandler := CreateInputHandler(
		l.browserHandlers,
		l.browserInput,
		l.browserContent,
		l.browserStatus,
	)

	l.browserInput.SetDoneFunc(inputHandler)

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
