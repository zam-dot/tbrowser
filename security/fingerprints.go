// security/fingerprint.go
package security

import (
	"fmt"
	"net/http"
	"strings"
)

type TechFingerprint struct {
	URL        string
	Server     string
	Framework  string
	JavaScript []string
	CSS        []string
	Analytics  []string
	CDN        []string
}

func FingerprintTechnology(
	targetURL string,
	htmlContent string,
	headers http.Header,
) *TechFingerprint {
	fp := &TechFingerprint{
		URL: targetURL,
	}

	// Detect server from headers
	fp.Server = detectServer(headers)

	// Detect technologies from HTML
	fp.Framework = detectFramework(htmlContent)
	fp.JavaScript = detectJavaScript(htmlContent)
	fp.CSS = detectCSS(htmlContent)
	fp.Analytics = detectAnalytics(htmlContent)
	fp.CDN = detectCDN(htmlContent)

	return fp
}

func detectServer(headers http.Header) string {
	server := headers.Get("Server")
	if server != "" {
		return server
	}

	// Fallback to X-Powered-By
	poweredBy := headers.Get("X-Powered-By")
	if poweredBy != "" {
		return poweredBy
	}

	return "Unknown"
}

func detectFramework(html string) string {
	// React
	if strings.Contains(html, "react") || strings.Contains(html, "React") {
		return "React"
	}

	// Vue.js
	if strings.Contains(html, "vue") || strings.Contains(html, "Vue") {
		return "Vue.js"
	}

	// Angular
	if strings.Contains(html, "angular") || strings.Contains(html, "ng-") {
		return "Angular"
	}

	// jQuery
	if strings.Contains(html, "jquery") {
		return "jQuery"
	}

	// WordPress
	if strings.Contains(html, "wp-") || strings.Contains(html, "wordpress") {
		return "WordPress"
	}

	return "Unknown"
}

func detectJavaScript(html string) []string {
	var libs []string

	jsPatterns := map[string]string{
		"React":     "react",
		"Vue.js":    "vue",
		"Angular":   "angular",
		"jQuery":    "jquery",
		"Bootstrap": "bootstrap",
		"Alpine.js": "alpine",
		"Three.js":  "three",
		"D3.js":     "d3",
		"Next.js":   "next",
		"Nuxt.js":   "nuxt",
		"Svelte":    "svelte",
		"Laravel":   "laravel",
		"Django":    "django",
	}

	for lib, pattern := range jsPatterns {
		if strings.Contains(strings.ToLower(html), pattern) {
			libs = append(libs, lib)
		}
	}

	return libs
}

func detectCSS(html string) []string {
	var frameworks []string

	cssPatterns := map[string]string{
		"Bootstrap":   "bootstrap",
		"Tailwind":    "tailwind",
		"Bulma":       "bulma",
		"Foundation":  "foundation",
		"Materialize": "materialize",
	}

	for framework, pattern := range cssPatterns {
		if strings.Contains(strings.ToLower(html), pattern) {
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

func detectAnalytics(html string) []string {
	var analytics []string

	analyticsPatterns := map[string]string{
		"Google Analytics":   "ga.js",
		"Google Tag Manager": "gtm.js",
		"Facebook Pixel":     "facebook.net",
		"Hotjar":             "hotjar",
	}

	for service, pattern := range analyticsPatterns {
		if strings.Contains(html, pattern) {
			analytics = append(analytics, service)
		}
	}

	return analytics
}

func detectCDN(html string) []string {
	var cdns []string

	cdnPatterns := map[string]string{
		"Cloudflare": "cloudflare",
		"Akamai":     "akamai",
		"Fastly":     "fastly",
		"CloudFront": "cloudfront",
		"BunnyCDN":   "bunny",
	}

	for cdn, pattern := range cdnPatterns {
		if strings.Contains(strings.ToLower(html), pattern) {
			cdns = append(cdns, cdn)
		}
	}

	return cdns
}

func (f *TechFingerprint) FormatForPanel() string {
	var output strings.Builder

	output.WriteString("[yellow]Technology Stack:[-]\n\n")

	// Server
	output.WriteString("[white]Web Server:[-]\n")
	output.WriteString(fmt.Sprintf("  • %s\n\n", f.Server))

	// Framework
	output.WriteString("[white]Framework:[-]\n")
	output.WriteString(fmt.Sprintf("  • %s\n\n", f.Framework))

	// JavaScript Libraries
	if len(f.JavaScript) > 0 {
		output.WriteString("[white]JavaScript:[-]\n")
		for _, js := range f.JavaScript {
			output.WriteString(fmt.Sprintf("  • %s\n", js))
		}
		output.WriteString("\n")
	}

	// CSS Frameworks
	if len(f.CSS) > 0 {
		output.WriteString("[white]CSS Frameworks:[-]\n")
		for _, css := range f.CSS {
			output.WriteString(fmt.Sprintf("  • %s\n", css))
		}
		output.WriteString("\n")
	}

	// Analytics
	if len(f.Analytics) > 0 {
		output.WriteString("[white]Analytics:[-]\n")
		for _, analytic := range f.Analytics {
			output.WriteString(fmt.Sprintf("  • %s\n", analytic))
		}
		output.WriteString("\n")
	}

	// CDN
	if len(f.CDN) > 0 {
		output.WriteString("[white]CDN:[-]\n")
		for _, cdn := range f.CDN {
			output.WriteString(fmt.Sprintf("  • %s\n", cdn))
		}
	}

	return output.String()
}
