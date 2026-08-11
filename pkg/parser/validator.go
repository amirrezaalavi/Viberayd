package parser

import (
	"encoding/base64"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberayd/internal/errors"
	"github.com/amirrezaalavi/Viberayd/internal/models"
)

var (
	uuidRe  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	uuidRe2 = regexp.MustCompile(`^[0-9a-fA-F]{32}$`) // some tools omit dashes
)

// ValidNetworkTypes lists accepted network values.
var ValidNetworkTypes = map[string]bool{
	"tcp": true, "kcp": true, "ws": true, "grpc": true,
	"http": true, "h2": true, "quic": true, "xhttp": true,
}

// ValidSecurityTypes lists accepted VMess security values.
var ValidSecurityTypes = map[string]bool{
	"auto": true, "none": true, "zero": true,
	"aes-128-gcm": true, "chacha20-poly1305": true,
}

// ValidFlowValues lists accepted VLESS / Trojan flow values.
var ValidFlowValues = map[string]bool{
	"":                        true,
	"xtls-rprx-direct":        true,
	"xtls-rprx-origin":        true,
	"xtls-rprx-vision":        true,
	"xtls-rprx-vision-udp443": true,
}

// ValidFingerprints lists accepted TLS fingerprint values.
var ValidFingerprints = map[string]bool{
	"":       true,
	"chrome": true, "firefox": true, "safari": true,
	"ios": true, "android": true, "edge": true, "360": true,
	"qq": true, "random": true, "randomized": true,
}

// ValidPlugins lists accepted Shadowsocks plugin names.
var ValidPlugins = map[string]bool{
	"":             true,
	"obfs-local":   true,
	"v2ray-plugin": true,
	"simple-obfs":  true,
}

// ValidateUUID checks standard 8-4-4-4-12 format (and 32-char fallback).
func ValidateUUID(s string) error {
	if uuidRe.MatchString(s) || uuidRe2.MatchString(s) {
		return nil
	}
	return errors.ErrInvalidUUID
}

// ValidatePort checks range 1–65535.
func ValidatePort(s string) error {
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 65535 {
		return errors.ErrInvalidPort
	}
	return nil
}

// ValidatePortInt is the int variant.
func ValidatePortInt(p int) error {
	if p < 1 || p > 65535 {
		return errors.ErrInvalidPort
	}
	return nil
}

// ValidatePublicKey checks that s is a valid base64 string (Reality x25519 keys are 32 bytes → 43 chars base64).
func ValidatePublicKey(s string) error {
	if len(s) == 0 {
		return errors.ErrInvalidPublicKey
	}
	// Normalize for URL-safe base64 and handle unpadded input
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return errors.ErrInvalidPublicKey
	}
	if len(b) != 32 {
		return errors.ErrInvalidPublicKey
	}
	return nil
}

// ValidateFingerprint checks fp against known values.
func ValidateFingerprint(s string) error {
	if ValidFingerprints[s] {
		return nil
	}
	return errors.ErrInvalidFingerprint
}

// ValidateFlow checks flow against known values.
func ValidateFlow(s string) error {
	if ValidFlowValues[s] {
		return nil
	}
	return errors.ErrInvalidFlow
}

// ValidateNetwork checks network type.
func ValidateNetwork(s string) error {
	if ValidNetworkTypes[s] {
		return nil
	}
	return errors.ErrInvalidNetwork
}

// ValidateSecurity checks VMess security type.
func ValidateSecurity(s string) error {
	if ValidSecurityTypes[s] {
		return nil
	}
	return errors.ErrInvalidSecurity
}

// ValidatePlugin checks Shadowsocks plugin name.
func ValidatePlugin(s string) error {
	if ValidPlugins[s] {
		return nil
	}
	return errors.ErrInvalidFormat
}

// ValidateHost checks that the server address is not empty.
func ValidateHost(s string) error {
	if s == "" {
		return errors.ErrMissingField
	}
	return nil
}

// ValidateIPOrHost checks that s is a valid hostname or IP.
func ValidateIPOrHost(s string) error {
	if s == "" {
		return errors.ErrMissingField
	}
	if net.ParseIP(s) != nil {
		return nil
	}
	if _, err := net.LookupHost(s); err == nil {
		return nil
	}
	// Don't fail hard on unresolvable hosts at parse time — DNS may work later
	return nil
}

// ValidateBaseConfig runs common validators on BaseConfig fields.
func ValidateBaseConfig(b models.BaseConfig) models.ValidationResult {
	vr := models.ValidationResult{Valid: true}
	if b.Server == "" {
		vr.AddError("server address is required")
	}
	if err := ValidatePortInt(b.Port); err != nil {
		vr.AddError("invalid port: " + strconv.Itoa(b.Port))
	}
	if b.Protocol != "" && !b.Protocol.IsValid() {
		vr.AddError("invalid protocol: " + string(b.Protocol))
	}
	if b.Network != "" {
		if err := ValidateNetwork(b.Network); err != nil {
			vr.AddWarning("unusual network type: " + b.Network)
		}
	}
	return vr
}
