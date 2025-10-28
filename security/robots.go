// Robots.txt analysis and crawler directive parsing.
// Fetches and analyzes robots.txt files for security reconnaissance.
// Identifies disallowed paths, sitemaps, and crawler access rules.
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

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Use CutPrefix with if-else chain (cleaner)
		if path, found := strings.CutPrefix(line, "Disallow:"); found {
			path = strings.TrimSpace(path)
			if path != "" {
				robots.DisallowedPaths = append(robots.DisallowedPaths, path)
			}
		} else if path, found := strings.CutPrefix(line, "Allow:"); found {
			path = strings.TrimSpace(path)
			if path != "" {
				robots.AllowedPaths = append(robots.AllowedPaths, path)
			}
		} else if sitemap, found := strings.CutPrefix(line, "Sitemap:"); found {
			sitemap = strings.TrimSpace(sitemap)
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
