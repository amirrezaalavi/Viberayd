package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberay/internal/errors"
	"github.com/amirrezaalavi/Viberay/internal/models"
)

// parseHysteria2 handles hysteria2:// and hy2:// URIs.
// Format: hysteria2://<auth>@<host>:<port>?obfs=salamander&obfs-password=<pass>&up=100&down=200&sni=<sni>&alpn=h3#Name
func parseHysteria2(raw string) (*models.Hysteria2Config, error) {
	raw = strings.TrimPrefix(raw, "hysteria2://")
	raw = strings.TrimPrefix(raw, "hy2://")
	uriStr, name := ExtractFragment(raw)

	u, err := url.Parse("hysteria2://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid hysteria2 URL", errors.ErrInvalidFormat)
	}

	auth := u.User.Username()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()

	up, _ := strconv.Atoi(q.Get("up"))
	down, _ := strconv.Atoi(q.Get("down"))
	hopInt, _ := strconv.Atoi(q.Get("hopInterval"))
	if hopInt == 0 {
		hopInt = 30
	}

	cfg := &models.Hysteria2Config{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolHysteria2,
			Name:     name,
		},
		TLSConfig: models.TLSConfig{
			Enabled:     true,
			SNI:         q.Get("sni"),
			ALPN:        q.Get("alpn"),
			Fingerprint: q.Get("fp"),
			SkipVerify:  q.Get("insecure") == "1" || strings.ToLower(q.Get("insecure")) == "true" || q.Get("allowInsecure") == "1",
		},
		Auth:         auth,
		ObfsType:     q.Get("obfs"),
		ObfsPassword: q.Get("obfs-password"),
		UpMbps:       up,
		DownMbps:     down,
		Ports:        q.Get("ports"),
		HopInterval:  hopInt,
	}

	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad hysteria2 port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing hysteria2 server", errors.ErrMissingField)
	}
	if cfg.Auth == "" {
		return nil, fmt.Errorf("%w: missing hysteria2 auth password", errors.ErrMissingField)
	}

	return cfg, nil
}
