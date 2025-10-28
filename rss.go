// rss.go
package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Compile ALL regex patterns once at package level
var (
	rssItemRegex  = regexp.MustCompile(`(?s)<item>(.*?)</item>`)
	rssTitleRegex = regexp.MustCompile(`(?s)<title>(.*?)</title>`)
	rssLinkRegex  = regexp.MustCompile(`(?s)<link>(.*?)</link>`)
	rssDescRegex  = regexp.MustCompile(`(?s)<description>(.*?)</description>`)
	htmlTagRegex  = regexp.MustCompile(`<[^>]*>`) // Reused for HTML stripping
)

func (r *RSSParser) ParseRSS(content string) (string, []string) {
	var result strings.Builder
	var links []string

	// Use pre-compiled regex
	items := rssItemRegex.FindAllStringSubmatch(content, -1)

	for i, item := range items {
		if i >= 10 {
			break
		}

		titleMatch := rssTitleRegex.FindStringSubmatch(item[1])
		title := "No title"
		if len(titleMatch) > 1 {
			title = strings.TrimSpace(titleMatch[1])
		}

		linkMatch := rssLinkRegex.FindStringSubmatch(item[1])
		link := ""
		if len(linkMatch) > 1 {
			link = strings.TrimSpace(linkMatch[1])
		}

		if link != "" {
			links = append(links, link)
			result.WriteString(fmt.Sprintf("[green][%d] %s[-]\n", len(links), title))
		} else {
			result.WriteString(fmt.Sprintf("[green]%s[-]\n", title))
		}

		descMatch := rssDescRegex.FindStringSubmatch(item[1])
		if len(descMatch) > 1 {
			// Use the pre-compiled htmlTagRegex
			cleanDesc := htmlTagRegex.ReplaceAllString(descMatch[1], "")
			description := strings.TrimSpace(cleanDesc)
			if description != "" && len(description) > 200 {
				description = description[:200] + "..."
			}
			result.WriteString(fmt.Sprintf("  %s\n", description))
		}
		result.WriteString("\n")
	}

	return result.String(), links
}
