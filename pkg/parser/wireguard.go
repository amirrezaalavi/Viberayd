package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberay/internal/errors"
	"github.com/amirrezaalavi/Viberay/internal/models"
)

// parseWireGuard handles wireguard:// URIs.
// Formats:
//
//	wireguard://base64(json)#Name
//	wireguard://<private_key>@<host>:<port>?publicKey=<pub>&address=<cidr>#Name
func parseWireGuard(raw string) (*models.WireGuardConfig, error) {
	raw = strings.TrimPrefix(raw, "wireguard://")
	uriStr, name := ExtractFragment(raw)

	// Check if it's base64-encoded JSON (no "@" in body)
	if idx := strings.Index(uriStr, "@"); idx < 0 {
		return parseWireGuardJSON(uriStr, name)
	}

	// Standard URI format: private_key@host:port?params
	u, err := url.Parse("wireguard://" + uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid wireguard URL", errors.ErrInvalidFormat)
	}

	privateKey := u.User.Username()
	port, _ := strconv.Atoi(u.Port())
	if port == 0 {
		port = 51820
	}

	q := u.Query()

	cfg := &models.WireGuardConfig{
		BaseConfig: models.BaseConfig{
			Server:   u.Hostname(),
			Port:     port,
			Protocol: models.ProtocolWireGuard,
			Name:     name,
		},
		PrivateKey:   privateKey,
		PublicKey:    q.Get("publicKey"),
		PresharedKey: q.Get("presharedKey"),
		LocalAddress: q.Get("address"),
		DNS:          q.Get("dns"),
		AllowedIPs:   q.Get("allowedIPs"),
		Reserved:     q.Get("reserved"),
	}

	mtu, _ := strconv.Atoi(q.Get("mtu"))
	if mtu > 0 {
		cfg.MTU = mtu
	} else {
		cfg.MTU = 1420
	}

	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing wireguard server", errors.ErrMissingField)
	}
	if err := ValidatePortInt(cfg.Port); err != nil {
		return nil, fmt.Errorf("%w: bad wireguard port", err)
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("%w: missing wireguard private key", errors.ErrMissingField)
	}
	if cfg.PublicKey == "" {
		return nil, fmt.Errorf("%w: missing wireguard public key", errors.ErrMissingField)
	}

	return cfg, nil
}

// wireguardJSON is the on-wire format for base64-encoded JSON configs.
type wireguardJSON struct {
	PrivateKey   string `json:"privateKey"`
	PublicKey    string `json:"publicKey"`
	PresharedKey string `json:"presharedKey,omitempty"`
	Address      string `json:"address"`
	DNS          string `json:"dns,omitempty"`
	MTU          int    `json:"mtu,omitempty"`
	Reserved     string `json:"reserved,omitempty"`
	AllowedIPs   string `json:"allowedIPs,omitempty"`
	Endpoint     string `json:"endpoint"`
}

func parseWireGuardJSON(uriStr, name string) (*models.WireGuardConfig, error) {
	b, err := DecodeBase64(uriStr)
	if err != nil {
		return nil, fmt.Errorf("%w: wireguard base64 decode", errors.ErrInvalidFormat)
	}

	var j wireguardJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, fmt.Errorf("%w: invalid wireguard JSON", errors.ErrInvalidFormat)
	}

	host, portStr, err := netSplitHostPort(j.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid wireguard endpoint", errors.ErrInvalidFormat)
	}
	port, _ := strconv.Atoi(portStr)
	if port == 0 {
		port = 51820
	}

	cfg := &models.WireGuardConfig{
		BaseConfig: models.BaseConfig{
			Server:   host,
			Port:     port,
			Protocol: models.ProtocolWireGuard,
			Name:     name,
		},
		PrivateKey:   j.PrivateKey,
		PublicKey:    j.PublicKey,
		PresharedKey: j.PresharedKey,
		LocalAddress: j.Address,
		DNS:          j.DNS,
		AllowedIPs:   j.AllowedIPs,
		Reserved:     j.Reserved,
	}

	if j.MTU > 0 {
		cfg.MTU = j.MTU
	} else {
		cfg.MTU = 1420
	}

	if cfg.Server == "" {
		return nil, fmt.Errorf("%w: missing wireguard server from endpoint", errors.ErrMissingField)
	}
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("%w: missing wireguard private key", errors.ErrMissingField)
	}
	if cfg.PublicKey == "" {
		return nil, fmt.Errorf("%w: missing wireguard public key", errors.ErrMissingField)
	}

	return cfg, nil
}
