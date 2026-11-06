package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// validateProxy checa o formato (vazio = sem proxy).
func validateProxy(addr string) error {
	if addr == "" {
		return nil
	}
	u, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("URL de proxy inválida: %w", err)
	}
	switch u.Scheme {
	case "http", "https", "socks5":
		if u.Host == "" {
			return fmt.Errorf("host do proxy ausente")
		}
		return nil
	default:
		return fmt.Errorf("scheme %q não suportado (use http, https ou socks5)", u.Scheme)
	}
}

// GET /api/sessions/{sid}/proxy → { proxy, enabled }
func (s *server) handleGetProxy(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	p := sess.getProxy()
	writeJSON(w, http.StatusOK, map[string]any{"proxy": p, "enabled": p != ""})
}

// PUT /api/sessions/{sid}/proxy { "proxy": "socks5://user:pass@host:1080" }
func (s *server) handleSetProxy(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	var b struct {
		Proxy string `json:"proxy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload inválido"})
		return
	}
	addr := strings.TrimSpace(b.Proxy)
	if err := validateProxy(addr); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sess.setProxy(addr)
	if err := sess.mgr.store.setProxy(r.Context(), sess.id, addr); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	go sess.reconnect()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

