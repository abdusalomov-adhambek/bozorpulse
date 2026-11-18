// Command api starts the BozorPulse modular-monolith API.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	aiApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/application"
	aiInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/infrastructure"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/infrastructure/forecast"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/infrastructure/llm"
	aiInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/ai/interfaces"
	analyticsApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/application"
	analyticsInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/infrastructure"
	analyticsInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/analytics/interfaces"
	catalogApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/application"
	catalogInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/infrastructure"
	catalogInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/catalog/interfaces"
	companiesApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/application"
	companiesInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/infrastructure"
	companiesInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/companies/interfaces"
	identityApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/application"
	identityInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/infrastructure"
	identityInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/identity/interfaces"
	inventoryApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/application"
	inventoryInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/infrastructure"
	inventoryInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/inventory/interfaces"
	paymentsApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/application"
	paymentsInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/infrastructure"
	paymentsInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/payments/interfaces"
	planningApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/application"
	planningInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/infrastructure"
	planningInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/interfaces"
	procurementApp "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/application"
	procurementInfra "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/infrastructure"
	procurementInterfaces "github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/procurement/interfaces"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/config"
	bhttp "github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/http"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/postgres"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

func main() {
	cfg := config.Load()
	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	sessions := auth.NewSessions(db)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		shared.WriteJSON(w, 200, map[string]string{"status": "ok", "service": "bozorpulse-api"})
	})

	identitySvc := identityApp.New(identityInfra.NewRepository(db), sessions)
	identityInterfaces.NewHandler(identitySvc).RegisterRoutes(mux)

	catalogSvc := catalogApp.New(catalogInfra.NewRepository(db))
	catalogInterfaces.NewHandler(catalogSvc).RegisterRoutes(mux)

	companiesSvc := companiesApp.New(companiesInfra.NewRepository(db))
	companiesInterfaces.NewHandler(companiesSvc).RegisterRoutes(mux)

	procurementSvc := procurementApp.New(procurementInfra.NewRepository(db))
	procurementInterfaces.NewHandler(procurementSvc).RegisterRoutes(mux)

	paymentsSvc := paymentsApp.New(paymentsInfra.NewRepository(db))
	paymentsInterfaces.NewHandler(paymentsSvc).RegisterRoutes(mux)

	inventorySvc := inventoryApp.New(inventoryInfra.NewRepository(db))
	inventoryInterfaces.NewHandler(inventorySvc).RegisterRoutes(mux)

	analyticsSvc := analyticsApp.New(analyticsInfra.NewRepository(db))
	analyticsInterfaces.NewHandler(analyticsSvc).RegisterRoutes(mux)

	// --- AI module: provider, tools, recommendations, conversation store ---
	var provider aiApp.LLMProvider = llm.NewRuleProvider()
	if cfg.AILLMEnabled {
		if cfg.LLMAPIKey == "" {
			log.Printf("WARN: AI_LLM_ENABLED=true but MODEL_API_KEY/LLM_API_KEY is empty; using rule-based assistant")
		} else {
			provider = llm.NewMetaProvider(cfg)
			log.Printf("AI assistant: Meta Model API enabled [model=%s]", cfg.LLMModel)
		}
	} else {
		log.Printf("AI assistant: rule-based mode (AI_LLM_ENABLED=false)")
	}

	aiRepo := aiInfra.NewRepository(db)
	recommender := aiApp.NewRecommender(aiRepo)
	conversationStore := aiInfra.NewConversationStore(db)
	productSearcher := aiInfra.NewProductSearchAdapter(catalogSvc, aiRepo)
	orderGetter := aiInfra.NewOrderGetAdapter(procurementSvc)
	forecaster := forecast.New(db, forecast.NewOpenMeteo(nil))
	offerComparer := aiInfra.NewOfferCompareAdapter(catalogSvc, companiesSvc, aiRepo, forecaster)

	registry := aiApp.NewRegistry(
		aiApp.NewMarketStatsTool(aiRepo),
		aiApp.NewRecommendTool(recommender),
		aiApp.NewSearchTool(productSearcher),
		aiApp.NewGetOrderTool(orderGetter),
		aiApp.NewCompareOffersTool(offerComparer),
		aiApp.NewPriceForecastTool(forecaster, productSearcher),
	)
	aiSvc := aiApp.New(aiRepo, provider, registry, aiRepo, productSearcher, orderGetter, offerComparer, conversationStore, cfg)
	aiInterfaces.NewHandler(aiSvc, recommender, conversationStore).RegisterRoutes(mux)

	// --- Planning module: purchase-timing plans built on price forecasts ---
	// Planning uses an offline forecaster (no weather network call) so plan
	// creation stays fast and deterministic; the assistant keeps the weather
	// aware instance above.
	planningForecaster := forecast.New(db, nil)
	planningSvc := planningApp.New(
		planningInfra.NewRepository(db),
		planningInfra.NewProductAdapter(catalogSvc),
		planningInfra.NewForecastAdapter(planningForecaster),
	)
	planningInterfaces.NewHandler(planningSvc).RegisterRoutes(mux)

	handler := bhttp.Chain(mux, bhttp.Logging, bhttp.Recover, bhttp.CORS, sessions.Middleware)
	srv := &http.Server{
		Addr: cfg.Addr, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("BozorPulse API listening on %s [env=%s]", cfg.Addr, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("shutdown complete")
}
