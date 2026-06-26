// Package fetcher downloads subscription content from remote URLs.
package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetch downloads the body from url and returns it as a string.
func Fetch(url string, timeout time.Duration) (string, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return string(b), nil
}
