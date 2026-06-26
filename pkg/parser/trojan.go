package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amiralavi/viberay/internal/errors"
	"github.com/amiralavi/viberay/internal/models"
)

// parseTrojan handles trojan:// URIs.
func parseTrojan(raw string) (*models.TrojanConfig, error) {
	raw = strings.TrimPrefix(raw, "trojan://")
	uriStr, name := ExtractFragment(raw)

	u, err := url.Parse("trojan://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid trojan URL", errors.ErrInvalidFormat)
	}

	password := u.User.Username()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()
	cfg := &models.TrojanConfig{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolTrojan,
			Name:     name,
			Network:  q.Get("type"),
		},
		TLSConfig: models.TLSConfig{
			Enabled:     true, // Trojan is always TLS
			SNI:         q.Get("sni"),
			Host:        q.Get("host"),
			Path:        q.Get("path"),
			ALPN:        q.Get("alpn"),
			Fingerprint: q.Get("fp"),
			SkipVerify:  q.Get("allowInsecure") == "1" || strings.ToLower(q.Get("allowInsecure")) == "true",
		},
		Password: password,
		Flow:     q.Get("flow"),
	}
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}

	if cfg.Password == "" {
		return nil, fmt.Errorf("%w: missing trojan password", errors.ErrMissingField)
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad trojan port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing trojan server", errors.ErrMissingField)
	}

	return cfg, nil
}
