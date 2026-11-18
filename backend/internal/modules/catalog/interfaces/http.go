package interfaces

import (
	"net/http"
	"strconv"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct{ svc *application.Service }

func NewHandler(svc *application.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Public catalog and price feed.
	mux.HandleFunc("GET /api/v1/categories", h.categories)
	mux.HandleFunc("GET /api/v1/regions", h.regions)
	mux.HandleFunc("GET /api/v1/products", h.products)
	mux.HandleFunc("GET /api/v1/products/{id}", h.product)

	admin := auth.Chain(auth.Require(auth.RoleAdmin), auth.RequireConfirmed)
	mux.Handle("POST /api/v1/categories", admin(http.HandlerFunc(h.createCategory)))
	mux.Handle("PATCH /api/v1/categories/{id}", admin(http.HandlerFunc(h.updateCategory)))
	mux.Handle("DELETE /api/v1/categories/{id}", admin(http.HandlerFunc(h.deleteCategory)))
	mux.Handle("POST /api/v1/regions", admin(http.HandlerFunc(h.createRegion)))

	write := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier), auth.RequireConfirmed)
	mux.Handle("POST /api/v1/products", write(http.HandlerFunc(h.createProduct)))
	mux.Handle("PATCH /api/v1/products/{id}", write(http.HandlerFunc(h.updateProduct)))
	mux.Handle("DELETE /api/v1/products/{id}", write(http.HandlerFunc(h.deleteProduct)))
	mux.Handle("POST /api/v1/products/{id}/discount", write(http.HandlerFunc(h.discount)))
	mux.HandleFunc("GET /api/v1/products/{id}/pricing", h.pricing)
	mux.HandleFunc("GET /api/v1/products/{id}/price-history", h.priceHistory)
	mux.Handle("POST /api/v1/products/{id}/pricing", write(http.HandlerFunc(h.createPricing)))
	mux.Handle("DELETE /api/v1/products/{id}/pricing/{ruleId}", write(http.HandlerFunc(h.deletePricing)))
	mux.HandleFunc("GET /api/v1/products/{id}/price", h.resolvePrice)

	supplierWrite := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier), auth.RequireConfirmed)
	mux.Handle("GET /api/v1/suppliers/me/delivery-zones", supplierWrite(http.HandlerFunc(h.myDeliveryZones)))
	mux.Handle("POST /api/v1/suppliers/me/delivery-zones", supplierWrite(http.HandlerFunc(h.createDeliveryZone)))
	mux.Handle("PATCH /api/v1/suppliers/me/delivery-zones/{id}", supplierWrite(http.HandlerFunc(h.updateDeliveryZone)))
	mux.Handle("DELETE /api/v1/suppliers/me/delivery-zones/{id}", supplierWrite(http.HandlerFunc(h.deleteDeliveryZone)))
	mux.HandleFunc("GET /api/v1/suppliers/{id}/delivery-zones", h.supplierDeliveryZones)
}

func (h *Handler) categories(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Categories(r.Context())
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": items, "total": len(items)})
}

func (h *Handler) regions(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Regions(r.Context())
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": items, "total": len(items)})
}

func (h *Handler) products(w http.ResponseWriter, r *http.Request) {
	limit, offset := shared.PageParams(r)
	minP, _ := strconv.ParseFloat(r.URL.Query().Get("min_price"), 64)
	maxP, _ := strconv.ParseFloat(r.URL.Query().Get("max_price"), 64)
	items, total, err := h.svc.Products(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("category_id"),
		r.URL.Query().Get("supplier_id"), minP, maxP, limit, offset)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, shared.List[domain.Product]{Data: items, Total: total, Limit: limit, Offset: offset})
}

