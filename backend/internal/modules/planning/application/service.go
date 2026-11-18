// Package application orchestrates planning use cases. All arithmetic,
// validation and ownership checks live here — never in HTTP handlers.
package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

const maxQuantity = 10000

// Service owns the planning use cases.
type Service struct {
	repo       PlanRepository
	products   ProductReader
	forecaster Forecaster
	now        func() time.Time
}

func New(repo PlanRepository, products ProductReader, forecaster Forecaster) *Service {
	return &Service{repo: repo, products: products, forecaster: forecaster, now: time.Now}
}

// Create computes projections for every item and persists the plan.
func (s *Service) Create(ctx context.Context, p auth.Principal, in PlanInput) (domain.Plan, error) {
	if err := s.authorize(p); err != nil {
		return domain.Plan{}, err
	}
	now := s.now()
	title, err := validateHeader(in)
	if err != nil {
		return domain.Plan{}, err
	}
	items, err := s.itemsFromInput(in.Items)
	if err != nil {
		return domain.Plan{}, err
	}
	plan := domain.Plan{
		ID:         "plan-" + newID(),
		CompanyID:  ownerKey(p),
		CustomerID: p.ID,
		Title:      title,
		RegionID:   in.RegionID,
		Status:     statusOrDefault(in.Status),
		Items:      items,
		CreatedAt:  now.UTC().Format(time.RFC3339),
		UpdatedAt:  now.UTC().Format(time.RFC3339),
	}
	if err := s.project(ctx, plan.RegionID, plan.Items, nil, now); err != nil {
		return domain.Plan{}, err
	}
	if err := s.repo.Save(ctx, plan); err != nil {
		return domain.Plan{}, err
	}
	return labelPlan(plan), nil
}

// Preview computes the same projections without persisting anything.
func (s *Service) Preview(ctx context.Context, p auth.Principal, in PlanInput) (domain.Plan, error) {
	if err := s.authorize(p); err != nil {
		return domain.Plan{}, err
	}
	now := s.now()
	title, err := validateHeader(in)
	if err != nil {
		return domain.Plan{}, err
	}
	items, err := s.itemsFromInput(in.Items)
	if err != nil {
		return domain.Plan{}, err
	}
	plan := domain.Plan{
		ID: "preview", CompanyID: ownerKey(p), CustomerID: p.ID,
		Title: title, RegionID: in.RegionID, Status: statusOrDefault(in.Status),
		Items: items, CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339),
	}
	if err := s.project(ctx, plan.RegionID, plan.Items, nil, now); err != nil {
		return domain.Plan{}, err
	}
	return labelPlan(plan), nil
}

// List returns the caller's plans (never another company's).
func (s *Service) List(ctx context.Context, p auth.Principal) ([]domain.Plan, error) {
	if err := s.authorize(p); err != nil {
		return nil, err
	}
	plans, err := s.repo.List(ctx, ownerKey(p))
	if err != nil {
		return nil, err
	}
	for i := range plans {
		plans[i] = labelPlan(plans[i])
	}
	return plans, nil
}

// Get returns one owner-scoped plan.
func (s *Service) Get(ctx context.Context, p auth.Principal, id string) (domain.Plan, error) {
	if err := s.authorize(p); err != nil {
		return domain.Plan{}, err
	}
	plan, err := s.repo.Get(ctx, ownerKey(p), id)
	if err != nil {
		return domain.Plan{}, err
	}
	return labelPlan(plan), nil
}

// Update edits a plan. Projections are recomputed only for changed items.
func (s *Service) Update(ctx context.Context, p auth.Principal, id string, in PlanInput) (domain.Plan, error) {
	if err := s.authorize(p); err != nil {
		return domain.Plan{}, err
	}
	plan, err := s.repo.Get(ctx, ownerKey(p), id)
	if err != nil {
		return domain.Plan{}, err
	}
	now := s.now()
	if strings.TrimSpace(in.Title) != "" {
		plan.Title = strings.TrimSpace(in.Title)
	}
	if in.RegionID != "" {
		plan.RegionID = in.RegionID
	}
	if in.Status != "" {
		if !domain.ValidateStatus(in.Status) {
			return domain.Plan{}, shared.WithStatus(errors.New("invalid plan status"), 422)
		}
		plan.Status = domain.PlanStatus(in.Status)
	}
	if in.Items != nil {
		items, err := s.itemsFromInput(in.Items)
		if err != nil {
			return domain.Plan{}, err
		}
		reuse := make(map[string]domain.PlanItem, len(plan.Items))
		for _, old := range plan.Items {
			reuse[itemKey(old)] = old
		}
		for i := range items {
			if old, ok := reuse[itemKey(items[i])]; ok {
				items[i].ID = old.ID
			} else {
				items[i].ID = "pi-" + newID()
			}
		}
		plan.Items = items
		if err := s.project(ctx, plan.RegionID, plan.Items, reuse, now); err != nil {
			return domain.Plan{}, err
		}
	}
	plan.UpdatedAt = now.UTC().Format(time.RFC3339)
	if err := s.repo.Save(ctx, plan); err != nil {
		return domain.Plan{}, err
	}
	return labelPlan(plan), nil
}

// Delete removes one owner-scoped plan.
func (s *Service) Delete(ctx context.Context, p auth.Principal, id string) error {
	if err := s.authorize(p); err != nil {
		return err
	}
	return s.repo.Delete(ctx, ownerKey(p), id)
}

