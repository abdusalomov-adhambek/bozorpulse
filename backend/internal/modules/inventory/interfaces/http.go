package interfaces

import (
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	guard := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier, auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/inventory/stock", guard(http.HandlerFunc(h.stock)))
	mux.Handle("GET /api/v1/inventory/movements", guard(http.HandlerFunc(h.movements)))
	mux.Handle("POST /api/v1/inventory/supplies", guard(http.HandlerFunc(h.supply)))
	mux.Handle("POST /api/v1/inventory/transfers", guard(http.HandlerFunc(h.transfer)))
	mux.Handle("POST /api/v1/inventory/write-offs", guard(http.HandlerFunc(h.writeOff)))
	mux.Handle("POST /api/v1/inventory/returns", guard(http.HandlerFunc(h.ret)))
}

func (h *Handler) stock(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	items, total, err := h.svc.Stock(r.Context(), p, r.URL.Query().Get("warehouse_id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": items, "total": total})
}

func (h *Handler) movements(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	items, total, err := h.svc.Movements(r.Context(), p, r.URL.Query().Get("warehouse_id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": items, "total": total})
}

func decodeInput(w http.ResponseWriter, r *http.Request, in *domain.MovementInput) bool {
	if err := shared.Decode(r, in); err != nil {
		shared.WriteError(w, err)
		return false
	}
	return true
}

func (h *Handler) supply(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var in domain.MovementInput
	if !decodeInput(w, r, &in) {
		return
	}
	m, err := h.svc.Supply(r.Context(), p, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, m)
}

func (h *Handler) transfer(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var in domain.MovementInput
	if !decodeInput(w, r, &in) {
		return
	}
	m, err := h.svc.Transfer(r.Context(), p, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, m)
}

func (h *Handler) writeOff(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var in domain.MovementInput
	if !decodeInput(w, r, &in) {
		return
	}
	m, err := h.svc.WriteOff(r.Context(), p, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, m)
}

func (h *Handler) ret(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var in domain.MovementInput
	if !decodeInput(w, r, &in) {
		return
	}
	m, err := h.svc.Return(r.Context(), p, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, m)
}
