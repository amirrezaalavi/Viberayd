package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads a testdata fixture by path relative to project root.
func loadFixture(path string) string {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// loadSubscription reads a chunk file and returns all non-empty lines.
func loadSubscription(path string) []string {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

// --- Benchmarks for ParseSingle ---

func BenchmarkParseSingle_SS(b *testing.B) {
	uri := loadFixture("../../testdata/examples/ss.txt")
	if uri == "" {
		b.Skip("ss.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_VMess(b *testing.B) {
	uri := loadFixture("../../testdata/examples/vmess.txt")
	if uri == "" {
		b.Skip("vmess.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_VLess(b *testing.B) {
	uri := loadFixture("../../testdata/examples/vless.txt")
	if uri == "" {
		b.Skip("vless.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_Trojan(b *testing.B) {
	uri := loadFixture("../../testdata/examples/trojan.txt")
	if uri == "" {
		b.Skip("trojan.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_Reality(b *testing.B) {
	uri := loadFixture("../../testdata/examples/reality.txt")
	if uri == "" {
		b.Skip("reality.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

// --- Benchmarks for Parse (batch) ---

func BenchmarkParse_Batch_100(b *testing.B) {
	lines := loadSubscription("../../testdata/subscriptions/chunk_000.txt")
	if len(lines) == 0 {
		b.Skip("chunk_000.txt not found or empty")
	}
	// Take first 100 lines if available
	input := strings.Join(lines[:min(len(lines), 100)], "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(input)
	}
}

func BenchmarkParse_Batch_500(b *testing.B) {
	var all []string
	for _, chunk := range []string{"chunk_000.txt", "chunk_001.txt", "chunk_002.txt", "chunk_003.txt"} {
		lines := loadSubscription("../../testdata/subscriptions/" + chunk)
		all = append(all, lines...)
	}
	if len(all) < 500 {
		b.Skipf("need at least 500 lines, got %d", len(all))
	}
	input := strings.Join(all[:500], "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(input)
	}
}

func BenchmarkParse_Batch_2000(b *testing.B) {
	var all []string
	for _, chunk := range []string{"chunk_000.txt", "chunk_001.txt", "chunk_002.txt", "chunk_003.txt", "chunk_004.txt"} {
		lines := loadSubscription("../../testdata/subscriptions/" + chunk)
		all = append(all, lines...)
	}
	if len(all) < 100 {
		b.Skipf("need at least 100 lines, got %d", len(all))
	}
	// Use all available lines (up to 2000)
	n := len(all)
	if n > 2000 {
		n = 2000
	}
	input := strings.Join(all[:n], "\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(input)
	}
}

// --- Benchmarks for DetectProtocol ---

func BenchmarkParseSingle_WireGuard(b *testing.B) {
	uri := loadFixture("../../testdata/examples/wireguard.txt")
	if uri == "" {
		b.Skip("wireguard.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_TUIC(b *testing.B) {
	uri := loadFixture("../../testdata/examples/tuic.txt")
	if uri == "" {
		b.Skip("tuic.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_Hysteria2(b *testing.B) {
	uri := loadFixture("../../testdata/examples/hysteria2.txt")
	if uri == "" {
		b.Skip("hysteria2.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

func BenchmarkParseSingle_Socks5(b *testing.B) {
	uri := loadFixture("../../testdata/examples/socks5.txt")
	if uri == "" {
		b.Skip("socks5.txt fixture not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseSingle(uri)
	}
}

// --- Benchmarks for DetectProtocol ---

func BenchmarkDetectProtocol(b *testing.B) {
	uris := loadSubscription("../../testdata/subscriptions/chunk_000.txt")
	if len(uris) == 0 {
		b.Skip("chunk_000.txt not found")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, u := range uris {
			_, _ = DetectProtocol(u)
		}
	}
}
