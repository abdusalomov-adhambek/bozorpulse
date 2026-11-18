package interfaces

import (
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	guard := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier, auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/orders", guard(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/orders", guard(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/orders/{id}", guard(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/orders/{id}/status", guard(http.HandlerFunc(h.status)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	limit, offset := shared.PageParams(r)
	items, total, err := h.svc.List(r.Context(), p, limit, offset)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, shared.List[domain.Order]{Data: items, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	if p.Role != auth.RoleCustomer {
		shared.WriteError(w, shared.ErrForbidden)
		return
	}
	var in domain.CreateInput
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	o, err := h.svc.Create(r.Context(), p.CompanyID, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, o)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	o, err := h.svc.Get(r.Context(), r.PathValue("id"), p)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, o)
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var in struct {
		Status string `json:"status"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	o, err := h.svc.SetStatus(r.Context(), r.PathValue("id"), in.Status, p)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, o)
}
