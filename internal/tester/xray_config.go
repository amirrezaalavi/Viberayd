package tester

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/amirrezaalavi/Viberay/internal/models"
)

// buildXrayConfig generates a minimal xray-core JSON config that exposes a
// SOCKS5 inbound on socksPort and routes all traffic through the outbound
// derived from cfg.
func buildXrayConfig(cfg models.ProxyConfig, socksPort int) (map[string]any, error) {
	outbound, err := buildOutbound(cfg)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"log": map[string]any{
			"loglevel": "warning",
		},
		"inbounds": []map[string]any{
			{
				"tag":      "socks-in",
				"protocol": "socks",
				"listen":   "127.0.0.1",
				"port":     socksPort,
				"settings": map[string]any{
					"auth": "noauth",
					"udp":  true,
				},
				"sniffing": map[string]any{
					"enabled":      true,
					"destOverride": []string{"http", "tls", "quic"},
				},
			},
		},
		"outbounds": []map[string]any{
			outbound,
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
			{
				"tag":      "block",
				"protocol": "blackhole",
			},
		},
		"routing": map[string]any{
			"rules": []map[string]any{
				{
					"type":        "field",
					"outboundTag": "proxy",
					"network":     "tcp,udp",
				},
			},
		},
	}, nil
}

// buildOutbound converts a ProxyConfig into an xray outbound object.
func buildOutbound(cfg models.ProxyConfig) (map[string]any, error) {
	switch cfg.Protocol() {
	case models.ProtocolSS:
		return buildSSOutbound(cfg.SS), nil
	case models.ProtocolVMess:
		return buildVMessOutbound(cfg.VMess), nil
	case models.ProtocolVLess:
		return buildVLessOutbound(cfg.VLess), nil
	case models.ProtocolTrojan:
		return buildTrojanOutbound(cfg.Trojan), nil
	case models.ProtocolReality:
		return buildRealityOutbound(cfg.Reality), nil
	case models.ProtocolWireGuard:
		return buildWireGuardOutbound(cfg.WireGuard), nil
	case models.ProtocolTUIC:
		return buildTUICOutbound(cfg.TUIC), nil
	case models.ProtocolHysteria2:
		return buildHysteria2Outbound(cfg.Hysteria2), nil
	case models.ProtocolSocks5:
		return buildSocks5Outbound(cfg.Socks5), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", cfg.Protocol())
	}
}

func buildSSOutbound(cfg *models.SSConfig) map[string]any {
	servers := []map[string]any{
		{
			"address":  cfg.Server,
			"port":     cfg.Port,
			"method":   cfg.Method,
			"password": cfg.Password,
		},
	}
	if cfg.Plugin != "" {
		servers[0]["plugin"] = cfg.Plugin
		if len(cfg.PluginOpts) > 0 {
			servers[0]["pluginOpts"] = cfg.PluginOpts
		}
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "shadowsocks",
		"settings": map[string]any{
			"servers": servers,
		},
	}
}

func buildVMessOutbound(cfg *models.VMessConfig) map[string]any {
	vnext := map[string]any{
		"address": cfg.Server,
		"port":    cfg.Port,
		"users": []map[string]any{
			{
				"id":       cfg.UUID,
				"alterId":  cfg.AlterID,
				"security": cfg.Security,
				"level":    0,
			},
		},
	}

	stream := map[string]any{
		"network": cfg.Network,
	}
	if cfg.Network == "" {
		stream["network"] = "tcp"
	}
	if cfg.Enabled {
		stream["security"] = "tls"
		tlsSettings := map[string]any{}
		if cfg.SNI != "" {
			tlsSettings["serverName"] = cfg.SNI
		}
		if cfg.Fingerprint != "" {
			tlsSettings["fingerprint"] = cfg.Fingerprint
		}
		if cfg.ALPN != "" {
			tlsSettings["alpn"] = []string{cfg.ALPN}
		}
		if cfg.SkipVerify {
			tlsSettings["allowInsecure"] = true
		}
		if len(tlsSettings) > 0 {
			stream["tlsSettings"] = tlsSettings
		}
	}
	if cfg.Host != "" || cfg.Path != "" {
		switch cfg.Network {
		case "ws":
			stream["wsSettings"] = map[string]any{
				"path": cfg.Path,
				"headers": map[string]any{
					"Host": cfg.Host,
				},
			}
		case "grpc":
			stream["grpcSettings"] = map[string]any{
				"serviceName": cfg.Path,
			}
		case "h2", "http":
			stream["httpSettings"] = map[string]any{
				"path": cfg.Path,
				"host": []string{cfg.Host},
			}
		}
	}

	return map[string]any{
		"tag":      "proxy",
		"protocol": "vmess",
		"settings": map[string]any{
			"vnext": []map[string]any{vnext},
		},
		"streamSettings": stream,
	}
}

