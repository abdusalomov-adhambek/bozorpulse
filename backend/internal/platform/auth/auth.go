// Package auth provides password hashing, token sessions, and role/status guards.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

const (
	RoleAdmin    = "admin"
	RoleSupplier = "supplier"
	RoleCustomer = "customer"

	StatusWaiting   = "waiting"
	StatusConfirmed = "confirmed"
	StatusRejected  = "rejected"
)

// Principal is the authenticated user attached to a request context.
type Principal struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CompanyID string `json:"company_id"`
}

type ctxKey struct{}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// HashPassword returns salt:hash using SHA-256 with a random salt.
// Production should replace this with bcrypt/argon2.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(sum[:]), nil
}

func CheckPassword(stored, password string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		return false
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sum := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(sum[:]) == parts[1]
}

// Sessions stores opaque bearer tokens in PostgreSQL.
type Sessions struct{ db *sql.DB }

func NewSessions(db *sql.DB) *Sessions { return &Sessions{db: db} }

func (s *Sessions) Create(ctx context.Context, userID string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1,$2,$3)`,
		token, userID, time.Now().Add(7*24*time.Hour))
	return token, err
}

func (s *Sessions) Principal(ctx context.Context, token string) (Principal, error) {
	var p Principal
	err := s.db.QueryRowContext(ctx, `SELECT u.id,u.username,u.role,u.status,COALESCE(c.id,'')
		FROM sessions s JOIN users u ON u.id=s.user_id
		LEFT JOIN companies c ON c.owner_user_id=u.id
		WHERE s.token=$1 AND s.expires_at > now()`, token).
		Scan(&p.ID, &p.Username, &p.Role, &p.Status, &p.CompanyID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, shared.ErrUnauthorized
	}
	return p, err
}

func (s *Sessions) Delete(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	return err
}

// Middleware loads the principal from the Bearer token when present.
func (s *Sessions) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if p, err := s.Principal(r.Context(), token); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, p))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Chain applies middleware in order (first is outermost).
func Chain(mws ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			next = mws[i](next)
		}
		return next
	}
}

// Require blocks unauthenticated requests and optionally restricts roles.
func Require(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok {
				shared.WriteError(w, shared.ErrUnauthorized)
				return
			}
			if len(roles) > 0 {
				allowed := false
				for _, role := range roles {
					if p.Role == role {
						allowed = true
					}
				}
				if !allowed {
					shared.WriteError(w, shared.ErrForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireConfirmed blocks waiting/rejected users from using cabinet features.
func RequireConfirmed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := FromContext(r.Context())
		if !ok {
			shared.WriteError(w, shared.ErrUnauthorized)
			return
		}
		if p.Role != RoleAdmin && p.Status != StatusConfirmed {
			shared.WriteError(w, shared.ErrNotApproved)
			return
		}
		next.ServeHTTP(w, r)
	})
}
