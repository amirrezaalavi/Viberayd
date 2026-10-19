package tester

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// maxDomainBody caps how much of a domain-check response body is read. The
// check only needs a prefix (status + a body floor), never the whole page.
const maxDomainBody = 64 * 1024

// SingleDomainResult is the outcome for one domain.
type SingleDomainResult struct {
	Domain   string
	HTTPCode int
	Bytes    int64
	Latency  time.Duration
	Error    string // "" when the domain passed
}

// DomainResult aggregates one CheckDomains run.
type DomainResult struct {
	Passed      bool // PassedCount >= quorum
	Checked     int
	PassedCount int
	Results     []SingleDomainResult
}

// CheckDomains fetches every domain through the SOCKS5 proxy at proxyAddr
// (host:port) in parallel. A domain passes only when the fetch returns HTTP
// 2xx with at least minBytes of body inside timeout. The run passes when at
// least quorum domains pass — one exit IP may be blocked by one domain
// without failing an otherwise good proxy. A 403 with a body still fails
// that domain: block pages are not proof of working egress.
func CheckDomains(ctx context.Context, proxyAddr string, domains []string, timeout time.Duration, minBytes int64, quorum int) DomainResult {
	res := DomainResult{Checked: len(domains)}
	if len(domains) == 0 {
		return res
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	proxyURL, err := url.Parse("socks5://" + proxyAddr)
	if err != nil {
		for _, d := range domains {
			res.Results = append(res.Results, SingleDomainResult{Domain: d, Error: fmt.Sprintf("invalid proxy address: %v", err)})
		}
		return res
	}

	results := make([]SingleDomainResult, len(domains))
	var wg sync.WaitGroup
	for i, domain := range domains {
		wg.Add(1)
		go func(i int, domain string) {
			defer wg.Done()
			results[i] = checkOneDomain(ctx, proxyURL, domain, timeout, minBytes)
		}(i, domain)
	}
	wg.Wait()

	res.Results = results
	for _, r := range results {
		if r.Error == "" {
			res.PassedCount++
		}
	}
	res.Passed = res.PassedCount >= quorum
	return res
}

// checkOneDomain fetches a single domain through the proxy. The http.Client
// timeout bounds the whole exchange (stdlib honors it, unlike raw-conn I/O).
func checkOneDomain(ctx context.Context, proxyURL *url.URL, domain string, timeout time.Duration, minBytes int64) SingleDomainResult {
	out := SingleDomainResult{Domain: domain}
	target := domain
	if len(target) < 4 || (target[:7] != "http://" && target[:8] != "https://") {
		target = "https://" + target
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   timeout,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		out.Error = fmt.Sprintf("build request: %v", err)
		return out
	}

	start := time.Now()
	resp, err := client.Do(req)
	out.Latency = time.Since(start)
	if err != nil {
		out.Error = fmt.Sprintf("fetch: %v", err)
		return out
	}
	defer resp.Body.Close()

	out.HTTPCode = resp.StatusCode
	n, _ := io.Copy(io.Discard, io.LimitReader(resp.Body, maxDomainBody+1))
	out.Bytes = n

	switch {
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		out.Error = fmt.Sprintf("bad status: %d", resp.StatusCode)
	case n < minBytes:
		out.Error = fmt.Sprintf("body too small: %d < %d", n, minBytes)
	}
	return out
}
