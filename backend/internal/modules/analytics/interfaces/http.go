package interfaces

import (
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	guard := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier, auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/analytics/dashboard", guard(http.HandlerFunc(h.dashboard)))
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	shared.WriteJSON(w, 200, h.svc.Dashboard(r.Context(), p))
}
