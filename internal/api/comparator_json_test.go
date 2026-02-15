package api_test

import (
	"encoding/json"
	"testing"

	"github.com/duboisf/linear/internal/api"
)

func TestStringComparator_MarshalJSON_OmitsNilFields(t *testing.T) {
	t.Parallel()

	sc := api.StringComparator{
		Nin: []string{"completed", "canceled"},
	}

	data, err := json.Marshal(sc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := raw["in"]; ok {
		t.Error("expected 'in' field to be omitted, but it was present")
	}
	if _, ok := raw["eq"]; ok {
		t.Error("expected 'eq' field to be omitted, but it was present")
	}
	nin, ok := raw["nin"]
	if !ok {
		t.Fatal("expected 'nin' field to be present")
	}
	arr, ok := nin.([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("expected nin to be array of 2, got %v", nin)
	}
}

func TestStringComparator_MarshalJSON_EqIgnoreCase(t *testing.T) {
	t.Parallel()

	name := "alice"
	sc := api.StringComparator{
		EqIgnoreCase: &name,
	}

	data, err := json.Marshal(sc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(raw) != 1 {
		t.Errorf("expected exactly 1 field, got %d: %v", len(raw), raw)
	}
	if raw["eqIgnoreCase"] != "alice" {
		t.Errorf("expected eqIgnoreCase=alice, got %v", raw["eqIgnoreCase"])
	}
}

func TestBooleanComparator_MarshalJSON_OmitsNilFields(t *testing.T) {
	t.Parallel()

	trueVal := true
	bc := api.BooleanComparator{
		Eq: &trueVal,
	}

	data, err := json.Marshal(bc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(raw) != 1 {
		t.Errorf("expected exactly 1 field, got %d: %v", len(raw), raw)
	}
	if _, ok := raw["neq"]; ok {
		t.Error("expected 'neq' field to be omitted, but it was present")
	}
	if raw["eq"] != true {
		t.Errorf("expected eq=true, got %v", raw["eq"])
	}
}

func TestNumberComparator_MarshalJSON_OmitsNilFields(t *testing.T) {
	t.Parallel()

	val := float64(11)
	nc := api.NumberComparator{
		Eq: &val,
	}

	data, err := json.Marshal(nc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(raw) != 1 {
		t.Errorf("expected exactly 1 field, got %d: %v", len(raw), raw)
	}
	if _, ok := raw["in"]; ok {
		t.Error("expected 'in' field to be omitted")
	}
}
