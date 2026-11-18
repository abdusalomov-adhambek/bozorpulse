package infrastructure

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type ConversationStore struct{ db *sql.DB }

var _ application.ConversationStore = (*ConversationStore)(nil)

func NewConversationStore(db *sql.DB) *ConversationStore { return &ConversationStore{db: db} }

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf)
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func (s *ConversationStore) FindOrCreate(ctx context.Context, p auth.Principal, conversationID, locale string) (string, error) {
	if conversationID != "" {
		var id string
		err := s.db.QueryRowContext(ctx, `SELECT id FROM ai_conversations WHERE id=$1 AND user_id=$2`, conversationID, p.ID).Scan(&id)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
	}
	if locale == "" {
		locale = "en"
	}
	id := newID()
	now := nowString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_conversations
		(id,user_id,company_id,locale,intent,data_status,summary,created_at,updated_at)
		VALUES ($1,$2,NULLIF($3,''),$4,'','','{}'::jsonb,$5,$5)`,
		id, p.ID, p.CompanyID, locale, now)
	return id, err
}

func (s *ConversationStore) Load(ctx context.Context, p auth.Principal, conversationID string) ([]domain.ChatMessage, domain.ConversationSummary, error) {
	var summaryRaw []byte
	err := s.db.QueryRowContext(ctx, `SELECT summary FROM ai_conversations WHERE id=$1 AND user_id=$2`, conversationID, p.ID).Scan(&summaryRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ConversationSummary{}, shared.ErrNotFound
		}
		return nil, domain.ConversationSummary{}, err
	}
	var summary domain.ConversationSummary
	if len(summaryRaw) > 0 {
		_ = json.Unmarshal(summaryRaw, &summary)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT role,content FROM ai_messages WHERE conversation_id=$1 ORDER BY created_at, id`, conversationID)
	if err != nil {
		return nil, summary, err
	}
	defer rows.Close()
	messages := make([]domain.ChatMessage, 0)
	for rows.Next() {
		var m domain.ChatMessage
		if err := rows.Scan(&m.Role, &m.Content); err != nil {
			return nil, summary, err
		}
		messages = append(messages, m)
	}
	return messages, summary, rows.Err()
}

func (s *ConversationStore) Append(ctx context.Context, p auth.Principal, conversationID string, messages []domain.ChatMessage, summary domain.ConversationSummary, intent domain.Intent, status domain.DataStatus, toolCalls []domain.ToolCall, sources []domain.Source) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	summaryRaw, _ := json.Marshal(summary)
	toolRaw, _ := json.Marshal(toolCalls)
	sourcesRaw, _ := json.Marshal(sources)
	now := nowString()
	for _, m := range messages {
		_, err = tx.ExecContext(ctx, `INSERT INTO ai_messages
			(id,conversation_id,role,content,tool_calls,sources,intent,data_status,created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			newID(), conversationID, m.Role, m.Content, toolRaw, sourcesRaw, string(intent), string(status), now)
		if err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE ai_conversations
		SET summary=$1, intent=$2, data_status=$3, updated_at=$4
		WHERE id=$5 AND user_id=$6`, summaryRaw, string(intent), string(status), now, conversationID, p.ID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *ConversationStore) List(ctx context.Context, p auth.Principal) ([]application.ConversationInfo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.locale,c.intent,c.data_status,
		COALESCE((SELECT m.content FROM ai_messages m WHERE m.conversation_id=c.id AND m.role='user'
		          ORDER BY m.created_at DESC, m.id DESC LIMIT 1),''),
		c.created_at,c.updated_at
		FROM ai_conversations c WHERE c.user_id=$1 ORDER BY c.updated_at DESC LIMIT 50`, p.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]application.ConversationInfo, 0)
	for rows.Next() {
		var c application.ConversationInfo
		if err := rows.Scan(&c.ID, &c.Locale, &c.Intent, &c.DataStatus, &c.Preview, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *ConversationStore) Get(ctx context.Context, p auth.Principal, conversationID string) (application.ConversationDetail, error) {
	var c application.ConversationDetail
	err := s.db.QueryRowContext(ctx, `SELECT id,locale,intent,data_status,created_at,updated_at
		FROM ai_conversations WHERE id=$1 AND user_id=$2`, conversationID, p.ID).
		Scan(&c.ID, &c.Locale, &c.Intent, &c.DataStatus, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c, shared.ErrNotFound
		}
		return c, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT role, content,
		COALESCE(tool_calls,'[]'::jsonb), COALESCE(sources,'[]'::jsonb),
		COALESCE(data_status,''), COALESCE(intent,'')
		FROM ai_messages WHERE conversation_id=$1 ORDER BY created_at, id`, conversationID)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	c.Messages = make([]domain.ChatMessage, 0)
	for rows.Next() {
		var m domain.ChatMessage
		var callsRaw, sourcesRaw []byte
		if err := rows.Scan(&m.Role, &m.Content, &callsRaw, &sourcesRaw, &m.DataStatus, &m.Intent); err != nil {
			return c, err
		}
		_ = json.Unmarshal(callsRaw, &m.ToolCalls)
		_ = json.Unmarshal(sourcesRaw, &m.Sources)
		c.Messages = append(c.Messages, m)
	}
	return c, rows.Err()
}

func (s *ConversationStore) Delete(ctx context.Context, p auth.Principal, conversationID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM ai_conversations WHERE id=$1 AND user_id=$2`, conversationID, p.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return shared.ErrNotFound
	}
	return nil
}
