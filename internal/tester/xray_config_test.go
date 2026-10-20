package tester

import (
	"strings"
	"testing"

	"github.com/amirrezaalavi/Viberayd/internal/models"
)

func base(proto models.Protocol) models.BaseConfig {
	return models.BaseConfig{Server: "1.2.3.4", Port: 443, Protocol: proto}
}

// TestBuildXrayConfigRejectsProtocolsXrayLacks pins the honest classification
// of protocols xray-core cannot load as an outbound. Verified against Xray
// 26.7.11, where `xray -test` fails both with "unknown config id".
func TestBuildXrayConfigRejectsProtocolsXrayLacks(t *testing.T) {
	tuic := models.ProxyConfig{TUIC: &models.TUICConfig{BaseConfig: base(models.ProtocolTUIC), UUID: "u", Password: "p"}}
	if _, err := buildXrayConfig(tuic, 10800); err == nil {
		t.Error("tuic should be rejected: xray-core has no tuic outbound")
	} else if !strings.Contains(err.Error(), "xray-core") {
		t.Errorf("tuic error should explain the cause, got: %v", err)
	}

	hy2 := models.ProxyConfig{Hysteria2: &models.Hysteria2Config{BaseConfig: base(models.ProtocolHysteria2), Auth: "x"}}
	if _, err := buildXrayConfig(hy2, 10801); err == nil {
		t.Error("hysteria2 should be rejected: xray-core has no hysteria2 outbound")
	}
}

func TestBuildXrayConfigAcceptsWireGuard(t *testing.T) {
	wg := models.ProxyConfig{WireGuard: &models.WireGuardConfig{
		BaseConfig: base(models.ProtocolWireGuard),
		PrivateKey: "private",
		PublicKey:  "public",
	}}
	if _, err := buildXrayConfig(wg, 10802); err != nil {
		t.Errorf("wireguard is a real xray outbound and must build: %v", err)
	}
}
