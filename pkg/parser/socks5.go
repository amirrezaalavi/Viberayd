package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberayd/internal/errors"
	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// parseSocks5 handles socks5://, socks4://, and socks:// URIs.
// Format: socks5://<username>:<password>@<host>:<port>#Name
// Format: socks5://<host>:<port>#Name
func parseSocks5(raw string) (*models.Socks5Config, error) {
	raw = strings.TrimPrefix(raw, "socks5://")
	raw = strings.TrimPrefix(raw, "socks4://")
	raw = strings.TrimPrefix(raw, "socks://")
	uriStr, name := ExtractFragment(raw)

	u, err := url.Parse("socks5://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid socks URL", errors.ErrInvalidFormat)
	}

	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 1080
	}

	cfg := &models.Socks5Config{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolSocks5,
			Name:     name,
		},
	}

	if u.User != nil {
		cfg.Username = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}

	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad socks port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing socks server", errors.ErrMissingField)
	}

	return cfg, nil
}
