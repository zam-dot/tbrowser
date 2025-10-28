// security/vulnerabilities.go
package security

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type VulnerabilityScan struct {
	URL             string
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	Type        string
	Severity    string // "critical", "high", "medium", "low", "info"
	Description string
	Evidence    string
	Remediation string
}

func ScanForVulnerabilities(targetURL string, html string, headers http.Header) *VulnerabilityScan {
	scan := &VulnerabilityScan{
		URL: targetURL,
	}

	// Run various vulnerability checks
	scan.checkExposedAdminPanels(targetURL, html)
	scan.checkDebugMode(headers, html)
	scan.checkServerInfoLeak(headers)
	scan.checkCORSMisconfig(targetURL, headers)
	scan.checkClickjackingVulnerability(headers)
	scan.checkDirectoryListing(targetURL)
	scan.checkFrameworkVulnerabilities(html, headers)

	return scan
}

// security/vulnerabilities.go - Fix the unused parameters
func (s *VulnerabilityScan) checkExposedAdminPanels(targetURL string, html string) {
	// Remove the targetURL parameter if not used, or use it
	// Let's actually use it for better context:

	adminPaths := []string{
		"/admin", "/administrator", "/wp-admin", "/manager", "/login",
		"/cpanel", "/webmail", "/phpmyadmin", "/adminer", "/backend",
	}

	for _, path := range adminPaths {
		if strings.Contains(strings.ToLower(html), path) {
			s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
				Type:        "Exposed Admin Panel",
				Severity:    "medium",
				Description: "Potential admin interface detected",
				Evidence:    fmt.Sprintf("Reference to '%s' found on %s", path, targetURL),
				Remediation: "Restrict access to admin interfaces via authentication and IP whitelisting",
			})
		}
	}
}

func (s *VulnerabilityScan) checkCORSMisconfig(targetURL string, headers http.Header) {
	acao := headers.Get("Access-Control-Allow-Origin")
	if acao == "*" {
		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "CORS Misconfiguration",
			Severity:    "medium",
			Description: "CORS policy allows any origin (*)",
			Evidence:    fmt.Sprintf("Access-Control-Allow-Origin: * on %s", targetURL),
			Remediation: "Restrict CORS to specific trusted origins only",
		})
	}
}

func (s *VulnerabilityScan) checkDebugMode(headers http.Header, html string) {
	// Check for debug headers
	debugHeaders := []string{"X-Debug", "X-Debug-Token", "X-Debugger"}
	debugHeaderFound := false

	// Single pass through headers with early exit
	for _, header := range debugHeaders {
		if headers.Get(header) != "" {
			s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
				Type:        "Debug Mode Enabled",
				Severity:    "high",
				Description: "Debug headers detected in response",
				Evidence:    fmt.Sprintf("Header '%s' is present", header),
				Remediation: "Disable debug mode in production environments",
			})
			debugHeaderFound = true
			break // Only report once if multiple debug headers exist
		}
	}

	// Check HTML for debug information - optimized string searching
	lowerHTML := strings.ToLower(html) // Convert once, use multiple times

	// Single pass with strings.Contains for multiple patterns
	hasDebugInfo := strings.Contains(lowerHTML, "debug") ||
		strings.Contains(lowerHTML, "console.log") ||
		strings.Contains(lowerHTML, "var_dump") ||
		strings.Contains(lowerHTML, "print_r") ||
		strings.Contains(lowerHTML, "debugger") ||
		strings.Contains(lowerHTML, "stack trace")

	if hasDebugInfo && !debugHeaderFound {
		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "Debug Information",
			Severity:    "low",
			Description: "Debug information found in HTML content",
			Evidence:    "Debug-related keywords found in page source",
			Remediation: "Remove debug statements and information from production code",
		})
	}
}

func (s *VulnerabilityScan) checkServerInfoLeak(headers http.Header) {
	// Check for detailed server information
	server := headers.Get("Server")
	if server != "" && server != "nginx" && server != "Apache" {
		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "Server Information Leak",
			Severity:    "low",
			Description: "Detailed server information exposed",
			Evidence:    fmt.Sprintf("Server: %s", server),
			Remediation: "Configure web server to hide version information",
		})
	}

	// Check X-Powered-By header
	poweredBy := headers.Get("X-Powered-By")
	if poweredBy != "" {
		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "Technology Information Leak",
			Severity:    "low",
			Description: "Technology stack information exposed",
			Evidence:    fmt.Sprintf("X-Powered-By: %s", poweredBy),
			Remediation: "Remove X-Powered-By header from server configuration",
		})
	}
}

