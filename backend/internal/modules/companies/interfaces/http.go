package interfaces

import (
	"errors"
	"net/http"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/suppliers", h.suppliers)
	mux.HandleFunc("GET /api/v1/suppliers/{id}", h.supplier)

	admin := auth.Chain(auth.Require(auth.RoleAdmin), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/customers", admin(http.HandlerFunc(h.customers)))

	confirmed := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier, auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/companies/me", confirmed(http.HandlerFunc(h.myCompany)))
	mux.Handle("PATCH /api/v1/companies/me", confirmed(http.HandlerFunc(h.updateCompany)))
	mux.Handle("GET /api/v1/warehouses", confirmed(http.HandlerFunc(h.warehouses)))
	mux.Handle("POST /api/v1/warehouses", confirmed(http.HandlerFunc(h.createWarehouse)))
	mux.Handle("GET /api/v1/warehouses/{id}", confirmed(http.HandlerFunc(h.warehouse)))
	mux.Handle("PATCH /api/v1/warehouses/{id}", confirmed(http.HandlerFunc(h.updateWarehouse)))
	mux.Handle("DELETE /api/v1/warehouses/{id}", confirmed(http.HandlerFunc(h.deleteWarehouse)))
}

func (h *Handler) suppliers(w http.ResponseWriter, r *http.Request) {
	limit, offset := shared.PageParams(r)
	items, total, err := h.svc.Suppliers(r.Context(), limit, offset)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, shared.List[domain.Company]{Data: items, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) supplier(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Company(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, c)
}

func (h *Handler) customers(w http.ResponseWriter, r *http.Request) {
	limit, offset := shared.PageParams(r)
	items, total, err := h.svc.Customers(r.Context(), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, shared.List[domain.Company]{Data: items, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) myCompany(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	c, err := h.svc.Company(r.Context(), p.CompanyID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, c)
}

func (h *Handler) updateCompany(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var input struct {
		Name        string `json:"name"`
		CompanyName string `json:"company_name"`
		INN         string `json:"inn"`
		BankAccount string `json:"bank_account"`
		RegionID    string `json:"region_id"`
	}
	if err := shared.Decode(r, &input); err != nil {
		shared.WriteError(w, err)
		return
	}
	in := domain.Company{
		Name:        input.Name,
		INN:         input.INN,
		BankAccount: input.BankAccount,
		RegionID:    input.RegionID,
	}
	if in.Name == "" {
		in.Name = input.CompanyName
	}
	// INN and bank account are registration-only details. Once an account is
	// confirmed, they cannot be changed through the cabinet.
	if p.Role == auth.RoleAdmin || p.Status == auth.StatusConfirmed {
		existing, err := h.svc.Company(r.Context(), p.CompanyID)
		if err != nil {
			shared.WriteError(w, err)
			return
		}
		innChanged := in.INN != "" && in.INN != existing.INN
		bankChanged := in.BankAccount != "" && in.BankAccount != existing.BankAccount
		if innChanged || bankChanged {
			shared.WriteError(w, shared.WithStatus(errors.New("INN and bank account cannot be changed after approval"), 403))
			return
		}
		in.INN = existing.INN
		in.BankAccount = existing.BankAccount
	}
	c, err := h.svc.UpdateCompany(r.Context(), p.CompanyID, in)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, c)
}

func (h *Handler) warehouses(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	items, err := h.svc.Warehouses(r.Context(), p.CompanyID, p.Role == auth.RoleAdmin)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": items, "total": len(items)})
}

func (h *Handler) warehouse(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	wh, err := h.svc.Warehouse(r.Context(), r.PathValue("id"), p.CompanyID, p.Role == auth.RoleAdmin)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, wh)
}

func (h *Handler) createWarehouse(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var wh domain.Warehouse
	if err := shared.Decode(r, &wh); err != nil {
		shared.WriteError(w, err)
		return
	}
	wh.OwnerID, wh.OwnerRole = p.CompanyID, p.Role
	created, err := h.svc.CreateWarehouse(r.Context(), wh)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, created)
}

func (h *Handler) updateWarehouse(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var wh domain.Warehouse
	if err := shared.Decode(r, &wh); err != nil {
		shared.WriteError(w, err)
		return
	}
	wh.ID = r.PathValue("id")
	existing, err := h.svc.Warehouse(r.Context(), wh.ID, p.CompanyID, p.Role == auth.RoleAdmin)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	wh.OwnerID, wh.OwnerRole = existing.OwnerID, existing.OwnerRole
	updated, err := h.svc.UpdateWarehouse(r.Context(), wh)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, updated)
}

func (h *Handler) deleteWarehouse(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	wh, err := h.svc.Warehouse(r.Context(), r.PathValue("id"), p.CompanyID, p.Role == auth.RoleAdmin)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if err := h.svc.DeleteWarehouse(r.Context(), wh.ID); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}
