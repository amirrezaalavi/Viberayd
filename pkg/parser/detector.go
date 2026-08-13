package parser

import (
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/amirrezaalavi/Viberayd/internal/errors"
	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// DetectProtocol identifies the proxy protocol from a URI prefix.
func DetectProtocol(raw string) (models.Protocol, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "vmess://"):
		return models.ProtocolVMess, nil
	case strings.HasPrefix(raw, "vless://"):
		return models.ProtocolVLess, nil
	case strings.HasPrefix(raw, "trojan://"):
		return models.ProtocolTrojan, nil
	case strings.HasPrefix(raw, "ss://"):
		return models.ProtocolSS, nil
	case strings.HasPrefix(raw, "hysteria2://"), strings.HasPrefix(raw, "hy2://"):
		return models.ProtocolHysteria2, nil
	case strings.HasPrefix(raw, "tuic://"):
		return models.ProtocolTUIC, nil
	case strings.HasPrefix(raw, "wireguard://"):
		return models.ProtocolWireGuard, nil
	case strings.HasPrefix(raw, "socks5://"), strings.HasPrefix(raw, "socks4://"), strings.HasPrefix(raw, "socks://"):
		return models.ProtocolSocks5, nil
	default:
		return "", errors.ErrInvalidProtocol
	}
}

// ExtractFragment splits the `#name` fragment from the URI and returns the clean URI + name.
func ExtractFragment(raw string) (uri, name string) {
	if i := strings.LastIndex(raw, "#"); i >= 0 {
		name = raw[i+1:]
		uri = raw[:i]
		// URL-decode the name if needed (commonly left as-is)
		if dec, err := url.QueryUnescape(name); err == nil {
			name = dec
		}
		return uri, name
	}
	return raw, ""
}

// IsBase64Encoded reports whether s looks like a base64-encoded payload rather
// than raw proxy URIs. The heuristic: proxy URIs always contain "://" (e.g.
// vmess://, ss://), while base64 does not include the colon character at all.
// We also allow whitespace (newlines, spaces, tabs) since wrapped subscription
// files commonly insert line breaks.
func IsBase64Encoded(s string) bool {
	if s == "" {
		return false
	}
	// Fast-fail: if it contains "://" it's definitely a proxy URI or URL — not base64.
	// Base64 alphabet is A-Z, a-z, 0-9, +, /, and = for padding — it never has colon.
	if strings.Contains(s, "://") {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '+' || c == '/':
		case c == '=':
		case c == '\n' || c == '\r' || c == ' ' || c == '	':
			// Whitespace is common in line-wrapped base64 subscription payloads
		default:
			return false
		}
	}
	return true
}

// LooksLikeBase64 reports whether s appears to be base64-encoded.
// Deprecated: Use IsBase64Encoded instead, which handles whitespace and uses
// the colon-absence heuristic for better accuracy.
func LooksLikeBase64(s string) bool {
	if len(s) == 0 || len(s)%4 != 0 {
		return false
	}
	// Allow standard base64 alphabet + padding
	for i, c := range s {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '+' || c == '/':
		case c == '=' && i >= len(s)-2: // padding only at end
		default:
			return false
		}
	}
	return true
}

// DecodeBase64 decodes standard base64 (with padding). It returns the input unchanged if decoding fails.
func DecodeBase64(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.ErrInvalidEncoding
	}
	return b, nil
}

// DecodeBase64URL decodes base64url (no padding, URL-safe alphabet).
func DecodeBase64URL(s string) ([]byte, error) {
	// Add padding if necessary
	pad := 4 - len(s)%4
	if pad != 4 {
		s += strings.Repeat("=", pad)
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.ErrInvalidEncoding
	}
	return b, nil
}
