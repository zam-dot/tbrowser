// fetcher.go
package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func NewFetcherWithConfig(cfg *Config) *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: time.Duration(cfg.Network.TimeoutSeconds) * time.Second},
		config: cfg,
	}
}

func (f *Fetcher) setHeaders(req *http.Request) {
	userAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	if f.config != nil {
		userAgent = f.config.Network.UserAgent
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set(
		"Accept",
		"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	)
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func (f *Fetcher) FetchURL(url string) (string, []string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", nil, err
	}

	f.setHeaders(req)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, err
	}

	extractor := NewExtractorWithConfig(f.config)
	cleanText, links := extractor.Extract(string(body), url)

	return cleanText, links, nil
}

// fetcher.go - Add this method
func (f *Fetcher) FetchRSS(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	f.setHeaders(req)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	req.Header.Set("User-Agent", "TUI-RSS-Reader/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "xml") && !strings.Contains(contentType, "rss") {
		return "", fmt.Errorf("server returned HTML instead of RSS (Content-Type: %s)", contentType)
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
