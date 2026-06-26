package parser

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/amiralavi/viberay/internal/errors"
	"github.com/amiralavi/viberay/internal/models"
)

// Parse accepts a raw string that may contain:
//   - a single proxy URI (ss://, vmess://, vless://, trojan://, reality://)
//   - a base64-encoded subscription list (one or more URIs)
//   - plain text with one URI per line
//
// It returns a slice of parsed ProxyConfigs and an aggregated error if any configs failed.
func Parse(input string) ([]models.ProxyConfig, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.ErrInvalidFormat
	}

	// If the entire payload is base64, decode it first.
	if LooksLikeBase64(input) {
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err == nil {
			input = string(decoded)
		}
		// If decoding fails, continue with raw input (it might just look like base64)
	}

	// Split into lines and parse each.
	var configs []models.ProxyConfig
	var errs []error

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		cfg, err := ParseSingle(line)
		if err != nil {
			errs = append(errs, fmt.Errorf("line %d: %w", len(configs)+len(errs)+1, err))
			continue
		}
		configs = append(configs, cfg)
	}

	if err := scanner.Err(); err != nil {
		return configs, err
	}

	if len(configs) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all %d lines failed to parse", len(errs))
	}
	return configs, nil
}

// ParseSingle parses exactly one proxy URI into a ProxyConfig.
func ParseSingle(raw string) (models.ProxyConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return models.ProxyConfig{}, errors.ErrInvalidFormat
	}

	// Detect protocol. Reality URLs start as vless:// but have reality params.
	proto, err := DetectProtocol(raw)
	if err != nil {
		return models.ProxyConfig{}, err
	}

	// Override protocol for Reality
	if proto == models.ProtocolVLess && IsRealityURL(raw) {
		proto = models.ProtocolReality
	}

	switch proto {
	case models.ProtocolSS:
		cfg, err := parseSS(raw)
		if err != nil {
			return models.ProxyConfig{}, err
		}
		return models.ProxyConfig{SS: cfg}, nil

	case models.ProtocolVMess:
		cfg, err := parseVMess(raw)
		if err != nil {
			return models.ProxyConfig{}, err
		}
		return models.ProxyConfig{VMess: cfg}, nil

	case models.ProtocolVLess:
		cfg, err := parseVLess(raw)
		if err != nil {
			return models.ProxyConfig{}, err
		}
		return models.ProxyConfig{VLess: cfg}, nil

	case models.ProtocolTrojan:
		cfg, err := parseTrojan(raw)
		if err != nil {
			return models.ProxyConfig{}, err
		}
		return models.ProxyConfig{Trojan: cfg}, nil

	case models.ProtocolReality:
		cfg, err := parseReality(raw)
		if err != nil {
			return models.ProxyConfig{}, err
		}
		return models.ProxyConfig{Reality: cfg}, nil

	default:
		return models.ProxyConfig{}, errors.ErrInvalidProtocol
	}
}
