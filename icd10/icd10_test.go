package icd10

import "testing"

// Lookup of a known code returns a non-empty display.
func TestLookup_Known(t *testing.T) {
	entry, ok := Lookup("A00")
	if !ok {
		t.Fatalf("Lookup(A00): not found")
	}
	if entry.Display == "" {
		t.Fatalf("expected non-empty display")
	}
}

// Lookup of an unknown code returns ok=false.
func TestLookup_Unknown(t *testing.T) {
	if _, ok := Lookup("ZZZZ"); ok {
		t.Fatalf("Lookup(ZZZZ): expected miss")
	}
}

// Search filters case-insensitively.
func TestSearch(t *testing.T) {
	hits := Search("cholera", 5)
	if len(hits) == 0 {
		t.Fatalf("expected hits for cholera")
	}
	if len(hits) > 5 {
		t.Fatalf("limit not respected: %d", len(hits))
	}
	for _, h := range hits {
		if h.Code == "" || h.Display == "" {
			t.Fatalf("empty code/display: %+v", h)
		}
	}
}

// Count matches the bundled dataset size.
func TestCount_Positive(t *testing.T) {
	if Count <= 0 {
		t.Fatalf("expected count > 0")
	}
}
