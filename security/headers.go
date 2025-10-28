// security/headers.go
package security

import (
	"fmt"
	"net/http"
	"strings"
)

type SecurityHeaders struct {
	URL              string
	Headers          map[string]string
	SecurityFindings []SecurityFinding
}

type SecurityFinding struct {
	Header      string
	Status      string // "present", "missing", "weak"
	Value       string
	Description string
	Severity    string // "high", "medium", "low"
}

func AnalyzeSecurityHeaders(targetURL string, headers http.Header) *SecurityHeaders {
	security := &SecurityHeaders{
		URL:     targetURL,
		Headers: make(map[string]string),
	}

	// Convert headers to map for easier processing
	for key, values := range headers {
		if len(values) > 0 {
			security.Headers[key] = values[0]
		}
	}

	// Analyze each security header
	security.analyzeHSTS()
	security.analyzeCSP()
	security.analyzeXFrameOptions()
	security.analyzeXContentTypeOptions()
	security.analyzeXXSSProtection()
	security.analyzeReferrerPolicy()
	security.analyzePermissionsPolicy()

	return security
}

func (s *SecurityHeaders) analyzeHSTS() {
	value, exists := s.Headers["Strict-Transport-Security"]
	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "Strict-Transport-Security",
			Status:      "missing",
			Description: "Forces HTTPS connections, prevents SSL stripping attacks",
			Severity:    "high",
		})
		return
	}

	// Check if HSTS includes preload and max-age
	status := "present"
	desc := "Forces HTTPS connections"

	if strings.Contains(strings.ToLower(value), "max-age=0") {
		status = "weak"
		desc = "HSTS is disabled (max-age=0)"
	} else if strings.Contains(strings.ToLower(value), "preload") {
		desc = "HSTS with preload - excellent security"
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "Strict-Transport-Security",
		Status:      status,
		Value:       value,
		Description: desc,
		Severity:    "high",
	})
}

func (s *SecurityHeaders) analyzeCSP() {
	value, exists := s.Headers["Content-Security-Policy"]
	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "Content-Security-Policy",
			Status:      "missing",
			Description: "Prevents XSS attacks by controlling resource loading",
			Severity:    "high",
		})
		return
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "Content-Security-Policy",
		Status:      "present",
		Value:       value,
		Description: "Protects against XSS attacks",
		Severity:    "high",
	})
}

func (s *SecurityHeaders) analyzeXFrameOptions() {
	value, exists := s.Headers["X-Frame-Options"]
	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "X-Frame-Options",
			Status:      "missing",
			Description: "Prevents clickjacking attacks",
			Severity:    "medium",
		})
		return
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "X-Frame-Options",
		Status:      "present",
		Value:       value,
		Description: "Protects against clickjacking",
		Severity:    "medium",
	})
}

func (s *SecurityHeaders) analyzeXContentTypeOptions() {
	value, exists := s.Headers["X-Content-Type-Options"]
	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "X-Content-Type-Options",
			Status:      "missing",
			Description: "Prevents MIME type sniffing attacks",
			Severity:    "medium",
		})
		return
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "X-Content-Type-Options",
		Status:      "present",
		Value:       value,
		Description: "Prevents MIME sniffing",
		Severity:    "medium",
	})
}

func (s *SecurityHeaders) analyzeXXSSProtection() {
	value, exists := s.Headers["X-XSS-Protection"]
	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "X-XSS-Protection",
			Status:      "missing",
			Description: "Legacy XSS protection (deprecated but still useful)",
			Severity:    "low",
		})
		return
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "X-XSS-Protection",
		Status:      "present",
		Value:       value,
		Description: "Legacy XSS protection",
		Severity:    "low",
	})
}

func (s *SecurityHeaders) analyzeReferrerPolicy() {
	value, exists := s.Headers["Referrer-Policy"]
	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "Referrer-Policy",
			Status:      "missing",
			Description: "Controls referrer information in requests",
			Severity:    "low",
		})
		return
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "Referrer-Policy",
		Status:      "present",
		Value:       value,
		Description: "Controls referrer information",
		Severity:    "low",
	})
}

func (s *SecurityHeaders) analyzePermissionsPolicy() {
	value, exists := s.Headers["Permissions-Policy"]
	if !exists {
		// Also check for Feature-Policy (older name)
		value, exists = s.Headers["Feature-Policy"]
	}

	if !exists {
		s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
			Header:      "Permissions-Policy",
			Status:      "missing",
			Description: "Controls browser features and APIs",
			Severity:    "medium",
		})
		return
	}

	s.SecurityFindings = append(s.SecurityFindings, SecurityFinding{
		Header:      "Permissions-Policy",
		Status:      "present",
		Value:       value,
		Description: "Controls browser features",
		Severity:    "medium",
	})
}

func (s *SecurityHeaders) FormatForPanel() string {
	var output strings.Builder

	output.WriteString("[yellow]Security Headers Analysis:[-]\n\n")

	if len(s.SecurityFindings) == 0 {
		output.WriteString("No security headers analyzed yet")
		return output.String()
	}

	// Count findings by status
	presentCount := 0
	missingCount := 0
	weakCount := 0

	for _, finding := range s.SecurityFindings {
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
		fmt.Sprintf("[green]✓ Present: %d[-] | [red]✗ Missing: %d[-] | [yellow]⚠ Weak: %d[-]\n\n",
			presentCount, missingCount, weakCount),
	)

	// Show findings
	for _, finding := range s.SecurityFindings {
		// Color code based on status and severity
		var statusColor, severityIcon string

		// More sophisticated approach
		if finding.Status == "present" {
			// Good things that are present - use positive colors
			switch finding.Severity {
			case "high":
				severityIcon = "🟢" // Green for important security features that are present
			case "medium":
				severityIcon = "🔵" // Blue for medium importance features that are present
			case "low":
				severityIcon = "⚪" // White/gray for low importance features that are present
			}
		} else {
			// Missing or weak things - use warning colors
			switch finding.Severity {
			case "high":
				severityIcon = "🔴" // Red for critically missing security features
			case "medium":
				severityIcon = "🟡" // Yellow for recommended but missing features
			case "low":
				severityIcon = "⚪" // White/gray for nice-to-have missing features
			}
		}

		output.WriteString(fmt.Sprintf("%s [%s]%s[-] %s\n",
			severityIcon, statusColor, finding.Header, finding.Status))

		if finding.Value != "" {
			output.WriteString(fmt.Sprintf("   Value: [white]%s[-]\n", finding.Value))
		}

		output.WriteString(fmt.Sprintf("   %s\n\n", finding.Description))
	}

	return output.String()
}
