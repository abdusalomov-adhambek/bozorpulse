package interfaces

import (
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("GET /api/v1/auth/me", auth.Require()(http.HandlerFunc(h.me)).ServeHTTP)

	admin := auth.Chain(auth.Require(auth.RoleAdmin), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/admin/users", admin(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/admin/users/{id}/approve", admin(http.HandlerFunc(h.approve)))
	mux.Handle("POST /api/v1/admin/users/{id}/reject", admin(http.HandlerFunc(h.reject)))
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var in domain.RegisterInput
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	u, token, err := h.svc.Register(r.Context(), in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, map[string]interface{}{"user": u, "token": token})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	u, token, err := h.svc.Login(r.Context(), in.Username, in.Password)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]interface{}{"user": u, "token": token})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	u, err := h.svc.Me(r.Context(), p.ID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, u)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset := shared.PageParams(r)
	users, total, err := h.svc.List(r.Context(), r.URL.Query().Get("role"), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, shared.List[domain.User]{Data: users, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, auth.StatusConfirmed)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, auth.StatusRejected)
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status string) {
	u, err := h.svc.SetStatus(r.Context(), r.PathValue("id"), status)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, u)
}
