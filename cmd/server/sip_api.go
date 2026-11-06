package main

import (
	"encoding/json"
	"net/http"
)

type SIPExtConfig struct {
	AuthUser    string `json:"authUser,omitempty"`
	Enabled     bool   `json:"enabled"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Pass        string `json:"pass"`
	Dest        string `json:"dest"`
	Proxy       string `json:"proxy,omitempty"`
	InboundMode string `json:"inboundMode,omitempty"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
	Advertise   string `json:"advertise,omitempty"`
}

func (s *server) handleSIPExtConfig(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}

	if r.Method == http.MethodGet {
		cfg := sess.getSIPExt()
		if cfg.Port <= 0 {
			cfg.Port = 5060
		}
		writeJSON(w, http.StatusOK, cfg)
		return
	}

	var body SIPExtConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Port <= 0 {
		body.Port = 5060
	}
	sess.setSIPExt(body)
	_ = sess.mgr.store.setSIPExt(r.Context(), sess.id, body.Enabled, body.Host, body.Port, body.User, body.Pass, body.Dest, body.Proxy, body.InboundMode)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

