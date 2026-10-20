package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberayd/internal/errors"
	"github.com/amirrezaalavi/Viberayd/internal/models"
)

// flexString accepts either a JSON string or a bare JSON number. Real feeds
// emit "port":443 and "aid":0 as numbers as often as strings, and a strict
// string field rejected the whole config for that alone.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(strings.Trim(string(b), `"`))
	return nil
}

// vmessJSON is the on-wire format used by vmess:// URIs.
type vmessJSON struct {
	V    string     `json:"v"`
	Ps   string     `json:"ps"`
	Add  string     `json:"add"`
	Port flexString `json:"port"`
	ID   string     `json:"id"`
	Aid  flexString `json:"aid"`
	Scy  string     `json:"scy"`
	Net  string     `json:"net"`
	Type string     `json:"type"`
	Host string     `json:"host"`
	Path string     `json:"path"`
	TLS  string     `json:"tls"`
	Sni  string     `json:"sni"`
	Alpn string     `json:"alpn"`
	Fp   string     `json:"fp"`
}

// maxVMessRepair bounds how many trailing bytes may be trimmed when repairing
// a payload glued to the next entry's scheme word.
const maxVMessRepair = 16

func parseVMess(raw string) (*models.VMessConfig, error) {
	raw = strings.TrimPrefix(raw, "vmess://")
	uriStr, name := ExtractFragment(raw)

	j, err := decodeVMessPayload(uriStr)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(string(j.Port))
	if port == 0 {
		port = 443
	}
	aid, _ := strconv.Atoi(string(j.Aid))

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

// decodeVMessPayload decodes one vmess payload, repairing a defect common in
// generated feeds: the base64 blob is concatenated with the scheme word of the
// next entry ("<base64>vless") instead of being newline-separated, so a strict
// decode fails. The repair is bounded and only accepted when the trimmed
// payload yields a JSON object that carries the fields a vmess link needs.
// Genuinely broken payloads still fail with the original error.
func decodeVMessPayload(payload string) (*vmessJSON, error) {
	j, strictErr := unmarshalVMess(payload)
	if strictErr == nil && (j.Add != "" || j.ID != "" || j.Port != "") {
		return j, nil
	}

	limit := maxVMessRepair
	if len(payload) <= limit {
		limit = len(payload) - 1
	}
	for trim := 1; trim <= limit; trim++ {
		candidate, err := unmarshalVMess(payload[:len(payload)-trim])
		if err != nil {
			continue
		}
		if candidate.Add != "" && candidate.ID != "" {
			return candidate, nil
		}
	}

	if strictErr != nil {
		return nil, strictErr
	}
	return j, nil
}

func unmarshalVMess(payload string) (*vmessJSON, error) {
	b, err := DecodeBase64(payload)
	if err != nil {
		return nil, err
	}
	var j vmessJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return nil, fmt.Errorf("%w: invalid vmess JSON: %v", errors.ErrInvalidFormat, err)
	}
	return &j, nil
}
