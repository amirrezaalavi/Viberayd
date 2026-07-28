package models

import (
	"fmt"
	"net"
	"strconv"
)

// Protocol identifies the proxy type.
type Protocol string

const (
	ProtocolSS      Protocol = "ss"
	ProtocolVMess   Protocol = "vmess"
	ProtocolVLess   Protocol = "vless"
	ProtocolTrojan  Protocol = "trojan"
	ProtocolReality Protocol = "reality"
	ProtocolWireGuard Protocol = "wireguard"
	ProtocolTUIC      Protocol = "tuic"
	ProtocolHysteria2 Protocol = "hysteria2"
	ProtocolSocks5    Protocol = "socks5"
)

// IsValid reports whether p is a known protocol.
func (p Protocol) IsValid() bool {
	switch p {
	case ProtocolSS, ProtocolVMess, ProtocolVLess, ProtocolTrojan, ProtocolReality,
		ProtocolWireGuard, ProtocolTUIC, ProtocolHysteria2, ProtocolSocks5:
		return true
	}
	return false
}

// BaseConfig holds fields shared by all proxy configurations.
type BaseConfig struct {
	Server   string   `json:"server" yaml:"server"`
	Port     int      `json:"port" yaml:"port"`
	Protocol Protocol `json:"protocol" yaml:"protocol"`
	Name     string   `json:"name,omitempty" yaml:"name,omitempty"`
	Remarks  string   `json:"remarks,omitempty" yaml:"remarks,omitempty"`
	Network  string   `json:"network,omitempty" yaml:"network,omitempty"` // tcp, ws, grpc, h2, http, quic, kcp
}

// Addr returns "host:port" suitable for net.Dial.
func (b BaseConfig) Addr() string {
	return net.JoinHostPort(b.Server, strconv.Itoa(b.Port))
}

// TLSConfig holds common TLS / transport-layer settings.
type TLSConfig struct {
	Enabled     bool   `json:"tls,omitempty" yaml:"tls,omitempty"`
	SNI         string `json:"sni,omitempty" yaml:"sni,omitempty"`
	Host        string `json:"host,omitempty" yaml:"host,omitempty"`
	Path        string `json:"path,omitempty" yaml:"path,omitempty"`
	ALPN        string `json:"alpn,omitempty" yaml:"alpn,omitempty"`
	SkipVerify  bool   `json:"skip_verify,omitempty" yaml:"skip_verify,omitempty"`
	Fingerprint string `json:"fp,omitempty" yaml:"fp,omitempty"` // e.g. chrome, firefox, safari, ios
}

// SSConfig represents a Shadowsocks configuration.
type SSConfig struct {
	BaseConfig
	Method     string            `json:"method" yaml:"method"`                         // e.g. aes-256-gcm, chacha20-ietf-poly1305
	Password   string            `json:"password" yaml:"password"`
	Plugin     string            `json:"plugin,omitempty" yaml:"plugin,omitempty"`     // e.g. obfs-local, v2ray-plugin
	PluginOpts map[string]string `json:"plugin_opts,omitempty" yaml:"plugin_opts,omitempty"`
}

func (s *SSConfig) String() string {
	return fmt.Sprintf("SS[%s:%d %s]", s.Server, s.Port, s.Method)
}

// VMessConfig represents a VMess configuration.
type VMessConfig struct {
	BaseConfig
	TLSConfig
	UUID     string `json:"uuid" yaml:"uuid"`
	AlterID  int    `json:"aid,omitempty" yaml:"aid,omitempty"`
	Security string `json:"security,omitempty" yaml:"security,omitempty"` // auto, none, aes-128-gcm, chacha20-poly1305, zero
}

func (v *VMessConfig) String() string {
	return fmt.Sprintf("VMess[%s:%d %s]", v.Server, v.Port, v.UUID)
}

// VLessConfig represents a VLESS configuration.
type VLessConfig struct {
	BaseConfig
	TLSConfig
	UUID       string `json:"uuid" yaml:"uuid"`
	Flow       string `json:"flow,omitempty" yaml:"flow,omitempty"`             // e.g. xtls-rprx-vision, xtls-rprx-vision-udp443
	Encryption string `json:"encryption,omitempty" yaml:"encryption,omitempty"` // typically "none"
}

func (v *VLessConfig) String() string {
	return fmt.Sprintf("VLess[%s:%d %s]", v.Server, v.Port, v.UUID)
}

// TrojanConfig represents a Trojan / Trojan-Go configuration.
type TrojanConfig struct {
	BaseConfig
	TLSConfig
	Password string `json:"password" yaml:"password"`
	Flow     string `json:"flow,omitempty" yaml:"flow,omitempty"`
}

func (t *TrojanConfig) String() string {
	return fmt.Sprintf("Trojan[%s:%d]", t.Server, t.Port)
}

// RealityConfig represents a REALITY-enabled VLESS configuration.
type RealityConfig struct {
	BaseConfig
	TLSConfig
	UUID      string `json:"uuid" yaml:"uuid"`
	Flow      string `json:"flow,omitempty" yaml:"flow,omitempty"`
	PublicKey string `json:"public_key" yaml:"public_key"`          // base64-encoded x25519 public key
	ShortID   string `json:"short_id,omitempty" yaml:"short_id,omitempty"`
	SpiderX   string `json:"spider_x,omitempty" yaml:"spider_x,omitempty"` // target path / URL
}

func (r *RealityConfig) String() string {
	return fmt.Sprintf("Reality[%s:%d %s]", r.Server, r.Port, r.UUID)
}


