// Command qualitycheck verifies the two properties a working-config feed
// must have:
//
//  1. every served line is a valid share link (strict round-trip through the
//     same parser the daemon uses), and
//  2. optionally, every such link really carries traffic end-to-end through
//     xray-core (the same proxy test path used by the "comprehensive" depth).
//
// It exists because a feed can report configs that no client can use: an
// earlier viberayd appended " <latency>ms" to each URI, so /sub served
// undecodable vmess links while still counting them as working.
//
// Usage:
//
//	qualitycheck -sub http://127.0.0.1:8084/sub
//	qualitycheck -sub working.txt -e2e -xray-bin xray
//
// Exit status is non-zero when at least one served line carries an annotation
// (such as the old " <latency>ms" suffix), is otherwise malformed, or (with
// -e2e) fails to carry traffic.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/amirrezaalavi/Viberayd/internal/concurrency"
	"github.com/amirrezaalavi/Viberayd/internal/models"
	"github.com/amirrezaalavi/Viberayd/internal/tester"
	"github.com/amirrezaalavi/Viberayd/pkg/parser"
)

// latencySuffix matches the old corrupted format "<uri> 123ms".
var latencySuffix = regexp.MustCompile(`\s\d+ms$`)

type lineResult struct {
	Line      int    `json:"line"`
	URI       string `json:"uri"`
	RawOK     bool   `json:"raw_ok"`
	ParseErr  string `json:"parse_error,omitempty"`
	LatencyMs bool   `json:"latency_suffix,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	// End-to-end fields, only populated with -e2e.
	E2E      bool   `json:"e2e,omitempty"`
	E2EOK    bool   `json:"e2e_ok,omitempty"`
	E2EErr   string `json:"e2e_error,omitempty"`
	HTTPCode int    `json:"http_code,omitempty"`
	Latency  string `json:"e2e_latency,omitempty"`
}

func main() {
	sub := flag.String("sub", "", "subscription URL or local file to check")
	e2e := flag.Bool("e2e", false, "also run an end-to-end proxy test per config")
	xrayBin := flag.String("xray-bin", "xray", "path to the xray-core binary")
	portBase := flag.Int("port-base", 10940, "first local SOCKS port for -e2e")
	timeout := flag.Duration("timeout", 15*time.Second, "per-config timeout")
	parallel := flag.Int("parallel", 4, "concurrent -e2e tests")
	limit := flag.Int("limit", 0, "check at most N lines (0 = all)")
	asJSON := flag.Bool("json", false, "emit JSON instead of text")
	flag.Parse()

	if *sub == "" {
		fmt.Fprintln(os.Stderr, "qualitycheck: -sub is required")
		flag.Usage()
		os.Exit(2)
	}

	body, err := load(*sub, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "qualitycheck: load %s: %v\n", *sub, err)
		os.Exit(2)
	}

	lines := splitLines(body)
	if *limit > 0 && len(lines) > *limit {
		lines = lines[:*limit]
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "qualitycheck: no config lines found")
		os.Exit(2)
	}

	results := make([]lineResult, 0, len(lines))
	var cfgByLine = map[int]models.ProxyConfig{}
	for i, line := range lines {
		r := lineResult{Line: i + 1, URI: line}
		if latencySuffix.MatchString(line) {
			r.LatencyMs = true
		}
		cfg, err := parser.ParseSingle(line)
		switch {
		case err != nil:
			r.ParseErr = err.Error()
		case cfg.Raw != line:
			r.ParseErr = "round-trip changed the link"
		default:
			r.RawOK = true
			r.Protocol = string(cfg.Protocol())
			cfgByLine[i] = cfg
		}
		results = append(results, r)
	}

	if *e2e {
		runE2E(results, cfgByLine, *xrayBin, *portBase, *timeout, *parallel)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(results)
	} else {
		report(results, *e2e)
	}

	malformed, e2eFailed := summarize(results)
	corrupt := countLatency(results)
	if malformed > 0 || corrupt > 0 || (*e2e && e2eFailed > 0) {
		os.Exit(1)
	}
}

// load reads the subscription from an http(s) URL or a local file, then
// transparently base64-decodes a whole-body encoded subscription.
func load(sub string, timeout time.Duration) (string, error) {
	var raw string
	if strings.HasPrefix(sub, "http://") || strings.HasPrefix(sub, "https://") {
		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(sub)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("http %d", resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		raw = string(b)
	} else {
		b, err := os.ReadFile(sub)
		if err != nil {
			return "", err
		}
		raw = string(b)
	}

	trimmed := strings.TrimSpace(raw)
	if parser.IsBase64Encoded(trimmed) {
		cleaned := strings.NewReplacer("\n", "", "\r", "", " ", "", "\t", "").Replace(trimmed)
		if decoded, err := parser.DecodeBase64(cleaned); err == nil {
			return string(decoded), nil
		}
	}
	return raw, nil
}

func splitLines(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	var out []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, l)
	}
	return out
}

// runE2E starts one xray instance per parsed config on its own local SOCKS
// port and issues a real HTTP request through it, exactly like the daemon's
// comprehensive depth.
func runE2E(results []lineResult, cfgByLine map[int]models.ProxyConfig, bin string, portBase int, timeout time.Duration, parallel int) {
	if parallel < 1 {
		parallel = 1
	}
	pm := concurrency.NewPortManager(portBase, parallel+1)
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	for i := range results {
		cfg, ok := cfgByLine[i]
		if !ok {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, cfg models.ProxyConfig) {
			defer wg.Done()
			defer func() { <-sem }()

			port, err := pm.Allocate()
			if err != nil {
				results[idx].E2E = true
				results[idx].E2EErr = fmt.Sprintf("allocate port: %v", err)
				return
			}
			defer pm.Release(port)

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			runner := tester.NewXrayRunner(bin)
			runner.WorkDir = os.TempDir()
			res := runner.TestXrayProxy(ctx, cfg, port, timeout)

			results[idx].E2E = true
			results[idx].E2EOK = res.Success
			results[idx].HTTPCode = res.HTTPCode
			results[idx].Latency = res.Latency.Round(time.Millisecond).String()
			if !res.Success {
				results[idx].E2EErr = res.Error
			}
		}(i, cfg)
	}
	wg.Wait()
}

func report(results []lineResult, e2e bool) {
	malformed, e2eFailed := summarize(results)
	for _, r := range results {
		status := "ok"
		switch {
		case !r.RawOK:
			status = "MALFORMED"
		case r.LatencyMs:
			status = "CORRUPT(latency-suffix)"
		}
		if e2e && r.E2E {
			if r.E2EOK {
				status += " e2e=ok"
				if r.HTTPCode != 0 {
					status += fmt.Sprintf(" http=%d", r.HTTPCode)
				}
				if r.Latency != "" {
					status += " " + r.Latency
				}
			} else {
				status += " e2e=FAIL(" + r.E2EErr + ")"
			}
		}
		fmt.Printf("%4d %-22s %s\n", r.Line, status, r.URI)
		if r.ParseErr != "" {
			fmt.Printf("     parse error: %s\n", r.ParseErr)
		}
	}

	fmt.Printf("\nchecked=%d strict_parse=%d malformed=%d latency_suffix=%d",
		len(results), len(results)-malformed, malformed, countLatency(results))
	if e2e {
		tested, passed := countE2E(results)
		fmt.Printf(" e2e_tested=%d e2e_passed=%d e2e_failed=%d", tested, passed, e2eFailed)
	}
	fmt.Println()
}

func summarize(results []lineResult) (malformed, e2eFailed int) {
	for _, r := range results {
		if !r.RawOK {
			malformed++
		}
		if r.E2E && !r.E2EOK {
			e2eFailed++
		}
	}
	return malformed, e2eFailed
}

func countLatency(results []lineResult) int {
	n := 0
	for _, r := range results {
		if r.LatencyMs {
			n++
		}
	}
	return n
}

func countE2E(results []lineResult) (tested, passed int) {
	for _, r := range results {
		if !r.E2E {
			continue
		}
		tested++
		if r.E2EOK {
			passed++
		}
	}
	return tested, passed
}