func buildVLessOutbound(cfg *models.VLessConfig) map[string]any {
	vnext := map[string]any{
		"address": cfg.Server,
		"port":    cfg.Port,
		"users": []map[string]any{
			{
				"id":         cfg.UUID,
				"flow":       cfg.Flow,
				"encryption": cfg.Encryption,
				"level":      0,
			},
		},
	}

	stream := map[string]any{
		"network": cfg.Network,
	}
	if cfg.Network == "" {
		stream["network"] = "tcp"
	}
	if cfg.Enabled {
		stream["security"] = "tls"
		tlsSettings := map[string]any{}
		if cfg.SNI != "" {
			tlsSettings["serverName"] = cfg.SNI
		}
		if cfg.Fingerprint != "" {
			tlsSettings["fingerprint"] = cfg.Fingerprint
		}
		if cfg.ALPN != "" {
			tlsSettings["alpn"] = []string{cfg.ALPN}
		}
		if cfg.SkipVerify {
			tlsSettings["allowInsecure"] = true
		}
		if len(tlsSettings) > 0 {
			stream["tlsSettings"] = tlsSettings
		}
	}
	if cfg.Host != "" || cfg.Path != "" {
		switch cfg.Network {
		case "ws":
			stream["wsSettings"] = map[string]any{
				"path": cfg.Path,
				"headers": map[string]any{
					"Host": cfg.Host,
				},
			}
		case "grpc":
			stream["grpcSettings"] = map[string]any{
				"serviceName": cfg.Path,
			}
		case "h2", "http":
			stream["httpSettings"] = map[string]any{
				"path": cfg.Path,
				"host": []string{cfg.Host},
			}
		}
	}

	return map[string]any{
		"tag":      "proxy",
		"protocol": "vless",
		"settings": map[string]any{
			"vnext": []map[string]any{vnext},
		},
		"streamSettings": stream,
	}
}

func buildTrojanOutbound(cfg *models.TrojanConfig) map[string]any {
	servers := []map[string]any{
		{
			"address":  cfg.Server,
			"port":     cfg.Port,
			"password": cfg.Password,
			"level":    0,
			"flow":     cfg.Flow,
		},
	}

	stream := map[string]any{
		"network":  cfg.Network,
		"security": "tls",
	}
	if cfg.Network == "" {
		stream["network"] = "tcp"
	}
	tlsSettings := map[string]any{}
	if cfg.SNI != "" {
		tlsSettings["serverName"] = cfg.SNI
	}
	if cfg.Fingerprint != "" {
		tlsSettings["fingerprint"] = cfg.Fingerprint
	}
	if cfg.ALPN != "" {
		tlsSettings["alpn"] = []string{cfg.ALPN}
	}
	if cfg.SkipVerify {
		tlsSettings["allowInsecure"] = true
	}
	if len(tlsSettings) > 0 {
		stream["tlsSettings"] = tlsSettings
	}
	if cfg.Host != "" || cfg.Path != "" {
		switch cfg.Network {
		case "ws":
			stream["wsSettings"] = map[string]any{
				"path": cfg.Path,
				"headers": map[string]any{
					"Host": cfg.Host,
				},
			}
		case "grpc":
			stream["grpcSettings"] = map[string]any{
				"serviceName": cfg.Path,
			}
		case "h2", "http":
			stream["httpSettings"] = map[string]any{
				"path": cfg.Path,
				"host": []string{cfg.Host},
			}
		}
	}

	return map[string]any{
		"tag":      "proxy",
		"protocol": "trojan",
		"settings": map[string]any{
			"servers": servers,
		},
		"streamSettings": stream,
	}
}