// WireGuardConfig represents a WireGuard configuration.
type WireGuardConfig struct {
	BaseConfig
	PrivateKey   string
	PublicKey    string
	PresharedKey string
	LocalAddress string
	DNS          string
	MTU          int
	Reserved     string
	AllowedIPs   string
	KernelMode   bool
}

func (w *WireGuardConfig) String() string {
	return fmt.Sprintf("WireGuard[%s:%d]", w.Server, w.Port)
}

// TUICConfig represents a TUIC configuration.
type TUICConfig struct {
	BaseConfig
	TLSConfig
	UUID              string
	Password          string
	CongestionControl string
	UDPRelayMode      string
	Heartbeat         string
	ReduceRTT         bool
	RequestTimeout    string
}

func (t *TUICConfig) String() string {
	return fmt.Sprintf("TUIC[%s:%d]", t.Server, t.Port)
}

// Hysteria2Config represents a Hysteria2 configuration.
type Hysteria2Config struct {
	BaseConfig
	TLSConfig
	Auth         string
	ObfsType     string
	ObfsPassword string
	UpMbps       int
	DownMbps     int
	Ports        string
	HopInterval  int
}

func (h *Hysteria2Config) String() string {
	return fmt.Sprintf("Hysteria2[%s:%d]", h.Server, h.Port)
}

// Socks5Config represents a SOCKS5 configuration.
type Socks5Config struct {
	BaseConfig
	Username string
	Password string
	UDP      bool
}

func (s *Socks5Config) String() string {
	return fmt.Sprintf("Socks5[%s:%d]", s.Server, s.Port)
}

// ProxyConfig is a tagged-union that can hold any supported configuration type.
// Only one pointer field is non-nil at a time.
type ProxyConfig struct {
	SS      *SSConfig
	VMess   *VMessConfig
	VLess   *VLessConfig
	Trojan  *TrojanConfig
	Reality   *RealityConfig
	WireGuard *WireGuardConfig
	TUIC      *TUICConfig
	Hysteria2 *Hysteria2Config
	Socks5    *Socks5Config
	Raw     string `json:"raw,omitempty" yaml:"raw,omitempty"` // original input URI
}

// Protocol returns the detected protocol, or empty string if unset.
func (p ProxyConfig) Protocol() Protocol {
	switch {
	case p.SS != nil:
		return ProtocolSS
	case p.VMess != nil:
		return ProtocolVMess
	case p.VLess != nil:
		return ProtocolVLess
	case p.Trojan != nil:
		return ProtocolTrojan
	case p.Reality != nil:
		return ProtocolReality
	case p.WireGuard != nil:
		return ProtocolWireGuard
	case p.TUIC != nil:
		return ProtocolTUIC
	case p.Hysteria2 != nil:
		return ProtocolHysteria2
	case p.Socks5 != nil:
		return ProtocolSocks5
	}
	return ""
}

// Name returns the human-readable name/remarks of the underlying config.
func (p ProxyConfig) Name() string {
	switch {
	case p.SS != nil:
		return p.SS.Name
	case p.VMess != nil:
		return p.VMess.Name
	case p.VLess != nil:
		return p.VLess.Name
	case p.Trojan != nil:
		return p.Trojan.Name
	case p.Reality != nil:
		return p.Reality.Name
	case p.WireGuard != nil:
		return p.WireGuard.Name
	case p.TUIC != nil:
		return p.TUIC.Name
	case p.Hysteria2 != nil:
		return p.Hysteria2.Name
	case p.Socks5 != nil:
		return p.Socks5.Name
	}
	return ""
}

// Addr returns the "host:port" address of the underlying config.
func (p ProxyConfig) Addr() string {
	switch {
	case p.SS != nil:
		return p.SS.Addr()
	case p.VMess != nil:
		return p.VMess.Addr()
	case p.VLess != nil:
		return p.VLess.Addr()
	case p.Trojan != nil:
		return p.Trojan.Addr()
	case p.Reality != nil:
		return p.Reality.Addr()
	case p.WireGuard != nil:
		return p.WireGuard.Addr()
	case p.TUIC != nil:
		return p.TUIC.Addr()
	case p.Hysteria2 != nil:
		return p.Hysteria2.Addr()
	case p.Socks5 != nil:
		return p.Socks5.Addr()
	}
	return ""
}

// String returns a short string representation.
func (p ProxyConfig) String() string {
	switch {
	case p.SS != nil:
		return p.SS.String()
	case p.VMess != nil:
		return p.VMess.String()
	case p.VLess != nil:
		return p.VLess.String()
	case p.Trojan != nil:
		return p.Trojan.String()
	case p.Reality != nil:
		return p.Reality.String()
	case p.WireGuard != nil:
		return p.WireGuard.String()
	case p.TUIC != nil:
		return p.TUIC.String()
	case p.Hysteria2 != nil:
		return p.Hysteria2.String()
	case p.Socks5 != nil:
		return p.Socks5.String()
	}
	return "ProxyConfig<empty>"
}

// Base returns the embedded BaseConfig, or zero value if unset.
func (p ProxyConfig) Base() BaseConfig {
	switch {
	case p.SS != nil:
		return p.SS.BaseConfig
	case p.VMess != nil:
		return p.VMess.BaseConfig
	case p.VLess != nil:
		return p.VLess.BaseConfig
	case p.Trojan != nil:
		return p.Trojan.BaseConfig
	case p.Reality != nil:
		return p.Reality.BaseConfig
	case p.WireGuard != nil:
		return p.WireGuard.BaseConfig
	case p.TUIC != nil:
		return p.TUIC.BaseConfig
	case p.Hysteria2 != nil:
		return p.Hysteria2.BaseConfig
	case p.Socks5 != nil:
		return p.Socks5.BaseConfig
	}
	return BaseConfig{}
}
