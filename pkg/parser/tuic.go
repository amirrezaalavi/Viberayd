package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberay/internal/errors"
	"github.com/amirrezaalavi/Viberay/internal/models"
)

// parseTUIC handles tuic:// URIs.
// Format: tuic://<uuid>:<password>@<host>:<port>?congestion_control=bbr&sni=<sni>&alpn=h3#Name
func parseTUIC(raw string) (*models.TUICConfig, error) {
	raw = strings.TrimPrefix(raw, "tuic://")
	uriStr, name := ExtractFragment(raw)

	u, err := url.Parse("tuic://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tuic URL", errors.ErrInvalidFormat)
	}

	userInfo := u.User
	uuid := userInfo.Username()
	password, _ := userInfo.Password()

	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 443
	}

	q := u.Query()

	cfg := &models.TUICConfig{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolTUIC,
			Name:     name,
		},
		TLSConfig: models.TLSConfig{
			Enabled:     true,
			SNI:         q.Get("sni"),
			ALPN:        q.Get("alpn"),
			Fingerprint: q.Get("fp"),
			SkipVerify:  q.Get("allowInsecure") == "1" || strings.ToLower(q.Get("allowInsecure")) == "true" || q.Get("skip_verify") == "1",
		},
		UUID:              uuid,
		Password:          password,
		CongestionControl: q.Get("congestion_control"),
		UDPRelayMode:      q.Get("udp_relay_mode"),
		Heartbeat:         q.Get("heartbeat"),
		ReduceRTT:         q.Get("reduce_rtt") == "1" || strings.ToLower(q.Get("reduce_rtt")) == "true",
		RequestTimeout:    q.Get("request_timeout"),
	}

	if cfg.CongestionControl == "" {
		cfg.CongestionControl = "bbr"
	}
	if cfg.UDPRelayMode == "" {
		cfg.UDPRelayMode = "native"
	}

	if err := ValidateUUID(cfg.UUID); err != nil {
		return nil, fmt.Errorf("%w: bad tuic uuid", err)
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad tuic port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing tuic server", errors.ErrMissingField)
	}

	return cfg, nil
}
