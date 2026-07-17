# Parser Package — Agent Guide

This package forms the extraction engine of VibeRay. It identifies, decodes, extracts, and validates raw proxy configurations from URIs, subscriptions, or text files.

## Purpose

To provide highly specific parsing support for Shadowsocks (`ss://`), VMess (`vmess://`), VLESS (`vless://`), Trojan (`trojan://`), and Reality (detected via `vless://` URLs matching specific parameters) formats. A core design pattern is **failing fast** — any malformed URI triggers immediate validation errors to prevent invalid nodes from entering the test pipeline.

---

## Files

- `parser.go`: Contains unified entry points `Parse()` (which handles base64 batch subscriptions and multiple lines) and `ParseSingle()` (which parses a single configuration link).
- `detector.go`: Handles sniffing protocol prefixes, base64 detection helpers, and `#name` fragment extraction.
- `validator.go`: Standardized structure format checkers (UUID, port ranges, Reality public keys, TLS fingerprint types, network types).
- `ss.go`: Parses Shadowsocks SIP002 URIs and legacy base64 formats.
- `vmess.go`: Parses VMess base64 JSON structures.
- `vless.go`: Parses VLESS URL-parameter links.
- `trojan.go`: Parses Trojan links.
- `reality.go`: Detects and extracts REALITY-specific parameters (e.g. public keys, short IDs, spiderX destinations).
- `parser_test.go`: Comprehensive test suites validating parser correctness against mock proxy URIs.

---

## Shared Utilities & Handlers

### 1. Unified Batch Parser (`Parse`)

Accepts a multiline string or single URI.
- **Base64 Subscriptions:** Sniffs if the entire string looks like base64. If yes, it decodes it first before processing (this handles subscription list URLs natively).
- **Line Scanner:** Splits text into non-empty lines, skips comment lines starting with `#`, and invokes `ParseSingle`.

### 2. Protocol sniffer (`detector.go`)

- **`DetectProtocol(raw)`**: Extracts the scheme prefix (e.g., `ss://`, `vmess://`) and returns a typed `Protocol`.
- **`IsRealityURL(raw)`**: Looks for `security=reality` inside VLESS parameters to isolate Reality configurations.
- **`LooksLikeBase64(s)`**: Uses regular expressions to check if a payload contains exclusively base64 character sets.

---

## Validation Layer (`validator.go`)

To avoid feeding bad structures to testers, validation checks run inline during parsing:
- **`ValidateUUID(uuid)`**: Regex checks for standard `8-4-4-4-12` formats or condensed 32-char sequences.
- **`ValidatePort(port)`**: Ensures ports are in the integer range `1–65535`.
- **`ValidatePublicKey(key)`**: Decodes the base64 string and checks for exactly a 32-byte (x25519) key.
- **`ValidateFingerprint(fp)`**: Checks if the fingerprint is supported by Xray (e.g., `chrome`, `firefox`, `safari`, etc.).
- **`ValidateFlow(flow)`**: Restricts flow values to allowed XTLS sequences.
- **`ValidateNetwork(net)`**: Standardizes network transport values (`tcp`, `ws`, `grpc`, etc.).

---

## Conventions & Gotchas

- **Store Raw URI:** Always store the unparsed, incoming URI string inside the `ProxyConfig.Raw` field during parsing. The downstream formatters rely on this field to output working nodes.
- **No Hard DNS Failures at Parse Time:** `ValidateIPOrHost` does a lookup check, but will not fail hard if the domain name doesn't resolve. DNS resolution may work later during pipeline test execution (since nameservers could change, or a VPN might route them differently).
