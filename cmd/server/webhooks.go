package main

import (
	"encoding/json"
	"net/http"
)

type WebhookTarget struct {
	ID           string   `json:"id"`
	Name         string   `json:"name,omitempty"`
	URL          string   `json:"url"`
	Secret       string   `json:"secret,omitempty"`
	Events       []string `json:"events,omitempty"`
	IgnoreGroups bool     `json:"ignoreGroups,omitempty"`
}

func (t WebhookTarget) view() map[string]any {
	ev := t.Events
	if ev == nil {
		ev = []string{}
	}
	return map[string]any{
		"id":           t.ID,
		"name":         t.Name,
		"url":          t.URL,
		"hasSecret":    t.Secret != "",
		"events":       ev,
		"ignoreGroups": t.IgnoreGroups,
		"status": map[string]any{
			"state":               "closed",
			"consecutiveFailures": 0,
		},
	}
}

func (s *server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	hooks := sess.getWebhooks()
	viewList := make([]map[string]any, 0, len(hooks))
	for _, h := range hooks {
		viewList = append(viewList, h.view())
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhooks": viewList, "max": 10})
}

func (s *server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	var in struct {
		Name         string   `json:"name"`
		URL          string   `json:"url"`
		Secret       string   `json:"secret"`
		Events       []string `json:"events"`
		IgnoreGroups bool     `json:"ignoreGroups"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url obrigatória"})
		return
	}
	h := WebhookTarget{
		ID:           newSessionID(),
		Name:         in.Name,
		URL:          in.URL,
		Secret:       in.Secret,
		Events:       in.Events,
		IgnoreGroups: in.IgnoreGroups,
	}
	hooks := append(sess.getWebhooks(), h)
	sess.setWebhooks(hooks)
	b, _ := json.Marshal(hooks)
	_ = sess.mgr.store.setWebhooks(r.Context(), sess.id, string(b))
	writeJSON(w, http.StatusOK, h.view())
}

func (s *server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	var in struct {
		Name         string   `json:"name"`
		URL          string   `json:"url"`
		Secret       *string  `json:"secret"`
		Events       []string `json:"events"`
		IgnoreGroups bool     `json:"ignoreGroups"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url obrigatória"})
		return
	}
	hooks := sess.getWebhooks()
	var updated *WebhookTarget
	for i := range hooks {
		if hooks[i].ID == id {
			hooks[i].Name = in.Name
			hooks[i].URL = in.URL
			hooks[i].Events = in.Events
			hooks[i].IgnoreGroups = in.IgnoreGroups
			if in.Secret != nil {
				hooks[i].Secret = *in.Secret
			}
			updated = &hooks[i]
			break
		}
	}
	if updated == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "webhook não encontrado"})
		return
	}
	sess.setWebhooks(hooks)
	b, _ := json.Marshal(hooks)
	_ = sess.mgr.store.setWebhooks(r.Context(), sess.id, string(b))
	writeJSON(w, http.StatusOK, updated.view())
}

func (s *server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	id := r.PathValue("id")
	hooks := sess.getWebhooks()
	filtered := make([]WebhookTarget, 0, len(hooks))
	for _, h := range hooks {
		if h.ID != id {
			filtered = append(filtered, h)
		}
	}
	sess.setWebhooks(filtered)
	b, _ := json.Marshal(filtered)
	_ = sess.mgr.store.setWebhooks(r.Context(), sess.id, string(b))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": 200, "message": "teste de webhook do ArendCalls"})
}

func (s *server) handleReenableWebhook(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

