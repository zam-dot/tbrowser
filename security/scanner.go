// Security scanning HTTP client with header preservation.
// Specialized fetcher for security analysis that retains response headers.
// Used by vulnerability detection and technology fingerprinting components.
package security

import (
	"io"
	"net/http"
	"time"
)

// Add to SecurityScanner struct
type SecurityScanner struct {
	timeout int
	client  *http.Client
}

func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		timeout: 10,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// New method to fetch HTML content with headers
func (s *SecurityScanner) FetchWithHeaders(targetURL string) (string, http.Header, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", nil, err
	}

	// Set realistic headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	return string(body), resp.Header, nil
}