func buildRealityOutbound(cfg *models.RealityConfig) map[string]any {
	vnext := map[string]any{
		"address": cfg.Server,
		"port":    cfg.Port,
		"users": []map[string]any{
			{
				"id":         cfg.UUID,
				"flow":       cfg.Flow,
				"encryption": "none",
				"level":      0,
			},
		},
	}

	stream := map[string]any{
		"network":  cfg.Network,
		"security": "reality",
	}
	if cfg.Network == "" {
		stream["network"] = "tcp"
	}

	realitySettings := map[string]any{
		"publicKey":   cfg.PublicKey,
		"shortId":     cfg.ShortID,
		"spiderX":     cfg.SpiderX,
		"fingerprint": cfg.Fingerprint,
	}
	if cfg.SNI != "" {
		realitySettings["serverName"] = cfg.SNI
	}
	if realitySettings["fingerprint"] == "" {
		realitySettings["fingerprint"] = "chrome"
	}
	stream["realitySettings"] = realitySettings

	if cfg.Host != "" || cfg.Path != "" {
		switch cfg.Network {
		case "ws":
			stream["wsSettings"] = map[string]any{
				"path": cfg.Path,
				"headers": map[string]any{
					"Host": cfg.Host,
				},
			}
		case "grpc":
			stream["grpcSettings"] = map[string]any{
				"serviceName": cfg.Path,
			}
		case "h2", "http":
			stream["httpSettings"] = map[string]any{
				"path": cfg.Path,
				"host": []string{cfg.Host},
			}
		}
	}

	return map[string]any{
		"tag":      "proxy",
		"protocol": "vless",
		"settings": map[string]any{
			"vnext": []map[string]any{vnext},
		},
		"streamSettings": stream,
	}
}
func buildWireGuardOutbound(cfg *models.WireGuardConfig) map[string]any {
	if cfg == nil {
		return map[string]any{"tag": "proxy", "protocol": "wireguard"}
	}
	reserved := []int{0, 0, 0}
	if cfg.Reserved != "" {
		parts := strings.Split(cfg.Reserved, ",")
		if len(parts) == 3 {
			r0, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			r1, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			r2, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			reserved = []int{r0, r1, r2}
		}
	}
	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = 1420
	}
	address := cfg.LocalAddress
	if address == "" {
		address = "172.16.0.2/32"
	}
	allowedIPs := []string{"0.0.0.0/0", "::/0"}
	if cfg.AllowedIPs != "" {
		allowedIPs = strings.Split(cfg.AllowedIPs, ",")
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "wireguard",
		"settings": map[string]any{
			"secretKey": cfg.PrivateKey,
			"address":   strings.Split(address, ","),
			"peers": []map[string]any{{
				"endpoint":    net.JoinHostPort(cfg.Server, strconv.Itoa(int(cfg.Port))),
				"publicKey":   cfg.PublicKey,
				"preSharedKey": cfg.PresharedKey,
				"allowedIPs":  allowedIPs,
			}},
			"mtu":      mtu,
			"reserved": reserved,
		},
	}
}

