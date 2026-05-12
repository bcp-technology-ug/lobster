package convert

import (
	"encoding/base64"
	"testing"
)

func TestEncodeCursor_DecodeCursor_roundTrip(t *testing.T) {
	t.Parallel()
	createdAt := "2024-01-01T00:00:00Z"
	id := "run-abc-123"
	token := EncodeCursor(createdAt, id)
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	gotCA, gotID, err := DecodeCursor(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCA == nil || *gotCA != createdAt {
		t.Errorf("createdAt: got %v want %q", gotCA, createdAt)
	}
	if gotID == nil || *gotID != id {
		t.Errorf("id: got %v want %q", gotID, id)
	}
}

func TestDecodeCursor_empty(t *testing.T) {
	t.Parallel()
	ca, id, err := DecodeCursor("")
	if err != nil {
		t.Fatalf("empty token should not error, got: %v", err)
	}
	if ca != nil || id != nil {
		t.Error("empty token should return nil cursors")
	}
}

func TestDecodeCursor_invalidBase64(t *testing.T) {
	t.Parallel()
	_, _, err := DecodeCursor("!!!not-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestDecodeCursor_invalidJSON(t *testing.T) {
	t.Parallel()
	raw := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
	_, _, err := DecodeCursor(raw)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestDecodeCursor_missingFields(t *testing.T) {
	t.Parallel()
	// valid base64 JSON but missing required fields
	raw := base64.RawURLEncoding.EncodeToString([]byte(`{"c":"","i":""}`))
	_, _, err := DecodeCursor(raw)
	if err == nil {
		t.Error("expected error when c and i are empty, got nil")
	}
}

func TestPageSizeOrDefault_zero(t *testing.T) {
	t.Parallel()
	if got := PageSizeOrDefault(0); got != DefaultPageSize {
		t.Errorf("got %d want %d", got, DefaultPageSize)
	}
}

func TestPageSizeOrDefault_normal(t *testing.T) {
	t.Parallel()
	if got := PageSizeOrDefault(10); got != 10 {
		t.Errorf("got %d want 10", got)
	}
}

func TestPageSizeOrDefault_clampHigh(t *testing.T) {
	t.Parallel()
	if got := PageSizeOrDefault(9999); got != 100 {
		t.Errorf("got %d want 100 (clamped)", got)
	}
}

func TestPageSizeOrDefault_boundary(t *testing.T) {
	t.Parallel()
	if got := PageSizeOrDefault(100); got != 100 {
		t.Errorf("got %d want 100", got)
	}
	if got := PageSizeOrDefault(101); got != 100 {
		t.Errorf("got %d want 100 (clamped)", got)
	}
}
