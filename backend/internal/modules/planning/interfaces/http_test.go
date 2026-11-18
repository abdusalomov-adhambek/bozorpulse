package interfaces

import "testing"

func TestPlanRequestInputPreservesNilItems(t *testing.T) {
	in := planRequest{Title: "Rename only"}
	got := in.input()
	if got.Items != nil {
		t.Fatalf("omitted items must stay nil so PATCH does not try to rebuild items: %#v", got.Items)
	}
}

func TestPlanRequestInputMapsItems(t *testing.T) {
	in := planRequest{Title: "Plan", Items: []itemRequest{{ProductID: "prd-1", Quantity: 5, TargetDate: "2026-12-01"}}}
	got := in.input()
	if len(got.Items) != 1 || got.Items[0].ProductID != "prd-1" || got.Items[0].Quantity != 5 {
		t.Fatalf("item mapping failed: %+v", got.Items)
	}
}
