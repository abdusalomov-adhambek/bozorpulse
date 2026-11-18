package interfaces

import (
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	guard := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier, auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/payments", guard(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/payments", guard(http.HandlerFunc(h.create)))
	mux.Handle("GET /api/v1/payments/{id}", guard(http.HandlerFunc(h.get)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset := shared.PageParams(r)
	items, total, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, shared.List[domain.Payment]{Data: items, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var p domain.Payment
	if err := shared.Decode(r, &p); err != nil {
		shared.WriteError(w, err)
		return
	}
	created, err := h.svc.Create(r.Context(), p)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, created)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, p)
}
