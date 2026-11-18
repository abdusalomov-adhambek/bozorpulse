package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bozorpulse/bozorpulse-ai/backend/internal/modules/planning/domain"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/platform/auth"
	"github.com/bozorpulse/bozorpulse-ai/backend/internal/shared"
)

var testNow = time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)

type fakeRepo struct {
	plans   map[string]domain.Plan
	getErr  error
	deletes []string
}

func newFakeRepo() *fakeRepo { return &fakeRepo{plans: map[string]domain.Plan{}} }

func (f *fakeRepo) Save(_ context.Context, plan domain.Plan) error {
	f.plans[plan.ID] = plan
	return nil
}
func (f *fakeRepo) Get(_ context.Context, owner, id string) (domain.Plan, error) {
	if f.getErr != nil {
		return domain.Plan{}, f.getErr
	}
	plan, ok := f.plans[id]
	if !ok {
		return domain.Plan{}, shared.ErrNotFound
	}
	if plan.CompanyID != owner {
		return domain.Plan{}, shared.ErrForbidden
	}
	return plan, nil
}
func (f *fakeRepo) List(_ context.Context, owner string) ([]domain.Plan, error) {
	out := []domain.Plan{}
	for _, plan := range f.plans {
		if plan.CompanyID == owner {
			out = append(out, plan)
		}
	}
	return out, nil
}
func (f *fakeRepo) Delete(_ context.Context, owner, id string) error {
	plan, err := f.Get(context.Background(), owner, id)
	if err != nil {
		return err
	}
	delete(f.plans, plan.ID)
	f.deletes = append(f.deletes, id)
	return nil
}

type fakeProducts struct {
	info     ProductInfo
	tier     float64
	err      error
	tierN    int
	productN int
}

func (f *fakeProducts) Product(_ context.Context, id string) (ProductInfo, error) {
	f.productN++
	if f.err != nil {
		return ProductInfo{}, f.err
	}
	info := f.info
	if info.ID == "" {
		info.ID = id
	}
	if info.Name == "" {
		info.Name = "Cement"
	}
	return info, nil
}
func (f *fakeProducts) BestTierPrice(_ context.Context, _ string, _ int) (float64, error) {
	f.tierN++
	return f.tier, f.err
}

type fakeForecaster struct {
	out   ForecastOutcome
	err   error
	calls int
}

func (f *fakeForecaster) Forecast(_ context.Context, _, _ string) (ForecastOutcome, error) {
	f.calls++
	if f.err != nil {
		return ForecastOutcome{}, f.err
	}
	return f.out, nil
}

func newTestService(repo PlanRepository, products ProductReader, fc Forecaster) *Service {
	s := New(repo, products, fc)
	s.now = func() time.Time { return testNow }
	return s
}

func customerPrincipal() auth.Principal {
	return auth.Principal{ID: "usr-1", CompanyID: "com-1", Role: auth.RoleCustomer, Status: auth.StatusConfirmed}
}

func goodForecast() ForecastOutcome {
	return ForecastOutcome{
		Predicted: 110, Low: 103, High: 117, Baseline: 100,
		TrendPct: 5, HorizonDays: 30, Samples: 12, DataStatus: domain.StatusDemo,
		Basis: "Based on 12 prices",
	}
}

