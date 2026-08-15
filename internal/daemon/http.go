package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type httpServer struct {
	daemon  *Daemon
	subSrv  *http.Server
	apiSrv  *http.Server
}

func (d *Daemon) StartHTTPServers() error {
	hs := &httpServer{daemon: d}
	d.http = hs

	mux := http.NewServeMux()
	mux.HandleFunc(d.Config.HTTP.SubPath, hs.handleSub)

	subAddr := fmt.Sprintf(":%d", d.Config.HTTP.Port)
	hs.subSrv = &http.Server{
		Addr:    subAddr,
		Handler: mux,
	}

	go func() {
		slog.Info("subscription HTTP server", "addr", subAddr, "path", d.Config.HTTP.SubPath)
		if err := hs.subSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("sub server", "error", err)
		}
	}()

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/health", hs.handleHealth)
	apiMux.HandleFunc("/api/urls", hs.handleURLs)
	apiMux.HandleFunc("/api/urls/", hs.handleURLsByID)
	apiMux.HandleFunc("/api/stats", hs.handleStats)
	apiMux.HandleFunc("/api/cycle/trigger", hs.handleCycleTrigger)
	apiMux.HandleFunc("/api/configs", hs.handleConfigs)
	apiMux.HandleFunc("/metrics", hs.handleMetrics)

	apiAddr := fmt.Sprintf(":%d", d.Config.HTTP.APIPort)
	hs.apiSrv = &http.Server{
		Addr:    apiAddr,
		Handler: apiMux,
	}

	go func() {
		slog.Info("management API server", "addr", apiAddr)
		if err := hs.apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api server", "error", err)
		}
	}()

	return nil
}

func (d *Daemon) StopHTTPServers(ctx context.Context) error {
	var errs []string

	if hs := d.http; hs != nil {
		if err := hs.subSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("sub: %v", err))
		}
		if err := hs.apiSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("api: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (hs *httpServer) handleSub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := os.ReadFile(hs.daemon.Config.Daemon.OutputFile)
	if err != nil {
		http.Error(w, "no data", http.StatusNotFound)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(encoded))
}

func (hs *httpServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (hs *httpServer) handleURLs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		hs.listURLs(w, r)
	case http.MethodPost:
		hs.addURL(w, r)
	case http.MethodPut:
		hs.replaceURLs(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// validSubscriptionURL reports whether a string is a usable http(s)
// subscription URL. Used by addURL and replaceURLs so garbage never enters
// the urls file (error correction at input time; the fetch loop tolerates
// unreachable URLs at runtime, but malformed entries are rejected here).
func validSubscriptionURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (hs *httpServer) listURLs(w http.ResponseWriter, r *http.Request) {
	urls, err := LoadURLs(hs.daemon.Config.Daemon.URLsFile)
	if err != nil {
		jsonError(w, "read urls: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(urls)
}

func (hs *httpServer) addURL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.URL == "" {
		jsonError(w, "url is required", http.StatusBadRequest)
		return
	}
	if !validSubscriptionURL(body.URL) {
		jsonError(w, "invalid url (must be http or https)", http.StatusBadRequest)
		return
	}

	f, err := os.OpenFile(hs.daemon.Config.Daemon.URLsFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		jsonError(w, "open urls file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n", body.URL); err != nil {
		jsonError(w, "write url: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added", "url": body.URL})
}

// replaceURLs atomically replaces the whole urls file with the given list
// (PUT /api/urls with {"urls": [...]}). Every non-empty line must be a valid
// http(s) URL; if any line is invalid the whole request is rejected (400 with
// the offending entries) so a bad batch can never leave the file half-written
// or silently drop URLs the operator thought were saved.
func (hs *httpServer) replaceURLs(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid json", http.StatusBadRequest)
		return
	}

	var valid, invalid []string
	for _, raw := range body.URLs {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !validSubscriptionURL(line) {
			invalid = append(invalid, line)
			continue
		}
		valid = append(valid, line)
	}
	if len(invalid) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "invalid url(s)",
			"invalid": invalid,
		})
		return
	}

	data := strings.Join(valid, "\n")
	if len(valid) > 0 {
		data += "\n"
	}
	if err := os.WriteFile(hs.daemon.Config.Daemon.URLsFile, []byte(data), 0644); err != nil {
		jsonError(w, "write urls: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "replaced", "urls": valid})
}

func (hs *httpServer) handleURLsByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/urls/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "invalid id (line number required)", http.StatusBadRequest)
		return
	}

	urls, err := LoadURLs(hs.daemon.Config.Daemon.URLsFile)
	if err != nil {
		jsonError(w, "read urls: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if id < 1 || id > len(urls) {
		jsonError(w, fmt.Sprintf("id %d out of range (1-%d)", id, len(urls)), http.StatusNotFound)
		return
	}

	removed := urls[id-1]
	urls = append(urls[:id-1], urls[id:]...)

	data := strings.Join(urls, "\n")
	if len(urls) > 0 {
		data += "\n"
	}

	if err := os.WriteFile(hs.daemon.Config.Daemon.URLsFile, []byte(data), 0644); err != nil {
		jsonError(w, "write urls: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "removed", "url": removed})
}

func (hs *httpServer) handleStats(w http.ResponseWriter, r *http.Request) {
	hs.daemon.StateMu.RLock()
	total := len(hs.daemon.State.Configs)
	working := countByState(hs.daemon.State, StateWorking)
	failed := countByState(hs.daemon.State, StateFailed)
	unreachable := countByState(hs.daemon.State, StateUnreachable)
	updatedAt := hs.daemon.State.UpdatedAt
	hs.daemon.StateMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":       total,
		"working":     working,
		"failed":      failed,
		"unreachable": unreachable,
		"updated_at":  updatedAt.Format(time.RFC3339),
	})
}

func (hs *httpServer) handleCycleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hs.daemon.Trigger()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "triggered"})
}

func (hs *httpServer) handleConfigs(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(perPageStr)
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	hs.daemon.StateMu.RLock()
	entries := make([]*ConfigEntry, 0, len(hs.daemon.State.Configs))
	for _, entry := range hs.daemon.State.Configs {
		entries = append(entries, entry)
	}
	total := len(entries)
	hs.daemon.StateMu.RUnlock()

	start := (page - 1) * perPage
	if start >= total {
		entries = nil
	} else {
		end := start + perPage
		if end > total {
			end = total
		}
		entries = entries[start:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"page":     page,
		"per_page": perPage,
		"total":    total,
		"configs":  entries,
	})
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