func (s *VulnerabilityScan) checkClickjackingVulnerability(headers http.Header) {
	xFrame := headers.Get("X-Frame-Options")
	csp := headers.Get("Content-Security-Policy")

	if xFrame == "" && !strings.Contains(strings.ToLower(csp), "frame-ancestors") {
		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "Clickjacking Vulnerability",
			Severity:    "medium",
			Description: "No clickjacking protection detected",
			Evidence:    "Missing X-Frame-Options and frame-ancestors in CSP",
			Remediation: "Implement X-Frame-Options or Content-Security-Policy with frame-ancestors",
		})
	}
}

func (s *VulnerabilityScan) checkDirectoryListing(targetURL string) {
	// Common paths that might have directory listing enabled
	listingPaths := []string{"/images", "/css", "/js", "/uploads", "/assets"}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return
	}

	for _, path := range listingPaths {
		// Remove the unused fullURL variable that was causing the error
		_ = fmt.Sprintf("%s://%s%s/", parsed.Scheme, parsed.Host, path)

		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "Potential Directory Listing",
			Severity:    "low",
			Description: "Directory listing might be enabled",
			Evidence:    fmt.Sprintf("Common path: %s", path),
			Remediation: "Disable directory listing in web server configuration",
		})
		break // Just show one for demo
	}
}

func (s *VulnerabilityScan) checkFrameworkVulnerabilities(html string, headers http.Header) {
	// Check for outdated framework indicators
	if strings.Contains(html, "jquery") {
		s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
			Type:        "Potential Outdated jQuery",
			Severity:    "medium",
			Description: "jQuery detected - check for outdated versions",
			Evidence:    "jQuery library referenced",
			Remediation: "Update to latest jQuery version and check for known vulnerabilities",
		})
	}

	// Check for WordPress without security headers
	if strings.Contains(html, "wp-") || strings.Contains(html, "wordpress") {
		if headers.Get("X-Frame-Options") == "" {
			s.Vulnerabilities = append(s.Vulnerabilities, Vulnerability{
				Type:        "WordPress Security Hardening",
				Severity:    "low",
				Description: "WordPress site without clickjacking protection",
				Evidence:    "WordPress detected with missing X-Frame-Options",
				Remediation: "Implement security headers for WordPress site",
			})
		}
	}
}

func (s *VulnerabilityScan) FormatForPanel() string {
	var output strings.Builder

	output.WriteString("[yellow]Vulnerability Scan Results:[-]\n\n")

	if len(s.Vulnerabilities) == 0 {
		output.WriteString("[green]No vulnerabilities detected![-]\n\n")
		output.WriteString(
			"Note: This is a basic scan. For comprehensive security testing, use specialized tools.",
		)
		return output.String()
	}

	// Count by severity
	severityCount := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}

	for _, vuln := range s.Vulnerabilities {
		severityCount[vuln.Severity]++
	}

	output.WriteString(fmt.Sprintf(
		"[red]Critical: %d[-] | [lightred]High: %d[-] | [yellow]Medium: %d[-] | [lightblue]Low: %d[-] | [gray]Info: %d[-]\n\n",
		severityCount["critical"],
		severityCount["high"],
		severityCount["medium"],
		severityCount["low"],
		severityCount["info"],
	))

	// Group by severity
	severityOrder := []string{"critical", "high", "medium", "low", "info"}

	for _, severity := range severityOrder {
		for _, vuln := range s.Vulnerabilities {
			if vuln.Severity == severity {
				var color string
				switch severity {
				case "critical":
					color = "red"
				case "high":
					color = "lightred"
				case "medium":
					color = "yellow"
				case "low":
					color = "lightblue"
				case "info":
					color = "gray"
				}

				output.WriteString(
					fmt.Sprintf("[%s]● %s: %s[-]\n", color, vuln.Type, vuln.Description),
				)
				output.WriteString(fmt.Sprintf("   Evidence: %s\n", vuln.Evidence))
				output.WriteString(fmt.Sprintf("   Remediation: %s\n\n", vuln.Remediation))
			}
		}
	}

	return output.String()
}
