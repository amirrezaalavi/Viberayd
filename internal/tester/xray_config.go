package tester

import (
	"fmt"

	"github.com/amiralavi/viberay/internal/models"
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