// Refresh recomputes every projection from fresh forecasts.
func (s *Service) Refresh(ctx context.Context, p auth.Principal, id string) (domain.Plan, error) {
	if err := s.authorize(p); err != nil {
		return domain.Plan{}, err
	}
	plan, err := s.repo.Get(ctx, ownerKey(p), id)
	if err != nil {
		return domain.Plan{}, err
	}
	now := s.now()
	if err := s.project(ctx, plan.RegionID, plan.Items, nil, now); err != nil {
		return domain.Plan{}, err
	}
	plan.UpdatedAt = now.UTC().Format(time.RFC3339)
	if err := s.repo.Save(ctx, plan); err != nil {
		return domain.Plan{}, err
	}
	return labelPlan(plan), nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (s *Service) authorize(p auth.Principal) error {
	if p.Role != auth.RoleCustomer {
		return shared.WithStatus(errors.New("only customers can manage plans"), 403)
	}
	return nil
}

// labelPlan guarantees the demo disclaimer on every demo-backed projection,
// including plans loaded from storage where only data_status is persisted.
func labelPlan(plan domain.Plan) domain.Plan {
	for i := range plan.Items {
		if plan.Items[i].Projection.DataStatus == domain.StatusDemo && plan.Items[i].Projection.Disclaimer == "" {
			plan.Items[i].Projection.Disclaimer = domain.DemoDisclaimer
		}
	}
	return plan
}

func ownerKey(p auth.Principal) string {
	if p.CompanyID != "" {
		return p.CompanyID
	}
	return p.ID // fallback when the principal has no company row
}

// validateHeader checks the fields shared by create/update/preview.
func validateHeader(in PlanInput) (string, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return "", shared.WithStatus(errors.New("title is required"), 400)
	}
	if in.Status != "" && !domain.ValidateStatus(in.Status) {
		return "", shared.WithStatus(errors.New("invalid plan status"), 422)
	}
	return title, nil
}

func statusOrDefault(s string) domain.PlanStatus {
	if domain.ValidateStatus(s) {
		return domain.PlanStatus(s)
	}
	return domain.PlanActive
}

func (s *Service) itemsFromInput(in []ItemInput) ([]domain.PlanItem, error) {
	if len(in) == 0 {
		return nil, shared.WithStatus(errors.New("at least one plan item is required"), 422)
	}
	out := make([]domain.PlanItem, 0, len(in))
	for i, raw := range in {
		productID := strings.TrimSpace(raw.ProductID)
		if productID == "" {
			return nil, shared.WithStatus(fmt.Errorf("item %d: product_id is required", i+1), 422)
		}
		if raw.Quantity < 1 || raw.Quantity > maxQuantity {
			return nil, shared.WithStatus(fmt.Errorf("item %d: quantity must be between 1 and %d", i+1, maxQuantity), 422)
		}
		target := strings.TrimSpace(raw.TargetDate)
		if !domain.ValidateTargetDate(target, s.now()) {
			return nil, shared.WithStatus(fmt.Errorf("item %d: target_date must be within the 30-day forecast horizon", i+1), 422)
		}
		out = append(out, domain.PlanItem{
			ID: "pi-" + newID(), ProductID: productID, Quantity: raw.Quantity,
			TargetDate: target, Note: strings.TrimSpace(raw.Note),
		})
	}
	return out, nil
}

func itemKey(item domain.PlanItem) string {
	return fmt.Sprintf("%s|%d|%s", item.ProductID, item.Quantity, item.TargetDate)
}

func toForecastInput(fc ForecastOutcome) domain.ForecastInput {
	return domain.ForecastInput{
		Predicted: fc.Predicted, Low: fc.Low, High: fc.High,
		Baseline: fc.Baseline, ChangePct: fc.ChangePct, TrendPct: fc.TrendPct,
		HorizonDays: fc.HorizonDays, Samples: fc.Samples,
		DataStatus: fc.DataStatus, Basis: fc.Basis, Warnings: fc.Warnings,
	}
}

// project resolves prices and forecasts with bounded parallelism. A single
// failing item fails the whole plan so partial, misleading data is never saved.
func (s *Service) project(ctx context.Context, regionID string, items []domain.PlanItem, reuse map[string]domain.PlanItem, now time.Time) error {
	const maxParallel = 4
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	record := func(err error) {
		mu.Lock()
		errs = append(errs, err)
		mu.Unlock()
	}

	for i := range items {
		key := itemKey(items[i])
		if old, ok := reuse[key]; ok && old.Projection.Recommendation != domain.RecInsufficient {
			items[i].Projection = old.Projection
			continue
		}
		if !domain.ValidateTargetDate(items[i].TargetDate, now) {
			items[i].Projection = domain.InsufficientProjection("Target date is outside the 30-day forecast range.", now)
			continue
		}
		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			record(ctx.Err())
			wg.Done()
			continue
		}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			item := &items[i]
			info, err := s.products.Product(ctx, item.ProductID)
			if err != nil {
				record(fmt.Errorf("item %d: product: %w", i+1, err))
				return
			}
			item.ProductName = info.Name
			item.Unit = info.Unit
			tier, err := s.products.BestTierPrice(ctx, item.ProductID, item.Quantity)
			if err != nil {
				record(fmt.Errorf("item %d: pricing: %w", i+1, err))
				return
			}
			if tier <= 0 {
				tier = info.BasePrice
			}
			fc, err := s.forecaster.Forecast(ctx, item.ProductID, regionID)
			if err != nil {
				record(fmt.Errorf("item %d: forecast: %w", i+1, err))
				return
			}
			item.Projection = domain.BuildProjection(toForecastInput(fc), tier, item.Quantity, now)
		}(i)
	}
	wg.Wait()
	return errors.Join(errs...)
}

func newID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
