package interfaces

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

type Handler struct {
	svc          *application.Service
	recommender  *application.Recommender
	conversations application.ConversationStore
}

func NewHandler(svc *application.Service, recommender *application.Recommender, conversations application.ConversationStore) *Handler {
	return &Handler{svc: svc, recommender: recommender, conversations: conversations}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	guard := auth.Chain(auth.Require(auth.RoleAdmin, auth.RoleSupplier, auth.RoleCustomer), auth.RequireConfirmed)
	mux.Handle("POST /api/v1/ai/chat", guard(http.HandlerFunc(h.chat)))
	mux.Handle("GET /api/v1/ai/recommendations", guard(http.HandlerFunc(h.recommendations)))
	mux.Handle("GET /api/v1/ai/conversations", guard(http.HandlerFunc(h.listConversations)))
	mux.Handle("GET /api/v1/ai/conversations/{id}", guard(http.HandlerFunc(h.getConversation)))
	mux.Handle("DELETE /api/v1/ai/conversations/{id}", guard(http.HandlerFunc(h.deleteConversation)))
	mux.HandleFunc("GET /api/v1/market/prices", h.marketPrices)
}

func (h *Handler) chat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Message        string               `json:"message"`
		ConversationID string               `json:"conversation_id"`
		Locale         string               `json:"locale"`
		Mode           string               `json:"mode"`
		History        []domain.ChatMessage `json:"history"`
	}
	if err := shared.Decode(r, &in); err != nil {
		shared.WriteError(w, err)
		return
	}
	if in.Message == "" {
		shared.WriteError(w, shared.WithStatus(errors.New("message is required"), 400))
		return
	}
	principal, _ := auth.FromContext(r.Context())
	if in.Mode == "shopping" && principal.Role != auth.RoleCustomer {
		in.Mode = "general"
	}
	reply, err := h.svc.Chat(r.Context(), principal, application.ChatInput{
		Message: in.Message, ConversationID: in.ConversationID, Locale: in.Locale,
		Mode: in.Mode, History: in.History,
	})
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, reply)
}

func (h *Handler) recommendations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	principal, _ := auth.FromContext(r.Context())
	set, err := h.recommender.RecommendSet(r.Context(), r.URL.Query().Get("product_id"), principal, limit)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, set)
}

func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.FromContext(r.Context())
	items, err := h.conversations.List(r.Context(), principal)
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, map[string]any{"data": items, "total": len(items)})
}

func (h *Handler) getConversation(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.FromContext(r.Context())
	detail, err := h.conversations.Get(r.Context(), principal, r.PathValue("id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, detail)
}

func (h *Handler) deleteConversation(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.FromContext(r.Context())
	if err := h.conversations.Delete(r.Context(), principal, r.PathValue("id")); err != nil {
		shared.WriteError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) marketPrices(w http.ResponseWriter, r *http.Request) {
	// Delegated to the AI module's stats reader through the service.
	result, err := h.svc.MarketPrices(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("category_id"))
	if err != nil {
		shared.WriteError(w, err)
		return
	}
	shared.WriteJSON(w, 200, result)
}
