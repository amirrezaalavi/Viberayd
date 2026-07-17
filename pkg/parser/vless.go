package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amiralavi/viberay/internal/errors"
	"github.com/amiralavi/viberay/internal/models"
)

// parseVLess handles vless:// URIs.
func parseVLess(raw string) (*models.VLessConfig, error) {
	raw = strings.TrimPrefix(raw, "vless://")
	uriStr, name := ExtractFragment(raw)

	u, err := url.Parse("vless://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid vless URL", errors.ErrInvalidFormat)
	}

	uuid := u.User.Username()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()
	path := q.Get("path")
	if path == "" && q.Get("type") == "grpc" {
		path = q.Get("serviceName")
	}

	cfg := &models.VLessConfig{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolVLess,
			Name:     name,
			Network:  q.Get("type"),
		},
		TLSConfig: models.TLSConfig{
			Enabled:     q.Get("security") == "tls" || q.Get("security") == "xtls" || q.Get("security") == "reality",
			SNI:         q.Get("sni"),
			Host:        q.Get("host"),
			Path:        path,
			ALPN:        q.Get("alpn"),
			Fingerprint: q.Get("fp"),
			SkipVerify:  q.Get("allowInsecure") == "1" || strings.ToLower(q.Get("allowInsecure")) == "true",
		},
		UUID:       uuid,
		Flow:       q.Get("flow"),
		Encryption: q.Get("encryption"),
	}
	if cfg.Encryption == "" {
		cfg.Encryption = "none"
	}
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}

	if err := ValidateUUID(cfg.UUID); err != nil {
		return nil, fmt.Errorf("%w: bad vless uuid", err)
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad vless port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing vless server", errors.ErrMissingField)
	}

	return cfg, nil
}