func (h *Handler) product(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Product(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, p)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	c, err := h.svc.CreateCategory(r.Context(), in.Name, in.ParentID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, c)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		ParentID string `json:"parent_id"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	c, err := h.svc.UpdateCategory(r.Context(), r.PathValue("id"), in.Name, in.ParentID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, c)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCategory(r.Context(), r.PathValue("id")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) createRegion(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	rg, err := h.svc.CreateRegion(r.Context(), in.Name)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, rg)
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var p domain.Product
	if err := shared.Decode(r, &p); err != nil {
		shared.WriteError(w, err)
		return
	}
	principal, _ := auth.FromContext(r.Context())
	if principal.Role == auth.RoleSupplier {
		if p.SupplierID != "" && p.SupplierID != principal.CompanyID {
			shared.WriteError(w, shared.ErrForbidden)
			return
		}
		p.SupplierID = principal.CompanyID
	}
	created, err := h.svc.CreateProduct(r.Context(), p)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, created)
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	var p domain.Product
	if err := shared.Decode(r, &p); err != nil {
		shared.WriteError(w, err)
		return
	}
	p.ID = r.PathValue("id")
	principal, _ := auth.FromContext(r.Context())
	existing, err := h.svc.Product(r.Context(), p.ID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if principal.Role == auth.RoleSupplier && existing.SupplierID != principal.CompanyID {
		shared.WriteError(w, shared.ErrForbidden)
		return
	}
	p.SupplierID = existing.SupplierID
	updated, err := h.svc.UpdateProduct(r.Context(), p)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, updated)
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.FromContext(r.Context())
	existing, err := h.svc.Product(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if principal.Role == auth.RoleSupplier && existing.SupplierID != principal.CompanyID {
		shared.WriteError(w, shared.ErrForbidden)
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), existing.ID); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) discount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DiscountPercent float64 `json:"discount_percent"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	principal, _ := auth.FromContext(r.Context())
	existing, err := h.svc.Product(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if principal.Role == auth.RoleSupplier && existing.SupplierID != principal.CompanyID {
		shared.WriteError(w, shared.ErrForbidden)
		return
	}
	p, err := h.svc.SetDiscount(r.Context(), existing.ID, in.DiscountPercent)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, p)
}

func (h *Handler) priceHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	history, err := h.svc.PriceHistory(r.Context(), r.PathValue("id"), limit)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, history)
}

func (h *Handler) pricing(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Pricing(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": items, "total": len(items)})
}

func (h *Handler) createPricing(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.FromContext(r.Context())
	product, err := h.svc.Product(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if principal.Role == auth.RoleSupplier && product.SupplierID != principal.CompanyID {
		shared.WriteError(w, shared.ErrForbidden)
		return
	}
	var rule domain.PricingRule
	if err := shared.Decode(r, &rule); err != nil {
		shared.WriteError(w, err)
		return
	}
	rule.ProductID = product.ID
	created, err := h.svc.CreatePricing(r.Context(), rule)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, created)
}

func (h *Handler) deletePricing(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.FromContext(r.Context())
	product, err := h.svc.Product(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	if principal.Role == auth.RoleSupplier && product.SupplierID != principal.CompanyID {
		shared.WriteError(w, shared.ErrForbidden)
		return
	}
	if err := h.svc.DeletePricing(r.Context(), r.PathValue("ruleId")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) resolvePrice(w http.ResponseWriter, r *http.Request) {
	qty, _ := strconv.Atoi(r.URL.Query().Get("quantity"))
	price, err := h.svc.ResolvePrice(r.Context(), r.PathValue("id"), r.URL.Query().Get("warehouse_id"), qty)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, price)
}

func (h *Handler) myDeliveryZones(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	zones, err := h.svc.DeliveryZones(r.Context(), p.CompanyID)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": zones, "total": len(zones)})
}

func (h *Handler) supplierDeliveryZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.svc.DeliveryZones(r.Context(), r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]interface{}{"data": zones, "total": len(zones)})
}

func (h *Handler) createDeliveryZone(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var zone domain.DeliveryZone
	if err := shared.Decode(r, &zone); err != nil {
		shared.WriteError(w, err)
		return
	}
	zone.SupplierID = p.CompanyID
	created, err := h.svc.CreateDeliveryZone(r.Context(), zone)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 201, created)
}

func (h *Handler) updateDeliveryZone(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	var zone domain.DeliveryZone
	if err := shared.Decode(r, &zone); err != nil {
		shared.WriteError(w, err)
		return
	}
	zone.ID = r.PathValue("id")
	zone.SupplierID = p.CompanyID
	updated, err := h.svc.UpdateDeliveryZone(r.Context(), zone)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, updated)
}

func (h *Handler) deleteDeliveryZone(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	if err := h.svc.DeleteDeliveryZone(r.Context(), r.PathValue("id"), p.CompanyID); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}