func buildTUICOutbound(cfg *models.TUICConfig) map[string]any {
	if cfg == nil {
		return map[string]any{"tag": "proxy", "protocol": "tuic"}
	}
	cc := cfg.CongestionControl
	if cc == "" {
		cc = "bbr"
	}
	udpMode := cfg.UDPRelayMode
	if udpMode == "" {
		udpMode = "native"
	}
	settings := map[string]any{
		"uuid":              cfg.UUID,
		"password":          cfg.Password,
		"congestion_control": cc,
		"udp_relay_mode":    udpMode,
	}
	if cfg.Heartbeat != "" {
		settings["heartbeat"] = cfg.Heartbeat
	}
	if cfg.ReduceRTT {
		settings["reduce_rtt"] = true
	}
	if cfg.RequestTimeout != "" {
		settings["request_timeout"] = cfg.RequestTimeout
	}
	tlsSettings := map[string]any{}
	if cfg.SNI != "" {
		tlsSettings["serverName"] = cfg.SNI
	}
	if cfg.ALPN != "" {
		tlsSettings["alpn"] = []string{cfg.ALPN}
	} else {
		tlsSettings["alpn"] = []string{"h3"}
	}
	if cfg.Fingerprint != "" {
		tlsSettings["fingerprint"] = cfg.Fingerprint
	}
	if cfg.SkipVerify {
		tlsSettings["allowInsecure"] = true
	}
	stream := map[string]any{
		"network":  "quic",
		"security": "tls",
	}
	if len(tlsSettings) > 0 {
		stream["tlsSettings"] = tlsSettings
	}
	return map[string]any{
		"tag":            "proxy",
		"protocol":       "tuic",
		"settings":       settings,
		"streamSettings": stream,
	}
}

func buildHysteria2Outbound(cfg *models.Hysteria2Config) map[string]any {
	if cfg == nil {
		return map[string]any{"tag": "proxy", "protocol": "hysteria2"}
	}
	settings := map[string]any{
		"auth":    cfg.Auth,
		"version": 2,
	}
	if cfg.UpMbps > 0 {
		settings["up"] = strconv.Itoa(cfg.UpMbps) + " mbps"
	}
	if cfg.DownMbps > 0 {
		settings["down"] = strconv.Itoa(cfg.DownMbps) + " mbps"
	}
	if cfg.ObfsType != "" {
		obfs := map[string]any{"type": cfg.ObfsType}
		if cfg.ObfsPassword != "" {
			obfs["password"] = cfg.ObfsPassword
		}
		settings["obfs"] = obfs
	}
	if cfg.Ports != "" {
		settings["ports"] = cfg.Ports
	}
	tlsSettings := map[string]any{}
	if cfg.SNI != "" {
		tlsSettings["serverName"] = cfg.SNI
	}
	if cfg.ALPN != "" {
		tlsSettings["alpn"] = []string{cfg.ALPN}
	} else {
		tlsSettings["alpn"] = []string{"h3"}
	}
	if cfg.Fingerprint != "" {
		tlsSettings["fingerprint"] = cfg.Fingerprint
	}
	if cfg.SkipVerify {
		tlsSettings["allowInsecure"] = true
	}
	stream := map[string]any{
		"network":  "hysteria2",
		"security": "tls",
	}
	if len(tlsSettings) > 0 {
		stream["tlsSettings"] = tlsSettings
	}
	return map[string]any{
		"tag":            "proxy",
		"protocol":       "hysteria2",
		"settings":       settings,
		"streamSettings": stream,
	}
}

func buildSocks5Outbound(cfg *models.Socks5Config) map[string]any {
	if cfg == nil {
		return map[string]any{"tag": "proxy", "protocol": "socks"}
	}
	server := map[string]any{
		"address": cfg.Server,
		"port":    cfg.Port,
	}
	if cfg.Username != "" || cfg.Password != "" {
		server["users"] = []map[string]any{{
			"user": cfg.Username,
			"pass": cfg.Password,
		}}
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "socks",
		"settings": map[string]any{
			"servers": []map[string]any{server},
		},
	}
}

