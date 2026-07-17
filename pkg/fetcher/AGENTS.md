# Fetcher Package — Agent Guide

This package downloads subscription content and lists of proxy configurations from remote URLs.

## Purpose

Instead of downloading lists manually first, users can pass standard remote subscription URLs directly as command inputs (e.g. `https://example.com/sub.txt`). The fetcher retrieves the payload body before sending it down to the parsing engine.

---

## Files

- `fetcher.go`: Standard HTTP client helper function `Fetch()`.
- `fetcher_test.go`: Unit tests mocking HTTP servers to verify correct retrieval, timeout limits, and non-200 HTTP status code responses.

---

## Public API

### `Fetch(url string, timeout time.Duration) (string, error)`

Downloads the body of a resource from a remote URL.
- **HTTP Client Bounds:** Employs a custom `&http.Client` with a strict timeout limit (defaults to `30s` if `0` is passed).
- **Status Checks:** Returns an error if the server answers with any HTTP status code other than `200 OK`.
- **Resource Cleanup:** Correctly closes the response stream (`defer resp.Body.Close()`) to prevent socket leaks.

---

## Conventions & Gotchas

- **Input Sanitization:** The fetcher assumes the input URL is fully-qualified (starts with `http://` or `https://`). It is checked and routed inside `cmd/viberay/main.go` prior to invocation.
- **DNS/Network Dependencies:** If the system is running in an environment without internet access or with strict proxy firewalls, subscription fetch calls may time out or fail.
