package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amiralavi/viberay/internal/errors"
	"github.com/amiralavi/viberay/internal/models"
)

// parseReality handles reality:// URIs or vless:// URIs with security=reality.
func parseReality(raw string) (*models.RealityConfig, error) {
	// If it's explicitly reality://, treat identically to vless:// with reality params
	raw = strings.TrimPrefix(raw, "reality://")
	if !strings.HasPrefix(raw, "vless://") {
		raw = "vless://" + raw
	}

	uriStr, name := ExtractFragment(strings.TrimPrefix(raw, "vless://"))
	u, err := url.Parse("vless://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid reality URL", errors.ErrInvalidFormat)
	}

	uuid := u.User.Username()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()
	cfg := &models.RealityConfig{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolReality,
			Name:     name,
			Network:  q.Get("type"),
		},
		TLSConfig: models.TLSConfig{
			Enabled:     true, // Reality is always TLS-like
			SNI:         q.Get("sni"),
			Host:        q.Get("host"),
			Path:        q.Get("path"),
			Fingerprint: q.Get("fp"),
		},
		UUID:      uuid,
		Flow:      q.Get("flow"),
		PublicKey: q.Get("pbk"),
		ShortID:   q.Get("sid"),
		SpiderX:   q.Get("spx"),
	}
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Flow == "" {
		cfg.Flow = "xtls-rprx-vision"
	}

	if err := ValidateUUID(cfg.UUID); err != nil {
		return nil, fmt.Errorf("%w: bad reality uuid", err)
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad reality port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing reality server", errors.ErrMissingField)
	}
	if err := ValidatePublicKey(cfg.PublicKey); err != nil {
		return nil, fmt.Errorf("%w: bad reality public key", err)
	}

	return cfg, nil
}

// IsRealityURL reports whether a vless:// URL contains reality parameters.
func IsRealityURL(raw string) bool {
	if !strings.HasPrefix(raw, "vless://") {
		return false
	}
	uriStr, _ := ExtractFragment(strings.TrimPrefix(raw, "vless://"))
	u, err := url.Parse("vless://" + uriStr)
	if err != nil {
		return false
	}
	return strings.ToLower(u.Query().Get("security")) == "reality" || u.Query().Get("pbk") != ""
}
