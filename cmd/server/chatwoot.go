package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ChatwootConfig struct {
	URL               string `json:"url"`
	AccountID         int    `json:"account_id"`
	AccountToken      string `json:"account_token,omitempty"`
	InboxID           int    `json:"inbox_id"`
	InboxIdentifier   string `json:"inbox_identifier"`
	MirrorAPI         bool   `json:"mirror_api"`
	ImportHistory     bool   `json:"import_history"`
	ImportHistoryDays int    `json:"import_history_days"`
}

func (c ChatwootConfig) valid() bool {
	return c.URL != "" && c.AccountID > 0 && c.InboxID > 0
}

func (s *server) handleGetChatwoot(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	cfg := sess.getChatwoot()
	cfg.AccountToken = ""
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":  cfg.valid(),
		"chatwoot": cfg,
	})
}

func (s *server) handleSetChatwoot(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	var in ChatwootConfig
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "payload inválido"})
		return
	}
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if in.AccountToken == "" {
		in.AccountToken = sess.getChatwoot().AccountToken
	}
	sess.setChatwoot(in)
	b, _ := json.Marshal(in)
	_ = sess.mgr.store.setChatwoot(r.Context(), sess.id, string(b))
	in.AccountToken = ""
	writeJSON(w, http.StatusOK, map[string]any{"chatwoot": in, "enabled": true})
}

func (s *server) handleDeleteChatwoot(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	sess.setChatwoot(ChatwootConfig{})
	_ = sess.mgr.store.setChatwoot(r.Context(), sess.id, "")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

