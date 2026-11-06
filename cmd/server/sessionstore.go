package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
)

type sessionRow struct {
	ID            string
	Name          string
	JID           string
	WebhookURL    string
	PanelInbound  bool
	Proxy         string
	Chatwoot      string
	Webhooks      string
	SIPUser       string
	SIPPass       string
	SIPExtEnabled bool
	SIPExtHost    string
	SIPExtPort    int
	SIPExtUser    string
	SIPExtPass    string
	SIPExtDest    string
	SIPExtProxy   string
	SIPExtOpts    string
}

type sessionStore struct{ db *sql.DB }

func newSessionStore(ctx context.Context, db *sql.DB) (*sessionStore, error) {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sessions (
		id   TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		jid  TEXT,
		webhook_url TEXT,
		panel_inbound INTEGER DEFAULT 0
	)`)
	if err != nil {
		return nil, err
	}
	// Add columns if they don't exist (for existing DBs).
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN webhook_url TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN panel_inbound INTEGER DEFAULT 0`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN proxy TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN chatwoot TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN webhooks TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_user TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_pass TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_enabled INTEGER DEFAULT 0`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_host TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_port INTEGER DEFAULT 5060`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_user TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_pass TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_dest TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_proxy TEXT`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN sip_ext_opts TEXT`)

	_, _ = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS mcp_keys (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		name TEXT NOT NULL,
		hash TEXT NOT NULL,
		prefix TEXT NOT NULL,
		read_only INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_used_at TIMESTAMP
	)`)

	// Global switch: does the panel show incoming calls at all. Default on.
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS panel_settings (
		id            INTEGER PRIMARY KEY CHECK (id = 1),
		inbound_calls INTEGER DEFAULT 1
	)`)
	if err != nil {
		return nil, err
	}
	return &sessionStore{db: db}, nil
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// panelInboundCalls reports the global switch (default true when never set).
func (s *sessionStore) panelInboundCalls(ctx context.Context) (bool, error) {
	var v int
	err := s.db.QueryRowContext(ctx, `SELECT inbound_calls FROM panel_settings WHERE id=1`).Scan(&v)
	if err == sql.ErrNoRows {
		return true, nil
	}
	return v != 0, err
}

func (s *sessionStore) setPanelInboundCalls(ctx context.Context, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO panel_settings (id, inbound_calls) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET inbound_calls=excluded.inbound_calls`, v)
	return err
}

func (s *sessionStore) list(ctx context.Context) ([]sessionRow, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, COALESCE(jid, ''), COALESCE(webhook_url, ''), COALESCE(panel_inbound, 0),
		COALESCE(proxy, ''), COALESCE(chatwoot, ''), COALESCE(webhooks, ''),
		COALESCE(sip_user, ''), COALESCE(sip_pass, ''), COALESCE(sip_ext_enabled, 0),
		COALESCE(sip_ext_host, ''), COALESCE(sip_ext_port, 5060), COALESCE(sip_ext_user, ''),
		COALESCE(sip_ext_pass, ''), COALESCE(sip_ext_dest, ''), COALESCE(sip_ext_proxy, ''), COALESCE(sip_ext_opts, '')
		FROM sessions ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sessionRow
	for rows.Next() {
		var r sessionRow
		var panelInbound, sipExtEnabled int
		if err := rows.Scan(&r.ID, &r.Name, &r.JID, &r.WebhookURL, &panelInbound,
			&r.Proxy, &r.Chatwoot, &r.Webhooks,
			&r.SIPUser, &r.SIPPass, &sipExtEnabled,
			&r.SIPExtHost, &r.SIPExtPort, &r.SIPExtUser,
			&r.SIPExtPass, &r.SIPExtDest, &r.SIPExtProxy, &r.SIPExtOpts); err != nil {
			return nil, err
		}
		r.PanelInbound = panelInbound != 0
		r.SIPExtEnabled = sipExtEnabled != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *sessionStore) setProxy(ctx context.Context, id, proxy string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET proxy = ? WHERE id = ?`, proxy, id)
	return err
}

func (s *sessionStore) setChatwoot(ctx context.Context, id, chatwoot string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET chatwoot = ? WHERE id = ?`, chatwoot, id)
	return err
}

func (s *sessionStore) setWebhooks(ctx context.Context, id, webhooks string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET webhooks = ? WHERE id = ?`, webhooks, id)
	return err
}

func (s *sessionStore) setSIPExt(ctx context.Context, id string, enabled bool, host string, port int, user, pass, dest, proxy, opts string) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET 
		sip_ext_enabled = ?, sip_ext_host = ?, sip_ext_port = ?, sip_ext_user = ?, 
		sip_ext_pass = ?, sip_ext_dest = ?, sip_ext_proxy = ?, sip_ext_opts = ? 
		WHERE id = ?`,
		en, host, port, user, pass, dest, proxy, opts, id)
	return err
}

func (s *sessionStore) listMCPKeys(ctx context.Context, sid string) ([]McpKey, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, name, prefix, read_only, created_at FROM mcp_keys WHERE session_id = ? ORDER BY created_at DESC`, sid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []McpKey
	for rows.Next() {
		var k McpKey
		var ro int
		if err := rows.Scan(&k.ID, &k.SessionID, &k.Name, &k.Prefix, &ro, &k.CreatedAt); err != nil {
			return nil, err
		}
		k.ReadOnly = ro != 0
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *sessionStore) insertMCPKey(ctx context.Context, k McpKey, hash string) error {
	ro := 0
	if k.ReadOnly {
		ro = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO mcp_keys (id, session_id, name, hash, prefix, read_only, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		k.ID, k.SessionID, k.Name, hash, k.Prefix, ro, k.CreatedAt)
	return err
}

func (s *sessionStore) deleteMCPKey(ctx context.Context, sid, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_keys WHERE session_id = ? AND id = ?`, sid, id)
	return err
}

func (s *sessionStore) insert(ctx context.Context, id, name string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (id, name, jid) VALUES (?, ?, NULL)`, id, name)
	return err
}

func (s *sessionStore) setJID(ctx context.Context, id, jid string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET jid = ? WHERE id = ?`, jid, id)
	return err
}

func (s *sessionStore) delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *sessionStore) updateName(ctx context.Context, id, name string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET name = ? WHERE id = ?`, name, id)
	return err
}

func (s *sessionStore) setWebhookURL(ctx context.Context, id, webhookURL string) error {
	var err error
	if webhookURL == "" {
		_, err = s.db.ExecContext(ctx, `UPDATE sessions SET webhook_url = NULL WHERE id = ?`, id)
	} else {
		_, err = s.db.ExecContext(ctx, `UPDATE sessions SET webhook_url = ? WHERE id = ?`, webhookURL, id)
	}
	return err
}

func (s *sessionStore) setPanelInbound(ctx context.Context, id string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET panel_inbound = ? WHERE id = ?`, v, id)
	return err
}
