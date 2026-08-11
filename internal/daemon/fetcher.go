package daemon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/models"
	"github.com/amirrezaalavi/Viberayd/pkg/fetcher"
	"github.com/amirrezaalavi/Viberayd/pkg/parser"
)

func LoadURLs(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read urls: %w", err)
	}

	var urls []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	return urls, nil
}

func FetchAndParse(urls []string, timeout time.Duration) (map[string]models.ProxyConfig, error) {
	result := make(map[string]models.ProxyConfig)

	for _, url := range urls {
		raw, err := fetcher.Fetch(url, timeout)
		if err != nil {
			slog.Warn("fetch failed", "url", url, "error", err)
			continue
		}

		configs, err := parser.Parse(raw)
		if err != nil {
			slog.Warn("parse failed", "url", url, "error", err)
			continue
		}

		for _, cfg := range configs {
			h := sha256Of(cfg.Raw)
			if _, exists := result[h]; !exists {
				result[h] = cfg
			}
		}

		slog.Info("fetched subscription", "url", url, "configs", len(configs), "new", len(result))
	}

	return result, nil
}

func MergeIntoState(s *State, configs map[string]models.ProxyConfig, sourceURL string) {
	for hash, cfg := range configs {
		if _, exists := s.Configs[hash]; exists {
			continue
		}

		addr := cfg.Addr()
		host, portStr, err := net.SplitHostPort(addr)
		port := 0
		if err == nil {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		} else {
			host = addr
		}

		s.Configs[hash] = &ConfigEntry{
			Raw:       cfg.Raw,
			Host:      host,
			Port:      port,
			Protocol:  string(cfg.Protocol()),
			SourceURL: sourceURL,
			FirstSeen: time.Now(),
			State:     StateUnknown,
		}
	}
}

func sha256Of(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
