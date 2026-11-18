// Package interfaces exposes planning over HTTP. Handlers only decode JSON,
// call one use case and map the result/errors.
package interfaces

import (
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	guard := auth.Chain(auth.Require(auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("POST /api/v1/plans", guard(http.HandlerFunc(h.create)))
	mux.Handle("POST /api/v1/plans/preview", guard(http.HandlerFunc(h.preview)))
	mux.Handle("GET /api/v1/plans", guard(http.HandlerFunc(h.list)))
	mux.Handle("GET /api/v1/plans/{id}", guard(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /api/v1/plans/{id}", guard(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /api/v1/plans/{id}", guard(http.HandlerFunc(h.remove)))
	mux.Handle("POST /api/v1/plans/{id}/refresh", guard(http.HandlerFunc(h.refresh)))
}

type itemRequest struct {
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
	TargetDate string `json:"target_date"`
	Note       string `json:"note"`
}

type planRequest struct {
	Title    string        `json:"title"`
	RegionID string        `json:"region_id"`
	Status   string        `json:"status"`
	Items    []itemRequest `json:"items"`
}

func (in planRequest) input() application.PlanInput {
	// Preserve nil so a partial PATCH that omits "items" does not look like an
	// explicit (invalid) empty item list.
	var items []application.ItemInput
	if in.Items != nil {
		items = make([]application.ItemInput, 0, len(in.Items))
		for _, it := range in.Items {
			items = append(items, application.ItemInput{
				ProductID: it.ProductID, Quantity: it.Quantity,
				TargetDate: it.TargetDate, Note: it.Note,
			})
		}
	}
	return application.PlanInput{Title: in.Title, RegionID: in.RegionID, Status: in.Status, Items: items}
}

func decodePlan(w http.ResponseWriter, r *http.Request) (planRequest, bool) {
	var in planRequest
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return in, false
	}
	return in, true
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	in, ok := decodePlan(w, r)
	if !ok {
		return
	}
	p, _ := auth.FromContext(r.Context())
	plan, err := h.svc.Create(r.Context(), p, in.input())
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, plan)
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	in, ok := decodePlan(w, r)
	if !ok {
		return
	}
	p, _ := auth.FromContext(r.Context())
	plan, err := h.svc.Preview(r.Context(), p, in.input())
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, plan)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	plans, err := h.svc.List(r.Context(), p)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]any{"data": plans, "total": len(plans)})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	plan, err := h.svc.Get(r.Context(), p, r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, plan)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	in, ok := decodePlan(w, r)
	if !ok {
		return
	}
	p, _ := auth.FromContext(r.Context())
	plan, err := h.svc.Update(r.Context(), p, r.PathValue("id"), in.input())
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, plan)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	if err := h.svc.Delete(r.Context(), p, r.PathValue("id")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	plan, err := h.svc.Refresh(r.Context(), p, r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, plan)
}
