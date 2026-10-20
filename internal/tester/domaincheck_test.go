package tester

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// fakeSocks5Server is a minimal no-auth SOCKS5 CONNECT server for hermetic
// tests: it accepts the greeting, dials the requested target itself, and
// pipes bytes. It speaks just enough RFC 1928 for http.Transport's SOCKS5
// dialer (IPv4 / domain / IPv6 CONNECT, success path only).
func fakeSocks5Server(t *testing.T) (addr string, close func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen socks5: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handleFakeSocks5(c)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func handleFakeSocks5(c net.Conn) {
	defer c.Close()
	// Greeting: VER, NMETHODS, METHODS...
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c, hdr); err != nil {
		return
	}
	if _, err := io.CopyN(io.Discard, c, int64(hdr[1])); err != nil {
		return
	}
	if _, err := c.Write([]byte{0x05, 0x00}); err != nil { // no auth
		return
	}
	// Request: VER CMD RSV ATYP ADDR PORT
	req := make([]byte, 4)
	if _, err := io.ReadFull(c, req); err != nil {
		return
	}
	if req[1] != 0x01 { // CONNECT only
		return
	}
	var host string
	switch req[3] {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(c, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	case 0x03: // domain
		var l [1]byte
		if _, err := io.ReadFull(c, l[:]); err != nil {
			return
		}
		name := make([]byte, l[0])
		if _, err := io.ReadFull(c, name); err != nil {
			return
		}
		host = string(name)
	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(c, ip); err != nil {
			return
		}
		host = net.IP(ip).String()
	default:
		return
	}
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(c, portBytes); err != nil {
		return
	}
	port := int(portBytes[0])<<8 | int(portBytes[1])

	upstream, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 5*time.Second)
	if err != nil {
		return
	}
	defer upstream.Close()
	if _, err := c.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	go func() { _, _ = io.Copy(upstream, c) }()
	_, _ = io.Copy(c, upstream)
}

// bodyServer returns an httptest server replying code with body.
func bodyServer(code int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = io.WriteString(w, body)
	}))
}

func TestCheckDomainsQuorumPass(t *testing.T) {
	socks, closeSocks := fakeSocks5Server(t)
	defer closeSocks()
	ok1 := bodyServer(200, "hello world, this is a body")
	defer ok1.Close()
	ok2 := bodyServer(200, "another fine body here")
	defer ok2.Close()
	blocked := bodyServer(403, "forbidden block page with a body")
	defer blocked.Close()

	res := CheckDomains(context.Background(), socks,
		[]string{ok1.URL, ok2.URL, blocked.URL}, 5*time.Second, 4, 2)
	if !res.Passed {
		t.Fatalf("expected pass (2 ok + 1 blocked, quorum 2), got %+v", res)
	}
	if res.PassedCount != 2 {
		t.Fatalf("PassedCount = %d, want 2", res.PassedCount)
	}
}

func TestCheckDomainsQuorumFail(t *testing.T) {
	socks, closeSocks := fakeSocks5Server(t)
	defer closeSocks()
	ok := bodyServer(200, "only good body")
	defer ok.Close()
	bad1 := bodyServer(403, "blocked one")
	defer bad1.Close()
	bad2 := bodyServer(403, "blocked two")
	defer bad2.Close()

	res := CheckDomains(context.Background(), socks,
		[]string{ok.URL, bad1.URL, bad2.URL}, 5*time.Second, 4, 2)
	if res.Passed {
		t.Fatalf("expected fail (1 ok + 2 blocked, quorum 2), got %+v", res)
	}
}

func TestCheckDomainsRejectsEmptyBody(t *testing.T) {
	socks, closeSocks := fakeSocks5Server(t)
	defer closeSocks()
	empty := bodyServer(200, "")
	defer empty.Close()

	res := CheckDomains(context.Background(), socks,
		[]string{empty.URL}, 5*time.Second, 4, 1)
	if res.Passed {
		t.Fatalf("expected fail on empty 200 body, got %+v", res)
	}
}

func TestCheckDomainsTimeout(t *testing.T) {
	socks, closeSocks := fakeSocks5Server(t)
	defer closeSocks()
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		_, _ = io.WriteString(w, "too late")
	}))
	defer slow.Close()

	res := CheckDomains(context.Background(), socks,
		[]string{slow.URL}, 200*time.Millisecond, 1, 1)
	if res.Passed {
		t.Fatalf("expected fail on slow domain, got %+v", res)
	}
	if len(res.Results) != 1 || res.Results[0].Error == "" {
		t.Fatalf("expected recorded error, got %+v", res.Results)
	}
}

func TestCheckDomainsBadProxy(t *testing.T) {
	ok := bodyServer(200, "fine body")
	defer ok.Close()

	// Nothing listens here: every domain must fail, run must not pass.
	res := CheckDomains(context.Background(), "127.0.0.1:1",
		[]string{ok.URL}, 2*time.Second, 1, 1)
	if res.Passed {
		t.Fatalf("expected fail on dead proxy, got %+v", res)
	}
}

func TestCheckDomainsEmpty(t *testing.T) {
	res := CheckDomains(context.Background(), "127.0.0.1:1", nil, time.Second, 1, 1)
	if res.Passed || res.Checked != 0 {
		t.Fatalf("expected empty fail-closed result, got %+v", res)
	}
}

func TestCheckDomainsRealisticURLForms(t *testing.T) {
	socks, closeSocks := fakeSocks5Server(t)
	defer closeSocks()
	srv := bodyServer(200, "some body here")
	defer srv.Close()

	hostport := srv.URL[len("http://"):]
	for _, form := range []string{srv.URL, "http://" + hostport} {
		res := CheckDomains(context.Background(), socks,
			[]string{form}, 5*time.Second, 4, 1)
		if !res.Passed {
			t.Fatalf("form %q: expected pass, got %+v", form, res)
		}
	}
}
