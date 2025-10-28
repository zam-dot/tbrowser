// security/robots.go
package security

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RobotsTxt struct {
	AllowedPaths    []string
	DisallowedPaths []string
	Sitemaps        []string
	RawContent      string
}

func FetchRobotsTxt(targetURL string) (*RobotsTxt, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// Build robots.txt URL
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", parsed.Scheme, parsed.Host)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(robotsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return ParseRobotsTxt(string(body)), nil
}

func ParseRobotsTxt(content string) *RobotsTxt {
	robots := &RobotsTxt{
		RawContent: content,
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Disallow:") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "Disallow:"))
			if path != "" {
				robots.DisallowedPaths = append(robots.DisallowedPaths, path)
			}
		} else if strings.HasPrefix(line, "Allow:") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "Allow:"))
			if path != "" {
				robots.AllowedPaths = append(robots.AllowedPaths, path)
			}
		} else if strings.HasPrefix(line, "Sitemap:") {
			sitemap := strings.TrimSpace(strings.TrimPrefix(line, "Sitemap:"))
			if sitemap != "" {
				robots.Sitemaps = append(robots.Sitemaps, sitemap)
			}
		}
	}

	return robots
}

func (r *RobotsTxt) FormatForPanel() string {
	var output strings.Builder

	output.WriteString("[yellow]Robots.txt Analysis:[-]\n\n")

	if len(r.DisallowedPaths) > 0 {
		output.WriteString("[red]Disallowed Paths:[-]\n")
		for _, path := range r.DisallowedPaths {
			output.WriteString(fmt.Sprintf("  • %s\n", path))
		}
		output.WriteString("\n")
	}

	if len(r.AllowedPaths) > 0 {
		output.WriteString("[green]Allowed Paths:[-]\n")
		for _, path := range r.AllowedPaths {
			output.WriteString(fmt.Sprintf("  • %s\n", path))
		}
		output.WriteString("\n")
	}

	if len(r.Sitemaps) > 0 {
		output.WriteString("[blue]Sitemaps:[-]\n")
		for _, sitemap := range r.Sitemaps {
			output.WriteString(fmt.Sprintf("  • %s\n", sitemap))
		}
	}

	if output.Len() == 0 {
		output.WriteString("No robots.txt rules found or file is empty")
	}

	return output.String()
}
