package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

const mcpInstructions = `ArendCalls: WhatsApp (mensagens, ligações de voz e vídeo, grupos, comunidades e contatos).
- Telefones sempre com DDI + DDD, só dígitos (ex.: 5561999998888). Grupos usam o ID terminado em @g.us.
- Se houver mais de uma sessão (número de WhatsApp), chame list_sessions e passe "session" (ID, nome ou telefone). Com uma sessão só, pode omitir.
- api_request cobre qualquer rota da API REST do ArendCalls.`

func (s *server) handleMCP(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		key, _ = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if key == "" {
		key = r.Header.Get("X-Api-Key")
	}
	if key == "" {
		key = r.URL.Query().Get("key")
	}
	key = strings.TrimSpace(key)

	var scopedSessionID string
	var readOnly bool

	if s.apiKey != "" && key == s.apiKey {
		// Global API key
	} else if s.panelCookieOK(r) {
		// Panel cookie
	} else if strings.HasPrefix(key, "acmcp_") && s.sessions != nil && s.sessions.store != nil {
		h := sha256.Sum256([]byte(key))
		hash := hex.EncodeToString(h[:])
		var sid string
		var ro int
		err := s.sessions.store.db.QueryRowContext(r.Context(),
			`SELECT session_id, read_only FROM mcp_keys WHERE hash = ? LIMIT 1`, hash).Scan(&sid, &ro)
		if err != nil || sid == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="arendcalls-mcp"`)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "chave MCP ausente ou inválida"})
			return
		}
		_, _ = s.sessions.store.db.ExecContext(r.Context(), `UPDATE mcp_keys SET last_used_at = CURRENT_TIMESTAMP WHERE hash = ?`, hash)
		scopedSessionID = sid
		readOnly = ro != 0
	} else {
		w.Header().Set("WWW-Authenticate", `Bearer realm="arendcalls-mcp"`)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "chave MCP ausente ou inválida"})
		return
	}

	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{
			"name":            "arendcalls",
			"title":           "ArendCalls",
			"protocolVersion": "2024-11-05",
			"status":          "ok",
		})
		return
	}

	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      any             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"jsonrpc": "2.0",
			"id":      nil,
			"error":   map[string]any{"code": -32700, "message": "parse error"},
		})
		return
	}

	reply := func(res any) {
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  res,
		})
	}
	replyErr := func(code int, msg string) {
		writeJSON(w, http.StatusOK, map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error":   map[string]any{"code": code, "message": msg},
		})
	}

	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		pv := p.ProtocolVersion
		if pv == "" {
			pv = "2024-11-05"
		}
		reply(map[string]any{
			"protocolVersion": pv,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "arendcalls", "title": "ArendCalls", "version": "1.0.0"},
			"instructions":    mcpInstructions,
		})
	case "notifications/initialized", "ping":
		reply(map[string]any{})
	case "tools/list":
		tools := []map[string]any{
			{
				"name":        "list_sessions",
				"description": "Lista as sessões de WhatsApp cadastradas no ArendCalls com ID, nome, número (JID) e estado de conexão.",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
			},
			{
				"name":        "start_call",
				"description": "Inicia uma ligação de voz do WhatsApp no ArendCalls para o número de destino.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"session": map[string]any{"type": "string", "description": "ID da sessão (opcional se a chave já for escopada)"},
						"to":      map[string]any{"type": "string", "description": "Telefone com DDI+DDD (ex.: 5511999998888)"},
					},
					"required": []string{"to"},
				},
			},
		}
		reply(map[string]any{"tools": tools})
	case "tools/call":
		var call struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &call); err != nil {
			replyErr(-32602, "invalid params")
			return
		}
		switch call.Name {
		case "list_sessions":
			infos := s.sessions.infos()
			if scopedSessionID != "" {
				var filtered []SessionInfo
				for _, inf := range infos {
					if inf.ID == scopedSessionID {
						filtered = append(filtered, inf)
					}
				}
				infos = filtered
			}
			b, _ := json.MarshalIndent(infos, "", "  ")
			reply(map[string]any{
				"content": []map[string]any{{"type": "text", "text": string(b)}},
			})
		case "start_call":
			if readOnly {
				replyErr(-32600, "chave MCP somente leitura")
				return
			}
			sid, _ := call.Arguments["session"].(string)
			if scopedSessionID != "" {
				sid = scopedSessionID
			}
			if sid == "" {
				infos := s.sessions.infos()
				if len(infos) > 0 {
					sid = infos[0].ID
				}
			}
			to, _ := call.Arguments["to"].(string)
			bodyBytes, _ := json.Marshal(map[string]string{"to": to})
			subReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/sessions/%s/calls", sid), bytes.NewReader(bodyBytes))
			subReq = subReq.WithContext(context.Background())
			rec := httptest.NewRecorder()
			s.handleStartCall(rec, subReq)
			reply(map[string]any{
				"content": []map[string]any{{"type": "text", "text": rec.Body.String()}},
				"isError": rec.Code >= 400,
			})
		default:
			replyErr(-32601, "unknown tool: "+call.Name)
		}
	default:
		replyErr(-32601, "method not found: "+req.Method)
	}
}
