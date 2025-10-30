// HTML content extraction and text cleaning.
// Converts HTML to readable terminal text with link preservation.
// Applies site-specific cleaning rules and removes unwanted elements.
package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Add at package level
var whitespaceRegex = regexp.MustCompile(`\s+`)

func NewExtractor() *Extractor {
	return NewExtractorWithConfig(DefaultConfigPtr())
}

func NewExtractorWithConfig(cfg *Config) *Extractor {
	return &Extractor{
		config: cfg,
		links:  &LinkProcessor{},
	}
}

func (e *Extractor) getSiteConfig(rawURL string) *SiteConfig {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	host := parsed.Host

	// Try exact match first
	if siteConfig, exists := e.config.SiteOverrides[host]; exists {
		return &siteConfig
	}

	// Skip wildcard logic for now - just do parent domain fallback
	domainParts := strings.Split(host, ".")
	if len(domainParts) >= 2 {
		parentDomain := strings.Join(domainParts[len(domainParts)-2:], ".")
		if siteConfig, exists := e.config.SiteOverrides[parentDomain]; exists {
			return &siteConfig
		}
	}

	return nil
}

func (l *LinkProcessor) ExtractLinksWithGoQuery(doc *goquery.Document) []string {
	links := []string{}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		linkText := strings.TrimSpace(s.Text())
		if linkText == "" {
			return
		}

		links = append(links, href)
		s.ReplaceWithHtml(fmt.Sprintf("[lightblue]→ %s[-]", linkText))
	})

	return links
}

func (l *LinkProcessor) AddLinksSection(text string, links []string) string {
	if len(links) > 0 {
		text += "\n\n--- Links ---\n"
		for i, link := range links {
			text += fmt.Sprintf("[lightblue][%d] %s[-]\n", i+1, link)
		}
	}
	return text
}

func (e *Extractor) Extract(html string, pageURL string) (string, []string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "Error parsing HTML", []string{}
	}

	// Remove unwanted elements
	doc.Find("script, noscript, style, meta, svg").Remove()

	if siteConfig := e.getSiteConfig(pageURL); siteConfig != nil {
		for _, selector := range siteConfig.RemoveElements {
			doc.Find(selector).Remove()
		}
	}

	var result strings.Builder
	var links []string
	linkCounter := 0

	// Process content
	doc.Find("h1, h2, h3, h4, h5, h6, p, a, td, li, code, pre").
		Each(func(i int, s *goquery.Selection) {
			// Skip <a> elements inside <li> (let the li handle them)
			if goquery.NodeName(s) == "a" && s.Closest("li").Length() > 0 {
				return
			}

			text := strings.TrimSpace(s.Text())
			if text == "" {
				return
			}

			tagName := goquery.NodeName(s)

			switch tagName {
			case "h1", "h2", "h3", "h4", "h5", "h6":
				// Only process headings that are NOT inside lists
				if s.Closest("li, ul, ol").Length() == 0 {
					result.WriteString(fmt.Sprintf("\n\n[lightblue]%s[-]\n\n", text))
				}
			case "a":
				href, exists := s.Attr("href")
				if exists && href != "" {
					if strings.HasPrefix(href, "magnet:") {
						linkCounter++
						links = append(links, href)
						// Show shortened version
						shortMagnet := href
						if len(shortMagnet) > 60 {
							shortMagnet = href[:60] + "..."
						}
						result.WriteString(
							fmt.Sprintf("[purple][%d][-] %s\n", linkCounter, shortMagnet),
						)
					} else {
						// Regular HTTP link
						linkCounter++
						links = append(links, href)
						result.WriteString(fmt.Sprintf("[blue][%d][-] %s\n", linkCounter, text))
					}
				}
			case "p":
				result.WriteString(text + "\n\n")
			case "pre":
			case "code":
				result.WriteString(fmt.Sprintf("[yellow]%s[-]\n\n", text))
			case "li":
				// Your existing li handling code here
				text = strings.TrimSpace(s.Text())
				text = whitespaceRegex.ReplaceAllString(text, " ")
				text = ensureWordSpacing(text)

				linksInLi := s.Find("a")
				if linksInLi.Length() > 0 {
					firstLink := linksInLi.First()
					href, exists := firstLink.Attr("href")
					if exists && href != "" {
						linkCounter++
						links = append(links, href)
						result.WriteString(fmt.Sprintf("\n[blue][%d][-] %s \n", linkCounter, text))
					} else {
						result.WriteString(fmt.Sprintf("\n%s \n", text))
					}
				} else {
					result.WriteString(fmt.Sprintf("\n%s \n", text))
				}

			}
		})

	content := result.String()

	// Later in the Extract method (around line 140):
	if strings.TrimSpace(content) == "" {
		content = doc.Find("body").Text()
		content = whitespaceRegex.ReplaceAllString(content, " ") // Use pre-compiled
		content = strings.TrimSpace(content)
	}

	if content == "" {
		content = "No content found"
	}

	return content, links
}

func ensureWordSpacing(text string) string {
	// Add space between letter+letter when no space exists
	return regexp.MustCompile(`([a-zA-ZåäöÅÄÖ])([A-ZÅÄÖ])`).ReplaceAllString(text, "${1} ${2}")
}

func isMagnetLink(href string) bool {
	return strings.HasPrefix(href, "magnet:?")
}
