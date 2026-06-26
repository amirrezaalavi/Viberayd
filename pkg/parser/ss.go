package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amiralavi/viberay/internal/errors"
	"github.com/amiralavi/viberay/internal/models"
)

// parseSS handles ss:// URIs.
// Formats:
//   ss://base64(method:password)@host:port#name
//   ss://base64(method:password@host:port)#name   (legacy)
//   ss://method:password@host:port#name            (plain SIP002)
func parseSS(raw string) (*models.SSConfig, error) {
	raw = strings.TrimPrefix(raw, "ss://")
	uriStr, name := ExtractFragment(raw)

	// SS URIs have the form ss://<credentials>@<server>:<port>
	// <credentials> may be base64(method:password) or plain method:password.
	at := strings.LastIndex(uriStr, "@")
	if at < 0 {
		return nil, fmt.Errorf("%w: missing @ in ss URI", errors.ErrInvalidFormat)
	}
	credPart := uriStr[:at]
	serverPort := uriStr[at+1:]

	// Try base64-decode of the credentials first.
	var cred string
	if LooksLikeBase64(credPart) {
		if b, err := DecodeBase64(credPart); err == nil {
			cred = string(b)
		}
	}
	if cred == "" {
		cred = credPart // plain text
	}

	// Now we have method:password. Parse server:port separately.
	host, portStr, err := netSplitHostPort(serverPort)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid server:port in ss URI", errors.ErrInvalidFormat)
	}
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 8388
	}

	colon := strings.Index(cred, ":")
	if colon < 0 {
		return nil, fmt.Errorf("%w: missing method:password separator", errors.ErrInvalidFormat)
	}
	method := cred[:colon]
	password := cred[colon+1:]

	cfg := &models.SSConfig{
		BaseConfig: models.BaseConfig{
			Server:   host,
			Port:     port,
			Protocol: models.ProtocolSS,
			Name:     name,
		},
		Method:   method,
		Password: password,
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: invalid port in ss URI", errors.ErrInvalidFormat)
	}
	return cfg, nil
}



func parsePluginOpts(s string) map[string]string {
	m := make(map[string]string)
	for _, part := range strings.Split(s, ";") {
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

// netSplitHostPort wraps net.SplitHostPort but tolerates missing port.
func netSplitHostPort(s string) (host, port string, err error) {
	if strings.HasPrefix(s, "[") {
		// IPv6 literal
		if i := strings.LastIndex(s, "]:"); i >= 0 {
			return s[1:i], s[i+2:], nil
		}
		if strings.HasSuffix(s, "]") {
			return s[1 : len(s)-1], "", fmt.Errorf("missing port")
		}
	}
	if i := strings.LastIndex(s, ":"); i >= 0 {
		return s[:i], s[i+1:], nil
	}
	return s, "", fmt.Errorf("missing port")
}
