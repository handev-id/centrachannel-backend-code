package response

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSanitizeData(t *testing.T) {
	t.Run("nil slice becomes empty array", func(t *testing.T) {
		var items []uuid.UUID
		got := sanitizeData(items)
		if got == nil {
			t.Fatal("expected non-nil slice, got nil")
		}
		if b, _ := json.Marshal(got); string(b) != "[]" {
			t.Fatalf("expected [] in json, got %s", b)
		}
	})

	t.Run("nil map becomes empty object", func(t *testing.T) {
		var m map[string]int
		got := sanitizeData(m)
		if got == nil {
			t.Fatal("expected non-nil map, got nil")
		}
		if b, _ := json.Marshal(got); string(b) != "{}" {
			t.Fatalf("expected {} in json, got %s", b)
		}
	})

	t.Run("explicit nil stays null", func(t *testing.T) {
		if got := sanitizeData(nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("non-nil data untouched", func(t *testing.T) {
		items := []uuid.UUID{uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000002")}
		if got := sanitizeData(items); len(got.([]uuid.UUID)) != 2 {
			t.Fatalf("expected original slice, got %v", got)
		}
	})

	t.Run("nil []byte stays nil (serializes as JSON string)", func(t *testing.T) {
		var raw []byte
		if got := sanitizeData(raw); got.([]byte) != nil {
			t.Fatalf("expected nil []byte untouched, got %v", got)
		}
	})
}

func TestCursorPaginatedEmptySliceMarshalsAsArray(t *testing.T) {
	// Regression: GET /conversations with zero rows used to emit data: null.
	var convs []*struct{ ID uuid.UUID }
	b, err := json.Marshal(Response{
		Meta: ResponseMeta{Message: "success", HasMore: boolPtr(false)},
		Data: sanitizeData(convs),
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if string(out.Data) != "[]" {
		t.Fatalf("expected data: [], got data: %s", out.Data)
	}
}

func intPtr(i int) *int    { return &i }
func boolPtr(b bool) *bool { return &b }

func TestValidateUUIDFields(t *testing.T) {
	type request struct {
		ID       uuid.UUID   `validate:"required,uuid"`
		Optional *uuid.UUID  `validate:"omitnil,required,uuid"`
		Roles    []uuid.UUID `validate:"required,min=1,dive,required,uuid"`
	}
	valid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	zero := uuid.Nil
	for _, tc := range []struct {
		name      string
		body      request
		wantError bool
	}{
		{"valid with optional omitted", request{ID: valid, Roles: []uuid.UUID{valid}}, false},
		{"valid optional", request{ID: valid, Optional: &valid, Roles: []uuid.UUID{valid}}, false},
		{"nil required UUID", request{Roles: []uuid.UUID{valid}}, true},
		{"explicit nil optional UUID", request{ID: valid, Optional: &zero, Roles: []uuid.UUID{valid}}, true},
		{"nil role UUID", request{ID: valid, Roles: []uuid.UUID{zero}}, true},
		{"empty roles", request{ID: valid, Roles: []uuid.UUID{}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate(&tc.body); (err != nil) != tc.wantError {
				t.Fatalf("validation error=%v, want error=%v", err, tc.wantError)
			}
		})
	}
}
