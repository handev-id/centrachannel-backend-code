package response

import (
	"encoding/json"
	"testing"
)

func TestSanitizeData(t *testing.T) {
	t.Run("nil slice becomes empty array", func(t *testing.T) {
		var items []int
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
		items := []int{1, 2}
		if got := sanitizeData(items); len(got.([]int)) != 2 {
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
	var convs []*struct{ ID int }
	b, err := json.Marshal(Response{
		Meta: ResponseMeta{Message: "success", LastID: intPtr(0), HasMore: boolPtr(false)},
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
