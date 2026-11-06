package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type McpKey struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	ReadOnly  bool      `json:"readOnly"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *server) handleListMcpKeys(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	keys, err := sess.mgr.store.listMCPKeys(r.Context(), sess.id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	endpoint := "/api/mcp"
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys, "max": 20, "endpoint": endpoint})
}

func (s *server) handleCreateMcpKey(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	var in struct {
		Name     string `json:"name"`
		ReadOnly bool   `json:"readOnly"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "Chave MCP"
	}
	keyID := newSessionID()
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	secret := "acmcp_" + hex.EncodeToString(b)
	prefix := secret[:12] + "…"
	h := sha256.Sum256([]byte(secret))
	hash := hex.EncodeToString(h[:])

	k := McpKey{
		ID:        keyID,
		SessionID: sess.id,
		Name:      name,
		Prefix:    prefix,
		ReadOnly:  in.ReadOnly,
		CreatedAt: time.Now(),
	}
	if err := sess.mgr.store.insertMCPKey(r.Context(), k, hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	baseURL := "http://" + r.Host
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		baseURL = "https://" + r.Host
	}
	endpoint := baseURL + "/api/mcp"
	fullURL := endpoint + "?key=" + secret
	writeJSON(w, http.StatusOK, map[string]any{
		"key":      k,
		"secret":   secret,
		"endpoint": endpoint,
		"url":      fullURL,
	})
}

func (s *server) handleDeleteMcpKey(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	_ = sess.mgr.store.deleteMCPKey(r.Context(), sess.id, id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