func TestCreateUsesTierPriceNotForecastBase(t *testing.T) {
	products := &fakeProducts{info: ProductInfo{Name: "Cement", Unit: "bag", BasePrice: 100}, tier: 200}
	fc := &fakeForecaster{out: goodForecast()}
	svc := newTestService(newFakeRepo(), products, fc)

	plan, err := svc.Create(t.Context(), customerPrincipal(), PlanInput{
		Title: "Q4 cement", Items: []ItemInput{{ProductID: "prd-1", Quantity: 10}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(plan.Items))
	}
	proj := plan.Items[0].Projection
	if proj.Predicted == nil || *proj.Predicted != 220 {
		t.Fatalf("expected 220 (200 tier x 1.10), got %+v", proj.Predicted)
	}
	if proj.EstimatedBudget == nil || *proj.EstimatedBudget != 2200 {
		t.Fatalf("expected budget 2200, got %+v", proj.EstimatedBudget)
	}
	if proj.CurrentUnitPrice == nil || *proj.CurrentUnitPrice != 200 {
		t.Fatalf("tier price not stored: %+v", proj.CurrentUnitPrice)
	}
	if fc.calls != 1 || products.tierN != 1 {
		t.Fatalf("forecast/pricing calls: fc=%d tier=%d", fc.calls, products.tierN)
	}
}

func TestCreateFallsBackToBasePriceWhenNoTier(t *testing.T) {
	products := &fakeProducts{info: ProductInfo{Name: "Cement", Unit: "bag", BasePrice: 100}, tier: 0}
	fc := &fakeForecaster{out: goodForecast()}
	svc := newTestService(newFakeRepo(), products, fc)
	plan, err := svc.Create(t.Context(), customerPrincipal(), PlanInput{
		Title: "No tier", Items: []ItemInput{{ProductID: "prd-1", Quantity: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Items[0].Projection.CurrentUnitPrice; got == nil || *got != 100 {
		t.Fatalf("expected base price 100, got %+v", got)
	}
}

func TestOwnerFallbackWhenNoCompany(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeProducts{tier: 100}, &fakeForecaster{out: goodForecast()})
	p := auth.Principal{ID: "usr-9", Role: auth.RoleCustomer} // no company
	plan, err := svc.Create(t.Context(), p, PlanInput{
		Title: "Solo", Items: []ItemInput{{ProductID: "prd-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.CompanyID != "usr-9" {
		t.Fatalf("owner should fall back to user id, got %q", plan.CompanyID)
	}
	if _, err := svc.Get(t.Context(), p, plan.ID); err != nil {
		t.Fatalf("owner should be able to read their plan: %v", err)
	}
}

func TestRoleGuardAndCrossCompanyAccess(t *testing.T) {
	svc := newTestService(newFakeRepo(), &fakeProducts{tier: 100}, &fakeForecaster{out: goodForecast()})
	if _, err := svc.Create(t.Context(), auth.Principal{Role: auth.RoleSupplier}, PlanInput{Title: "x"}); err == nil {
		t.Fatal("supplier must not create plans")
	}

	owner := customerPrincipal()
	plan, err := svc.Create(t.Context(), owner, PlanInput{
		Title: "Mine", Items: []ItemInput{{ProductID: "prd-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	other := auth.Principal{ID: "usr-2", CompanyID: "com-2", Role: auth.RoleCustomer}
	if _, err := svc.Get(t.Context(), other, plan.ID); !errors.Is(err, shared.ErrForbidden) {
		t.Fatalf("cross-company read must be forbidden, got %v", err)
	}
	if err := svc.Delete(t.Context(), other, plan.ID); !errors.Is(err, shared.ErrForbidden) {
		t.Fatalf("cross-company delete must be forbidden, got %v", err)
	}
}

func TestValidation(t *testing.T) {
	svc := newTestService(newFakeRepo(), &fakeProducts{tier: 100}, &fakeForecaster{out: goodForecast()})
	cases := []PlanInput{
		{Title: "  ", Items: []ItemInput{{ProductID: "p", Quantity: 1}}},
		{Title: "ok"},
		{Title: "ok", Items: []ItemInput{{ProductID: "", Quantity: 1}}},
		{Title: "ok", Items: []ItemInput{{ProductID: "p", Quantity: 0}}},
		{Title: "ok", Items: []ItemInput{{ProductID: "p", Quantity: 10001}}},
		{Title: "ok", Items: []ItemInput{{ProductID: "p", Quantity: 1, TargetDate: "2027-01-01"}}},
		{Title: "ok", Status: "bogus", Items: []ItemInput{{ProductID: "p", Quantity: 1}}},
	}
	for i, in := range cases {
		if _, err := svc.Create(t.Context(), customerPrincipal(), in); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestTitleOnlyUpdateDoesNotCallForecaster(t *testing.T) {
	repo := newFakeRepo()
	products := &fakeProducts{info: ProductInfo{Name: "Cement", Unit: "bag", BasePrice: 100}, tier: 200}
	fc := &fakeForecaster{out: goodForecast()}
	svc := newTestService(repo, products, fc)
	plan, err := svc.Create(t.Context(), customerPrincipal(), PlanInput{
		Title: "Original", Items: []ItemInput{{ProductID: "prd-1", Quantity: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	before := fc.calls
	updated, err := svc.Update(t.Context(), customerPrincipal(), plan.ID, PlanInput{Title: "Renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Renamed" {
		t.Fatalf("title not updated: %q", updated.Title)
	}
	if fc.calls != before {
		t.Fatalf("title-only update must not forecast (calls %d -> %d)", before, fc.calls)
	}
	if products.tierN != 1 {
		t.Fatalf("title-only update must not re-resolve pricing, calls=%d", products.tierN)
	}
}

func TestUpdateQuantityRecomputesProjection(t *testing.T) {
	repo := newFakeRepo()
	fc := &fakeForecaster{out: goodForecast()}
	svc := newTestService(repo, &fakeProducts{info: ProductInfo{Name: "Cement", Unit: "bag"}, tier: 100}, fc)
	plan, err := svc.Create(t.Context(), customerPrincipal(), PlanInput{
		Title: "Plan", Items: []ItemInput{{ProductID: "prd-1", Quantity: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	oldBudget := *plan.Items[0].Projection.EstimatedBudget
	before := fc.calls
	updated, err := svc.Update(t.Context(), customerPrincipal(), plan.ID, PlanInput{
		Items: []ItemInput{{ProductID: "prd-1", Quantity: 4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fc.calls <= before {
		t.Fatal("quantity change must trigger a new forecast")
	}
	if got := *updated.Items[0].Projection.EstimatedBudget; got <= oldBudget {
		t.Fatalf("budget should scale with quantity: %f -> %f", oldBudget, got)
	}
}

func TestInsufficientForecastStillSavesNullProjection(t *testing.T) {
	fc := &fakeForecaster{out: ForecastOutcome{HorizonDays: 30, DataStatus: domain.StatusInsufficient}}
	svc := newTestService(newFakeRepo(), &fakeProducts{tier: 100}, fc)
	plan, err := svc.Create(t.Context(), customerPrincipal(), PlanInput{
		Title: "Thin data", Items: []ItemInput{{ProductID: "prd-1", Quantity: 3}},
	})
	if err != nil {
		t.Fatal(err)
	}
	proj := plan.Items[0].Projection
	if proj.Predicted != nil || proj.EstimatedBudget != nil {
		t.Fatalf("insufficient forecast must yield nulls, got %+v", proj)
	}
	if proj.Recommendation != domain.RecInsufficient {
		t.Fatalf("expected insufficient recommendation, got %s", proj.Recommendation)
	}
}

func TestForecastErrorFailsWholePlan(t *testing.T) {
	fc := &fakeForecaster{err: errors.New("forecast down")}
	svc := newTestService(newFakeRepo(), &fakeProducts{tier: 100}, fc)
	if _, err := svc.Create(t.Context(), customerPrincipal(), PlanInput{
		Title: "Broken", Items: []ItemInput{{ProductID: "prd-1", Quantity: 1}},
	}); err == nil {
		t.Fatal("a failed item must not be saved as a partial plan")
	}
}

func TestGetRestoresDemoDisclaimerFromDataStatus(t *testing.T) {
	repo := newFakeRepo()
	repo.plans["plan-stored"] = domain.Plan{
		ID: "plan-stored", CompanyID: "com-1",
		Items: []domain.PlanItem{{Projection: domain.PriceProjection{DataStatus: domain.StatusDemo}}},
	}
	svc := newTestService(repo, &fakeProducts{tier: 100}, &fakeForecaster{})
	got, err := svc.Get(t.Context(), customerPrincipal(), "plan-stored")
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0].Projection.Disclaimer != domain.DemoDisclaimer {
		t.Fatalf("demo disclaimer must be restored on read: %+v", got.Items[0].Projection)
	}
}

func TestPreviewDoesNotPersist(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo, &fakeProducts{tier: 100}, &fakeForecaster{out: goodForecast()})
	plan, err := svc.Preview(t.Context(), customerPrincipal(), PlanInput{
		Title: "What if", Items: []ItemInput{{ProductID: "prd-1", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.plans) != 0 {
		t.Fatal("preview must not persist")
	}
	if plan.ID != "preview" {
		t.Fatalf("unexpected preview id %q", plan.ID)
	}
}
