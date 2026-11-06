package main

import (
	"net/http"
)

type NativeFormInfo struct {
	FlowID        string `json:"flowId"`
	FlowName      string `json:"flowName,omitempty"`
	FormType      string `json:"formType,omitempty"`
	Body          string `json:"body,omitempty"`
	CTA           string `json:"cta,omitempty"`
	Screen        string `json:"screen,omitempty"`
	SentCount     int    `json:"sentCount"`
	LastSentAt    int64  `json:"lastSentAt,omitempty"`
	ResponseCount int    `json:"responseCount"`
}

func (s *server) handleListNativeForms(w http.ResponseWriter, r *http.Request) {
	sess := s.sessionByID(w, r.PathValue("sid"))
	if sess == nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"forms": []NativeFormInfo{},
		"count": 0,
	})
}

