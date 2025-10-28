// Security scan reporting and export functionality.
// Generates comprehensive security reports in JSON and text formats.
// Consolidates findings from all security analysis components into unified reports.
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ScanReport struct {
	URL             string             `json:"url"`
	Timestamp       time.Time          `json:"timestamp"`
	Technology      *TechFingerprint   `json:"technology,omitempty"`
	Headers         *SecurityHeaders   `json:"headers,omitempty"`
	Vulnerabilities *VulnerabilityScan `json:"vulnerabilities,omitempty"`
	Robots          *RobotsTxt         `json:"robots,omitempty"`
}

func GenerateScanReport(
	url string,
	tech *TechFingerprint,
	headers *SecurityHeaders,
	vulns *VulnerabilityScan,
	robots *RobotsTxt,
) *ScanReport {
	return &ScanReport{
		URL:             url,
		Timestamp:       time.Now(),
		Technology:      tech,
		Headers:         headers,
		Vulnerabilities: vulns,
		Robots:          robots,
	}
}

func (r *ScanReport) SaveAsJSON(filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

func (r *ScanReport) SaveAsText(filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var output strings.Builder

	output.WriteString("SECURITY SCAN REPORT\n")
	output.WriteString(strings.Repeat("=", 50) + "\n")
	output.WriteString(fmt.Sprintf("URL: %s\n", r.URL))
	output.WriteString(fmt.Sprintf("Date: %s\n", r.Timestamp.Format("2006-01-02 15:04:05")))
	output.WriteString("\n")

	// Technology Stack
	if r.Technology != nil {
		output.WriteString("TECHNOLOGY STACK\n")
		output.WriteString(strings.Repeat("-", 50) + "\n")
		output.WriteString(fmt.Sprintf("Server: %s\n", r.Technology.Server))
		output.WriteString(fmt.Sprintf("Framework: %s\n", r.Technology.Framework))

		if len(r.Technology.JavaScript) > 0 {
			output.WriteString(
				fmt.Sprintf("JavaScript: %s\n", strings.Join(r.Technology.JavaScript, ", ")),
			)
		}
		if len(r.Technology.CSS) > 0 {
			output.WriteString(
				fmt.Sprintf("CSS Frameworks: %s\n", strings.Join(r.Technology.CSS, ", ")),
			)
		}
		if len(r.Technology.Analytics) > 0 {
			output.WriteString(
				fmt.Sprintf("Analytics: %s\n", strings.Join(r.Technology.Analytics, ", ")),
			)
		}
		if len(r.Technology.CDN) > 0 {
			output.WriteString(fmt.Sprintf("CDN: %s\n", strings.Join(r.Technology.CDN, ", ")))
		}
		output.WriteString("\n")
	}

	// Security Headers
	if r.Headers != nil && len(r.Headers.SecurityFindings) > 0 {
		output.WriteString("SECURITY HEADERS\n")
		output.WriteString(strings.Repeat("-", 50) + "\n")

		presentCount := 0
		missingCount := 0
		weakCount := 0

		for _, finding := range r.Headers.SecurityFindings {
			switch finding.Status {
			case "present":
				presentCount++
			case "missing":
				missingCount++
			case "weak":
				weakCount++
			}
		}

		output.WriteString(
			fmt.Sprintf(
				"Present: %d | Missing: %d | Weak: %d\n\n",
				presentCount,
				missingCount,
				weakCount,
			),
		)

		for _, finding := range r.Headers.SecurityFindings {
			statusIcon := "✓"
			if finding.Status == "missing" {
				statusIcon = "✗"
			} else if finding.Status == "weak" {
				statusIcon = "⚠"
			}

			output.WriteString(
				fmt.Sprintf("%s %s: %s\n", statusIcon, finding.Header, finding.Description),
			)
			if finding.Value != "" {
				output.WriteString(fmt.Sprintf("   Value: %s\n", finding.Value))
			}
			output.WriteString("\n")
		}
	}

	// Vulnerabilities
	if r.Vulnerabilities != nil && len(r.Vulnerabilities.Vulnerabilities) > 0 {
		output.WriteString("VULNERABILITIES\n")
		output.WriteString(strings.Repeat("-", 50) + "\n")

		severityCount := map[string]int{}
		for _, vuln := range r.Vulnerabilities.Vulnerabilities {
			severityCount[vuln.Severity]++
		}

		output.WriteString(
			fmt.Sprintf("Critical: %d | High: %d | Medium: %d | Low: %d | Info: %d\n\n",
				severityCount["critical"], severityCount["high"], severityCount["medium"],
				severityCount["low"], severityCount["info"]),
		)

		for _, vuln := range r.Vulnerabilities.Vulnerabilities {
			output.WriteString(fmt.Sprintf("● %s: %s\n", vuln.Type, vuln.Description))
			output.WriteString(fmt.Sprintf("  Evidence: %s\n", vuln.Evidence))
			output.WriteString(fmt.Sprintf("  Remediation: %s\n\n", vuln.Remediation))
		}
	} else if r.Vulnerabilities != nil {
		output.WriteString("VULNERABILITIES\n")
		output.WriteString(strings.Repeat("-", 50) + "\n")
		output.WriteString("No vulnerabilities detected\n\n")
	}

	// Robots.txt
	if r.Robots != nil {
		output.WriteString("ROBOTS.TXT ANALYSIS\n")
		output.WriteString(strings.Repeat("-", 50) + "\n")

		if len(r.Robots.DisallowedPaths) > 0 {
			output.WriteString("Disallowed Paths:\n")
			for _, path := range r.Robots.DisallowedPaths {
				output.WriteString(fmt.Sprintf("  • %s\n", path))
			}
			output.WriteString("\n")
		}

		if len(r.Robots.AllowedPaths) > 0 {
			output.WriteString("Allowed Paths:\n")
			for _, path := range r.Robots.AllowedPaths {
				output.WriteString(fmt.Sprintf("  • %s\n", path))
			}
			output.WriteString("\n")
		}

		if len(r.Robots.Sitemaps) > 0 {
			output.WriteString("Sitemaps:\n")
			for _, sitemap := range r.Robots.Sitemaps {
				output.WriteString(fmt.Sprintf("  • %s\n", sitemap))
			}
			output.WriteString("\n")
		}

		if len(r.Robots.DisallowedPaths) == 0 && len(r.Robots.AllowedPaths) == 0 {
			output.WriteString("No robots.txt rules found or file is empty\n\n")
		}
	}

	_, err = file.WriteString(output.String())
	return err
}
