package parser

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberay/internal/errors"
	"github.com/amirrezaalavi/Viberay/internal/models"
)

// vmessJSON is the on-wire format used by vmess:// URIs.
type vmessJSON struct {
	V    string `json:"v"`
	Ps   string `json:"ps"`
	Add  string `json:"add"`
	Port string `json:"port"`
	ID   string `json:"id"`
	Aid  string `json:"aid"`
	Scy  string `json:"scy"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	Path string `json:"path"`
	TLS  string `json:"tls"`
	Sni  string `json:"sni"`
	Alpn string `json:"alpn"`
	Fp   string `json:"fp"`
}

func parseVMess(raw string) (*models.VMessConfig, error) {
	raw = strings.TrimPrefix(raw, "vmess://")
	uriStr, name := ExtractFragment(raw)

	b, err := DecodeBase64(uriStr)
	if err != nil {
		return nil, err
	}

	var j vmessJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, fmt.Errorf("%w: invalid vmess JSON: %v", errors.ErrInvalidFormat, err)
	}

	port, _ := strconv.Atoi(j.Port)
	if port == 0 {
		port = 443
	}
	aid, _ := strconv.Atoi(j.Aid)

	cfg := &models.VMessConfig{
		BaseConfig: models.BaseConfig{
			Server:   j.Add,
			Port:     port,
			Protocol: models.ProtocolVMess,
			Name:     name,
			Network:  j.Net,
		},
		TLSConfig: models.TLSConfig{
			Enabled:     strings.ToLower(j.TLS) == "tls" || strings.ToLower(j.TLS) == "xtls",
			SNI:         j.Sni,
			Host:        j.Host,
			Path:        j.Path,
			ALPN:        j.Alpn,
			Fingerprint: j.Fp,
		},
		UUID:     j.ID,
		AlterID:  aid,
		Security: j.Scy,
	}

	if cfg.Security == "" {
		cfg.Security = "auto"
	}

	// Validate core fields
	if err := ValidateUUID(cfg.UUID); err != nil {
		return nil, fmt.Errorf("%w: bad vmess uuid", err)
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad vmess port", err)
	}
	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing vmess server address", errors.ErrMissingField)
	}

	return cfg, nil
}
